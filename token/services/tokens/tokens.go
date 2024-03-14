/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens

import (
	"runtime/debug"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var logger = flogging.MustGetLogger("token-sdk.tokens")

// Authorization is an interface that defines method to check the relation between a token or TMS
// and wallets (owner, auditor, etc.)
type Authorization interface {
	// IsMine returns true if the passed token is owned by an owner wallet in the passed TMS
	IsMine(tms *token.ManagementService, tok *token2.Token) ([]string, bool)
	// AmIAnAuditor return true if the passed TMS contains an auditor wallet for any of the auditor identities
	// defined in the public parameters of the passed TMS.
	AmIAnAuditor(tms *token.ManagementService) bool
}

type Issued interface {
	// Issued returns true if the passed issuer issued the passed token
	Issued(tms *token.ManagementService, issuer view.Identity, tok *token2.Token) bool
}

type GetTMSProviderFunc = func() *token.ManagementServiceProvider

// Transaction models a token transaction
type Transaction interface {
	ID() string
	Network() string
	Channel() string
	Namespace() string
	Request() *token.Request
}

// Tokens is the interface for the owner service
type Tokens struct {
	TMSProvider TMSProvider
	Ownership   Authorization
	Issued      Issued
	Storage     *DBStorage
}

// AppendTransaction appends the content of the passed transaction to the token db.
// If the transaction is already in there, nothing more happens.
// The operation is atomic.
func (t *Tokens) AppendTransaction(tx Transaction) (err error) {
	txID := tx.ID()
	tms, err := t.TMSProvider.GetManagementService(
		token.WithNetwork(tx.Network()),
		token.WithChannel(tx.Channel()),
		token.WithNamespace(tx.Namespace()),
	)
	if err != nil {
		return errors.WithMessagef(err, "failed getting token management service [%s:%s:%s]", tx.Network(), tx.Channel(), tx.Namespace())
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("transaction [%s on (%s)] is known, extract tokens", txID, tms.ID())
	}
	request := tx.Request()
	if request == nil {
		logger.Debugf("transaction [%s], no request found, skip it", txID)
		return nil
	}
	if request.Metadata == nil {
		logger.Debugf("transaction [%s], no metadata found, skip it", txID)
		return nil
	}

	ts, err := t.Storage.NewTransaction()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil && ts != nil {
			if err := ts.Rollback(); err != nil {
				logger.Errorf("failed to rollback [%s][%s]", err, debug.Stack())
			}
		}
	}()
	exists, err := ts.TransactionExists(tx.ID())
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	precision := tms.PublicParametersManager().PublicParameters().Precision()
	md, err := request.GetMetadata()
	if err != nil {
		logger.Debugf("transaction [%s], failed to get metadata [%s]", txID, err)
		return err
	}
	if tms.PublicParametersManager().PublicParameters().GraphHiding() {
		ids := md.SpentTokenID()
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s] with graph hiding, delete inputs [%v]", txID, ids)
		}
		if err := ts.DeleteTokens(txID, ids); err != nil {
			return err
		}
	}

	auditorFlag := t.Ownership.AmIAnAuditor(tms)
	if auditorFlag {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s], I must be the auditor", txID)
		}
	}

	is, os, err := request.InputsAndOutputs()
	if err != nil {
		return errors.WithMessagef(err, "failed to get request's outputs")
	}

	// parse the inputs
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("parse [%d] inputs and [%d] outputs from [%s]", is.Count(), os.Count(), tx.ID())
	}
	for _, input := range is.Inputs() {
		if input.Id == nil {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("transaction [%s] found an input that is not mine, skip it", tx.ID())
			}
			continue
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s] delete input [%s]", tx.ID(), input.Id)
		}
		if err = ts.DeleteToken(input.Id.TxId, input.Id.Index, tx.ID()); err != nil {
			return err
		}
		continue
	}

	// parse the outputs
	for _, output := range os.Outputs() {
		// get token in the clear
		tok, issuer, tokenOnLedgerMetadata, err2 := md.GetToken(output.LedgerOutput)
		if err2 != nil {
			logger.Errorf("transaction [%s], found a token but failed getting the clear version, skipping it [%s]", txID, err2)
			continue
		}
		if tok == nil {
			logger.Warnf("failed getting token in the clear for [%s, %s]", output.ID(txID), string(output.LedgerOutput))
			continue
		}

		if len(output.LedgerOutput) == 0 {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("transaction [%s] without graph hiding, delete input [%d]", txID, output.Index)
			}
			if err = ts.DeleteToken(tx.ID(), output.Index, tx.ID()); err != nil {
				return err
			}
			continue
		}

		issuerFlag := !issuer.IsNone() && t.Issued.Issued(tms, issuer, tok)
		ids, mine := t.Ownership.IsMine(tms, tok)
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			if mine {
				logger.Debugf("transaction [%s], found a token and it is mine", txID)
			} else {
				logger.Debugf("transaction [%s], found a token and it is NOT mine", txID)
			}
			if issuerFlag {
				logger.Debugf("transaction [%s], found a token and I have issued it", txID)
			}
			logger.Debugf("store token [%s:%d][%s]", txID, output.Index, hash.Hashable(output.LedgerOutput))
		}

		if !mine && !auditorFlag && !issuerFlag {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("transaction [%s], discarding token, not mine, not an auditor, not an issuer", txID)
			}
			continue
		}

		if err = ts.AppendToken(
			txID,
			output.Index,
			tok,
			output.LedgerOutput,
			tokenOnLedgerMetadata,
			ids,
			issuer,
			precision,
			Flags{
				Mine:    mine,
				Auditor: auditorFlag,
				Issuer:  issuerFlag,
			},
		); err != nil {
			return err
		}

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("Done parsing write key [%s]", output.ID(txID))
		}
	}

	if err = ts.Commit(); err != nil {
		return err
	}
	return nil
}

// StorePublicParams stores the passed public parameters in the token db
func (t *Tokens) StorePublicParams(raw []byte) error {
	return t.Storage.StorePublicParams(raw)
}

// DeleteToken marks the entries corresponding to the passed token ids as deleted.
// The deletion is attributed to the passed deletedBy argument.
func (t *Tokens) DeleteToken(deletedBy string, ids ...*token2.ID) (err error) {
	ts, err := t.Storage.NewTransaction()
	if err != nil {
		return err
	}
	defer func() {
		if err := ts.Rollback(); err != nil {
			logger.Errorf("failed to rollback [%s]", err)
		}
	}()

	for _, id := range ids {
		if id != nil {
			if err = ts.DeleteToken(id.TxId, id.Index, deletedBy); err != nil {
				return errors.WithMessagef(err, "failed to delete [%s] by [%s]", id, deletedBy)
			}
		}
	}
	if err = ts.Commit(); err != nil {
		return err
	}
	return nil
}
