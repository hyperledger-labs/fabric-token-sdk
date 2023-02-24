/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"strconv"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/processor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var logger = flogging.MustGetLogger("token-sdk.vault.processor")

type net interface {
	Name() string
	Channel(id string) (*fabric.Channel, error)
}

type RWSetProcessor struct {
	network    net
	nss        []string
	sp         view2.ServiceProvider
	ownership  network.Authorization
	issued     network.Issued
	tokenStore processor.TokenStore
}

func NewTokenRWSetProcessor(network net, ns string, sp view2.ServiceProvider, ownership network.Authorization, issued network.Issued, tokenStore processor.TokenStore) *RWSetProcessor {
	return &RWSetProcessor{
		network:    network,
		nss:        []string{ns},
		sp:         sp,
		ownership:  ownership,
		issued:     issued,
		tokenStore: tokenStore,
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
	if tx.Network() != r.network.Name() {
		logger.Debugf("tx's network [%s]!=[%s]", tx.Network(), r.network.Name())
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
	tms := token.GetManagementService(
		r.sp,
		token.WithNetwork(tx.Network()),
		token.WithChannel(tx.Channel()),
		token.WithNamespace(ns),
	)
	if tms == nil {
		return errors.Errorf("failed getting token management service [%s:%s:%s]", tx.Network(), tx.Channel(), ns)
	}

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
			logger.Debugf("setting new public parameters...")
			err = tms.PublicParametersManager().SetPublicParameters(val)
			if err != nil {
				return errors.Wrapf(err, "failed updating public params ")
			}
			logger.Debugf("setting new public parameters...done.")
			break
		}
	}
	logger.Debugf("Successfully updated public parameters")
	return nil
}

func (r *RWSetProcessor) tokenRequest(req fabric.Request, tx fabric.ProcessTransaction, rws *fabric.RWSet, ns string) error {
	txID := tx.ID()

	ch, err := r.network.Channel(tx.Channel())
	if err != nil {
		return errors.Wrapf(err, "failed getting channel [%s]", tx.Channel())
	}
	if !ch.MetadataService().Exists(txID) {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s] is not known to this node, no need to extract tokens", txID)
		}
		return nil
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("transaction [%s] is known, extract tokens", txID)
		logger.Debugf("transaction [%s], parsing writes [%d]", txID, rws.NumWrites(ns))
	}
	transientMap, err := ch.MetadataService().LoadTransient(txID)
	if err != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s], failed getting transient map", txID)
		}
		return err
	}
	if !transientMap.Exists(keys.TokenRequestMetadata) {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s], no transient map found", txID)
		}
		return nil
	}

	tms := token.GetManagementService(
		r.sp,
		token.WithNetwork(tx.Network()),
		token.WithChannel(tx.Channel()),
		token.WithNamespace(ns),
	)
	if tms == nil {
		return errors.Errorf("failed getting token management service [%s:%s:%s]", tx.Network(), tx.Channel(), ns)
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("transaction [%s on (%s)] is known, extract tokens", txID, tms.ID())
	}
	metadata, err := tms.NewMetadataFromBytes(transientMap.Get(keys.TokenRequestMetadata))
	if err != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s], failed getting zkat state from transient map [%s]", txID, err)
		}
		return err
	}

	wrappedRWS := &rwsWrapper{RWSet: rws}

	if tms.PublicParametersManager().GraphHiding() {
		ids := metadata.SpentTokenID()
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s] with graph hiding, delete inputs [%v]", txID, ids)
		}
		for _, id := range ids {
			if err := r.tokenStore.DeleteFabToken(ns, id.TxId, id.Index, wrappedRWS, tx.ID()); err != nil {
				return err
			}
		}
	}

	for i := 0; i < rws.NumWrites(ns); i++ {
		key, val, err := rws.GetWriteAt(ns, i)
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

		// This is a delete, add a delete for fabtoken
		if len(val) == 0 {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("transaction [%s] without graph hiding, delete input [%s:%d]", txID, components[0], index)
			}
			if err := r.tokenStore.DeleteFabToken(ns, components[0], index, wrappedRWS, tx.ID()); err != nil {
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
		tok, issuer, tokenInfoRaw, err := metadata.GetToken(val)
		if err != nil {
			logger.Errorf("transaction [%s], found a token but failed getting the clear version, skipping it [%s]", txID, err)
			continue
		}
		if tok == nil {
			logger.Warnf("failed getting token in the clear for key [%s, %s]", key, string(val))
			continue
		}

		ids, mine := r.ownership.IsMine(tms, tok)
		if mine {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("transaction [%s], found a token and it is mine", txID)
			}
			// Add a lookup key to identity quickly that this token belongs to this
			mineTokenID, err := keys.CreateTokenMineKey(components[0], index)
			if err != nil {
				return errors.Wrapf(err, "failed computing mine key for for key [%s]", key)
			}
			err = rws.SetState(ns, mineTokenID, []byte{1})
			if err != nil {
				return err
			}

			// Store Fabtoken-like entry
			if err := r.tokenStore.StoreFabToken(ns, txID, index, tok, wrappedRWS, tokenInfoRaw, ids); err != nil {
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
			if err := r.tokenStore.StoreAuditToken(ns, txID, index, tok, wrappedRWS, tokenInfoRaw); err != nil {
				return err
			}
		}

		if !issuer.IsNone() && r.issued.Issued(tms, issuer, tok) {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("transaction [%s], found a token and I have issued it", txID)
			}
			if err := r.tokenStore.StoreIssuedHistoryToken(ns, txID, index, tok, wrappedRWS, tokenInfoRaw, issuer, tms.PublicParametersManager().Precision()); err != nil {
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

type rwsWrapper struct {
	*fabric.RWSet
}

func (r *rwsWrapper) SetState(namespace string, key string, value []byte) error {
	return r.RWSet.SetState(namespace, key, value)
}

func (r *rwsWrapper) GetState(namespace string, key string) ([]byte, error) {
	return r.RWSet.GetState(namespace, key, fabric.FromStorage)
}

func (r *rwsWrapper) GetStateMetadata(namespace, key string) (map[string][]byte, error) {
	return r.RWSet.GetStateMetadata(namespace, key, fabric.FromStorage)
}

func (r *rwsWrapper) DeleteState(namespace string, key string) error {
	return r.RWSet.DeleteState(namespace, key)
}

func (r *rwsWrapper) SetStateMetadata(namespace, key string, metadata map[string][]byte) error {
	return r.RWSet.SetStateMetadata(namespace, key, metadata)
}
