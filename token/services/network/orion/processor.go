/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"strconv"
	"strings"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"

	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/processor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/rws/keys"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type ONS interface {
	Name() string
	MetadataService() *orion.MetadataService
}

type RWSetProcessor struct {
	network    ONS
	nss        []string
	sp         view2.ServiceProvider
	ownership  network.Authorization
	issued     network.Issued
	tokenStore processor.TokenStore
}

func NewTokenRWSetProcessor(network ONS, ns string, sp view2.ServiceProvider, ownership network.Authorization, issued network.Issued, tokenStore processor.TokenStore) *RWSetProcessor {
	return &RWSetProcessor{
		network:    network,
		nss:        []string{ns},
		sp:         sp,
		ownership:  ownership,
		issued:     issued,
		tokenStore: tokenStore,
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
	if tx.Network() != r.network.Name() {
		logger.Debugf("tx's network [%s]!=[%s]", tx.Network(), r.network.Name())
		return nil
	}

	return r.tokenRequest(req, tx, rws, ns)
}

func (r *RWSetProcessor) tokenRequest(req orion.Request, tx orion.ProcessTransaction, rws *orion.RWSet, ns string) error {
	txID := tx.ID()

	if !r.network.MetadataService().Exists(txID) {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s] is not known to this node, no need to extract tokens", txID)
		}
		return nil
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("transaction [%s] is known, extract tokens", txID)
		logger.Debugf("transaction [%s], parsing writes [%d]", txID, rws.NumWrites(ns))
	}
	transientMap, err := r.network.MetadataService().LoadTransient(txID)
	if err != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s], failed getting transient map", txID)
		}
		return err
	}
	if !transientMap.Exists(ttx.TokenRequestMetadata) {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s], no transient map found", txID)
		}
		return nil
	}

	tms := token.GetManagementService(
		r.sp,
		token.WithNetwork(tx.Network()),
		token.WithNamespace(ns),
	)
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("transaction [%s on (%s)] is known, extract tokens", txID, tms.ID())
	}
	metadata, err := tms.NewMetadataFromBytes(transientMap.Get(ttx.TokenRequestMetadata))
	if err != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s], failed getting zkat state from transient map [%s]", txID, err)
		}
		return err
	}

	wrappedRWS := &rwsWrapper{RWSet: rws}

	pp := tms.PublicParametersManager().PublicParameters()
	if pp.GraphHiding() {
		// Delete inputs
		ids := metadata.SpentTokenID()
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s] with graph hiding, delete inputs [%v]", txID, ids)
		}
		for _, id := range ids {
			if err := r.tokenStore.DeleteToken(ns, id.TxId, id.Index, wrappedRWS, tx.ID()); err != nil {
				return err
			}
		}
	}

	var keysToAppend []string
	var valuesToAppend [][]byte

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

		// the vault does not understand keys with `~`, therefore we store the equivalent key-value pairs without that symbol.
		keysToAppend = append(keysToAppend, key)
		valuesToAppend = append(valuesToAppend, tokenOnLedger)

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
			if err := r.tokenStore.DeleteToken(ns, components[0], index, wrappedRWS, tx.ID()); err != nil {
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
			if err := r.tokenStore.StoreToken(ns, txID, index, tok, wrappedRWS, tokenOnLedger, tokenOnLedgerMetadata, ids); err != nil {
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
			if err := r.tokenStore.StoreAuditToken(ns, txID, index, tok, wrappedRWS, tokenOnLedger, tokenOnLedgerMetadata); err != nil {
				return err
			}
		}

		if !issuer.IsNone() && r.issued.Issued(tms, issuer, tok) {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("transaction [%s], found a token and I have issued it", txID)
			}
			if err := r.tokenStore.StoreIssuedHistoryToken(ns, txID, index, tok, wrappedRWS, tokenOnLedger, tokenOnLedgerMetadata, issuer, pp.Precision()); err != nil {
				return err
			}
		}

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("Done parsing write key [%s]", key)
		}
	}

	for i := 0; i < len(valuesToAppend); i++ {
		if err := wrappedRWS.SetState(ns, keysToAppend[i], valuesToAppend[i]); err != nil {
			logger.Errorf("failed to set state [%s]", err)
			return errors.Wrapf(err, "failed to set state [%s]", keysToAppend[i])
		}
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("transaction [%s] is known, extract tokens, done!", txID)
	}

	return nil
}

type rwsWrapper struct {
	*orion.RWSet
}

func (r *rwsWrapper) SetState(namespace string, key string, value []byte) error {
	return r.RWSet.SetState(namespace, notOrionKey(key), value)
}

func (r *rwsWrapper) GetState(namespace string, key string) ([]byte, error) {
	return r.RWSet.GetState(namespace, notOrionKey(key), orion.FromStorage)
}

func (r *rwsWrapper) GetStateMetadata(namespace, key string) (map[string][]byte, error) {
	return r.RWSet.GetStateMetadata(namespace, notOrionKey(key), orion.FromStorage)
}

func (r *rwsWrapper) DeleteState(namespace string, key string) error {
	return r.RWSet.DeleteState(namespace, notOrionKey(key))
}

func (r *rwsWrapper) SetStateMetadata(namespace, key string, metadata map[string][]byte) error {
	return r.RWSet.SetStateMetadata(namespace, notOrionKey(key), metadata)
}
