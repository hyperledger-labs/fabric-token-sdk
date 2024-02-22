/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"strconv"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/processor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/rws/keys"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var logger = flogging.MustGetLogger("token-sdk.vault.processor")

type GetTMSProviderFunc = func() *token.ManagementServiceProvider
type GetTokenRequestFunc = func(tms *token.ManagementService, txID string) ([]byte, error)

type RWSetProcessor struct {
	network         string
	nss             []string
	GetTMSProvider  GetTMSProviderFunc
	GetTokenRequest GetTokenRequestFunc
	ownership       network.Authorization
	issued          network.Issued
	tokenStore      processor.TokenStore
}

func NewTokenRWSetProcessor(network string, ns string, GetTMSProvider GetTMSProviderFunc, GetTokenRequest GetTokenRequestFunc, ownership network.Authorization, issued network.Issued, tokenStore processor.TokenStore) *RWSetProcessor {
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

func (r *RWSetProcessor) Process(req fabric.Request, tx fabric.ProcessTransaction, rws *fabric.RWSet, ns string) error {
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

	fn, _ := tx.FunctionAndParameters()
	logger.Debugf("process namespace and function [%s:%s]", ns, fn)
	switch fn {
	case "init":
		return r.init(tx, rws, ns)
	default:
		return r.tokenRequest(req, tx, rws, ns)
	}
}

// init when invoked extracts the public params from rwset and updates the local version
func (r *RWSetProcessor) init(tx fabric.ProcessTransaction, rws *fabric.RWSet, ns string) error {
	tsmProvider := r.GetTMSProvider()
	setUpKey, err := keys.CreateSetupKey()
	if err != nil {
		return errors.Errorf("failed creating setup key")
	}
	for i := 0; i < rws.NumWrites(ns); i++ {
		key, val, err := rws.GetWriteAt(ns, i)
		if err != nil {
			return err
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("Parsing write key [%s]", key)
		}
		if key == setUpKey {
			if err := tsmProvider.Update(token.TMSID{
				Network:   tx.Network(),
				Channel:   tx.Channel(),
				Namespace: ns,
			}, val); err != nil {
				return errors.Wrapf(err, "failed updating public params")
			}
			if err := r.tokenStore.StorePublicParams(val); err != nil {
				return errors.Wrapf(err, "failed storing public params")
			}
			break
		}
	}
	return nil
}

func (r *RWSetProcessor) tokenRequest(req fabric.Request, tx fabric.ProcessTransaction, rws *fabric.RWSet, ns string) error {
	txID := tx.ID()

	tms, err := r.GetTMSProvider().GetManagementService(
		token.WithNetwork(tx.Network()),
		token.WithChannel(tx.Channel()),
		token.WithNamespace(ns),
	)
	if err != nil {
		return errors.WithMessagef(err, "failed getting token management service [%s:%s:%s]", tx.Network(), tx.Channel(), ns)
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

	for i := 0; i < rws.NumWrites(ns); i++ {
		key, tokenOnLedger, err := rws.GetWriteAt(ns, i)
		if err != nil {
			return err
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("Parsing write key [%s]", key)
		}
		prefix, components, err := keys.SplitCompositeKey(key)
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

		ids, mine := r.ownership.IsMine(tms, tok)
		if mine {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("transaction [%s], found a token and it is mine", txID)
			}
			if err := r.tokenStore.StoreToken(txID, index, tok, tokenOnLedger, tokenOnLedgerMetadata, ids, precision); err != nil {
				return err
			}
		} else {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("transaction [%s], found a token and it is NOT mine", txID)
			}
		}

		// if I'm an auditor, store the audit entry
		if r.ownership.AmIAnAuditor(tms) {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("transaction [%s], found a token and I must be the auditor", txID)
			}
			if err := r.tokenStore.StoreAuditToken(txID, index, tok, tokenOnLedger, tokenOnLedgerMetadata, precision); err != nil {
				return err
			}
		}

		if !issuer.IsNone() && r.issued.Issued(tms, issuer, tok) {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("transaction [%s], found a token and I have issued it", txID)
			}
			if err := r.tokenStore.StoreIssuedHistoryToken(txID, index, tok, tokenOnLedger, tokenOnLedgerMetadata, issuer, precision); err != nil {
				return err
			}
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
