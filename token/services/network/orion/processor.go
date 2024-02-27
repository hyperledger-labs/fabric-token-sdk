/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"strconv"
	"strings"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"

	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/rws/keys"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type ONS interface {
	Name() string
	MetadataService() *orion.MetadataService
}

type GetTMSProviderFunc = func() *token.ManagementServiceProvider
type GetTokenRequestFunc = func(tms *token.ManagementService, txID string) ([]byte, error)

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

type RWSetProcessor struct {
	network         string
	nss             []string
	GetTMSProvider  GetTMSProviderFunc
	GetTokenRequest GetTokenRequestFunc
	ownership       Authorization
	issued          Issued
	tokenStore      tokens.TokenStore
}

func NewTokenRWSetProcessor(network string, ns string, GetTMSProvider GetTMSProviderFunc, GetTokenRequest GetTokenRequestFunc, ownership Authorization, issued Issued, tokenStore tokens.TokenStore) *RWSetProcessor {
	return &RWSetProcessor{
		network:         network,
		nss:             []string{ns},
		GetTMSProvider:  GetTMSProvider,
		GetTokenRequest: GetTokenRequest,
		ownership:       ownership,
		issued:          issued,
		tokenStore:      tokenStore,
	}
}

func (r *RWSetProcessor) Process(req orion.Request, tx orion.ProcessTransaction, rws *orion.RWSet, ns string) error {
	found := false
	for _, ans := range r.nss {
		if ns == ans {
			found = true
			break
		}
	}
	if !found {
		logger.Debugf("this processor cannot parse namespace [%s]", ns)
		return errors.Errorf("this processor cannot parse namespace [%s]", ns)
	}

	// Match the network name
	if tx.Network() != r.network {
		logger.Debugf("tx's network [%s]!=[%s]", tx.Network(), r.network)
		return nil
	}

	return r.tokenRequest(req, tx, rws, ns)
}

func (r *RWSetProcessor) tokenRequest(req orion.Request, tx orion.ProcessTransaction, rws *orion.RWSet, ns string) error {
	txID := tx.ID()

	tms, err := r.GetTMSProvider().GetManagementService(
		token.WithNetwork(tx.Network()),
		token.WithNamespace(ns),
	)
	if err != nil {
		return errors.WithMessagef(err, "failed getting token management service [%s:%s]", tx.Network(), ns)
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("transaction [%s on (%s)] is known, extract tokens", txID, tms.ID())
	}
	trRaw, err := r.GetTokenRequest(tms, txID)
	if err != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s], failed getting token request [%s]", txID, err)
		}
		return errors.WithMessagef(err, "failed to get token request for [%s]", txID)
	}
	if len(trRaw) == 0 {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s], no token request found, skip it", txID)
		}
		return nil
	}
	request, err := tms.NewFullRequestFromBytes(trRaw)
	if err != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s], failed getting zkat state from transient map [%s]", txID, err)
		}
		return err
	}
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
			if err := r.tokenStore.DeleteToken(id.TxId, id.Index, tx.ID()); err != nil {
				return err
			}
		}
	}

	auditorFlag := r.ownership.AmIAnAuditor(tms)
	if auditorFlag {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s], I must be the auditor", txID)
		}
	}

	for i := 0; i < rws.NumWrites(ns); i++ {
		key, tokenOnLedger, err := rws.GetWriteAt(ns, i)
		if err != nil {
			return err
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("Parsing write key [%s]", key)
		}
		prefix, components, err := keys.SplitCompositeKey(strings.ReplaceAll(key, "~", string(rune(0))))
		if err != nil {
			return errors.WithMessagef(err, "failed to split key [%s]", key)
		}
		if prefix != keys.TokenKeyPrefix {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("expected prefix [%s], got [%s], skipping", keys.TokenKeyPrefix, prefix)
			}
			continue
		}
		switch components[0] {
		case keys.TokenMineKeyPrefix:
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("expected key without the mine prefix, skipping")
			}
			continue
		case keys.TokenRequestKeyPrefix:
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("expected key without the token request prefix, skipping")
			}
			continue
		case keys.SerialNumber:
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("expected key without the serial number prefix, skipping")
			}
			continue
		case keys.IssueActionMetadata:
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("expected key without the issue action metadata, skipping")
			}
			continue
		case keys.TransferActionMetadata:
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("expected key without the transfer action metadata, skipping")
			}
			continue
		}

		index, err := strconv.ParseUint(components[1], 10, 64)
		if err != nil {
			logger.Errorf("invalid output index for key [%s]", key)
			return errors.Wrapf(err, "invalid output index for key [%s]", key)
		}

		// This is a delete op, delete it from the token store
		if len(tokenOnLedger) == 0 {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("transaction [%s] without graph hiding, delete input [%s:%d]", txID, components[0], index)
			}
			if err := r.tokenStore.DeleteToken(components[0], index, tx.ID()); err != nil {
				return err
			}
			continue
		}

		if components[0] != txID {
			logger.Errorf("invalid output, must refer to tx id [%s], got [%s]", txID, components[0])
			return errors.Errorf("invalid output, must refer to tx id [%s], got [%s]", txID, components[0])
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s], found a token...", txID)
		}

		// get token in the clear
		tok, issuer, tokenOnLedgerMetadata, err := metadata.GetToken(tokenOnLedger)
		if err != nil {
			logger.Errorf("transaction [%s], found a token but failed getting the clear version, skipping it [%s]", txID, err)
			continue
		}
		if tok == nil {
			logger.Warnf("failed getting token in the clear for key [%s, %s]", key, string(tokenOnLedger))
			continue
		}

		issuerFlag := !issuer.IsNone() && r.issued.Issued(tms, issuer, tok)
		ids, mine := r.ownership.IsMine(tms, tok)
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

		if err := r.tokenStore.AppendToken(
			txID,
			index,
			tok,
			tokenOnLedger,
			tokenOnLedgerMetadata,
			ids,
			issuer,
			precision,
			tokens.Flags{
				Mine:    mine,
				Auditor: auditorFlag,
				Issuer:  issuerFlag,
			},
		); err != nil {
			return err
		}

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("Done parsing write key [%s]", key)
		}
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("transaction [%s] is known, extract tokens, done!", txID)
	}

	return nil
}
