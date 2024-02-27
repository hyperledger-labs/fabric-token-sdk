/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
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
	TokenStore  TokenStore
}

func (t *Tokens) AppendTransaction(tx Transaction) error {
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
	if request.Metadata == nil {
		logger.Debugf("transaction [%s], no metadata found, skip it", txID)
		return nil
	}
	metadata, err := request.GetMetadata()
	if err != nil {
		logger.Debugf("transaction [%s], failed to get metadata [%s]", txID, err)
		return err
	}

	precision := tms.PublicParametersManager().PublicParameters().Precision()
	if tms.PublicParametersManager().PublicParameters().GraphHiding() {
		ids := metadata.SpentTokenID()
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s] with graph hiding, delete inputs [%v]", txID, ids)
		}
		for _, id := range ids {
			if err := t.TokenStore.DeleteToken(id.TxId, id.Index, txID); err != nil {
				return err
			}
		}
	}

	auditorFlag := t.Ownership.AmIAnAuditor(tms)
	if auditorFlag {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s], I must be the auditor", txID)
		}
	}

	// parse the outputs
	os, err := request.Outputs()
	if err != nil {
		return errors.WithMessagef(err, "failed to get request's outputs")
	}

	for _, output := range os.Outputs() {
		// get token in the clear
		tok, issuer, tokenOnLedgerMetadata, err := metadata.GetToken(output.LedgerOutput)
		if err != nil {
			logger.Errorf("transaction [%s], found a token but failed getting the clear version, skipping it [%s]", txID, err)
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
			if err := t.TokenStore.DeleteToken(tx.ID(), output.Index, tx.ID()); err != nil {
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
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			if issuerFlag {
				logger.Debugf("transaction [%s], found a token and I have issued it", txID)
			}
		}

		if err := t.TokenStore.AppendToken(
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

	return nil
}
