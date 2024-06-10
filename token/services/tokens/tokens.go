/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens

import (
	"runtime/debug"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var logger = logging.MustGetLogger("token-sdk.tokens")

// Authorization is an interface that defines method to check the relation between a token or TMS
// and wallets (owner, auditor, etc.)
type Authorization interface {
	// IsMine returns true if the passed token is owned by an owner wallet in the passed TMS
	IsMine(tms *token.ManagementService, tok *token2.Token) ([]string, bool)
	// AmIAnAuditor returns true if the passed TMS contains an auditor wallet for any of the auditor identities
	// defined in the public parameters of the passed TMS.
	AmIAnAuditor(tms *token.ManagementService) bool
	// OwnerType returns the type of owner (e.g. 'idemix' or 'htlc') and the identity bytes
	OwnerType(raw []byte) (string, []byte, error)
}

type Issued interface {
	// Issued returns true if the passed issuer issued the passed token
	Issued(tms *token.ManagementService, issuer token.Identity, tok *token2.Token) bool
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

// Tokens is the interface for the token service
type Tokens struct {
	TMSProvider TMSProvider
	Ownership   Authorization
	Issued      Issued
	Storage     *DBStorage
}

func (t *Tokens) Append(tmsID token.TMSID, txID string, request *token.Request) (err error) {
	tms, err := t.TMSProvider.GetManagementService(token.WithTMSID(tmsID))
	if err != nil {
		return errors.WithMessagef(err, "failed getting token management service [%s]", tmsID)
	}

	logger.Debugf("transaction [%s on (%s)] is known, extract tokens", txID, tms.ID())
	if request == nil {
		logger.Debugf("transaction [%s], no request found, skip it", txID)
		return nil
	}
	if request.Metadata == nil {
		logger.Debugf("transaction [%s], no metadata found, skip it", txID)
		return nil
	}

	exists, err := t.Storage.TransactionExists(txID)
	if err != nil {
		logger.Errorf("transaction [%s], failed to check existence in db [%s]", txID, err)
		return errors.WithMessagef(err, "transaction [%s], failed to check existence in db", txID)
	}
	if exists {
		logger.Debugf("transaction [%s], exists in db, skipping", txID)
		return nil
	}

	graphHiding := tms.PublicParametersManager().PublicParameters().GraphHiding()
	precision := tms.PublicParametersManager().PublicParameters().Precision()
	auditorFlag := t.Ownership.AmIAnAuditor(tms)
	if auditorFlag {
		logger.Debugf("transaction [%s], I must be the auditor", txID)
	}

	md, err := request.GetMetadata()
	if err != nil {
		logger.Debugf("transaction [%s], failed to get metadata [%s]", txID, err)
		return errors.WithMessagef(err, "transaction [%s], failed to get request metadata", txID)
	}

	is, os, err := request.InputsAndOutputs()
	if err != nil {
		return errors.WithMessagef(err, "failed to get request's outputs")
	}
	toSpend, toAppend := t.parse(tms, txID, md, is, os, auditorFlag, precision, graphHiding)

	logger.Debugf("transaction [%s] start db transaction", txID)
	ts, err := t.Storage.NewTransaction()
	if err != nil {
		return errors.WithMessagef(err, "transaction [%s], failed to start db transaction", txID)
	}
	defer func() {
		if err == nil {
			return
		}
		if err1 := ts.Rollback(); err1 != nil {
			logger.Errorf("error rolling back [%s][%s]", err1, debug.Stack())
		} else {
			logger.Infof("transaction [%s] rolled back", txID)
		}
	}()
	for _, tta := range toAppend {
		err = ts.AppendToken(tta)
		if err != nil {
			return errors.WithMessagef(err, "transaction [%s], failed to append token", txID)
		}
	}
	err = ts.DeleteTokens(txID, toSpend)
	if err != nil {
		return errors.WithMessagef(err, "transaction [%s], failed to delete tokens", txID)
	}
	if err = ts.Commit(); err != nil {
		return errors.WithMessagef(err, "transaction [%s], failed to commit tokens to database", txID)
	}
	logger.Debugf("transaction [%s], committed tokens to database", txID)

	return nil
}

type MetaData interface {
	SpentTokenID() []*token2.ID
	GetToken(raw []byte) (*token2.Token, token.Identity, []byte, error)
}

// parse returns the tokens to store and spend as the result of a transaction
func (t *Tokens) parse(tms *token.ManagementService, txID string, md MetaData, is *token.InputStream, os *token.OutputStream, auditorFlag bool, precision uint64, graphHiding bool) (toSpend []*token2.ID, toAppend []TokenToAppend) {
	if graphHiding {
		ids := md.SpentTokenID()
		logger.Debugf("transaction [%s] with graph hiding, delete inputs [%v]", txID, ids)
		toSpend = append(toSpend, ids...)
	}

	logger.Debugf("parse [%d] inputs and [%d] outputs from [%s]", is.Count(), os.Count(), txID)

	// parse the inputs
	for _, input := range is.Inputs() {
		if input.Id == nil {
			logger.Debugf("transaction [%s] found an input that is not mine, skip it", txID)
			continue
		}
		logger.Debugf("transaction [%s] delete input [%s]", txID, input.Id)
		toSpend = append(toSpend, input.Id)
	}

	// parse the outputs
	for _, output := range os.Outputs() {
		// get token in the clear
		tok, issuer, tokenOnLedgerMetadata, err := md.GetToken(output.LedgerOutput)
		if err != nil {
			logger.Errorf("transaction [%s], found a token but failed getting the clear version, skipping it [%s]", txID, err)
			continue
		}
		if tok == nil {
			logger.Warnf("failed getting token in the clear for [%s, %s]", output.ID(txID), string(output.LedgerOutput))
			continue
		}

		if len(output.LedgerOutput) == 0 {
			logger.Debugf("transaction [%s] without graph hiding, delete input [%d]", txID, output.Index)
			toSpend = append(toSpend, &token2.ID{TxId: txID, Index: output.Index})
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
			logger.Debugf("transaction [%s], discarding token, not mine, not an auditor, not an issuer", txID)
			continue
		}

		tta := TokenToAppend{
			txID:                  txID,
			index:                 output.Index,
			tok:                   tok,
			tokenOnLedger:         output.LedgerOutput,
			tokenOnLedgerMetadata: tokenOnLedgerMetadata,
			owners:                ids,
			issuer:                issuer,
			precision:             precision,
			flags: Flags{
				Mine:    mine,
				Auditor: auditorFlag,
				Issuer:  issuerFlag,
			},
		}
		toAppend = append(toAppend, tta)

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("done parsing write key [%s]", output.ID(txID))
		}
	}
	return
}

func (t *Tokens) AppendRaw(tmsID token.TMSID, txID string, requestRaw []byte) (err error) {
	logger.Debugf("get tms for [%s]", txID)
	tms, err := t.TMSProvider.GetManagementService(token.WithTMSID(tmsID))
	if err != nil {
		return errors.WithMessagef(err, "failed getting token management service [%s]", tmsID)
	}
	logger.Debugf("get tms for [%s], done", txID)
	tr, err := tms.NewFullRequestFromBytes(requestRaw)
	if err != nil {
		return errors.WithMessagef(err, "failed unmarshal token request [%s]", txID)
	}
	logger.Debugf("append token request for [%s]", txID)
	return t.Append(tmsID, txID, tr)
}

// AppendTransaction appends the content of the passed transaction to the token db.
// If the transaction is already in there, nothing more happens.
// The operation is atomic.
func (t *Tokens) AppendTransaction(tx Transaction) (err error) {
	return t.Append(
		token.TMSID{
			Network:   tx.Network(),
			Channel:   tx.Channel(),
			Namespace: tx.Namespace(),
		},
		tx.ID(),
		tx.Request(),
	)
}

// StorePublicParams stores the passed public parameters in the token db
func (t *Tokens) StorePublicParams(raw []byte) error {
	return t.Storage.StorePublicParams(raw)
}

// DeleteToken marks the entries corresponding to the passed token ids as deleted.
// The deletion is attributed to the passed deletedBy argument.
func (t *Tokens) DeleteToken(deletedBy string, ids ...*token2.ID) (err error) {
	return t.Storage.tokenDB.DeleteTokens(deletedBy, ids...)
}
