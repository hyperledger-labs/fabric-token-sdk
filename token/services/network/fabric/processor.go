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
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

var logger = flogging.MustGetLogger("token-sdk.vault.processor")

type net interface {
	Name() string
	Channel(id string) (*fabric.Channel, error)
}

type Ownership interface {
	IsMine(tms *token.ManagementService, tok *token2.Token) ([]string, bool)
}

type Issued interface {
	// Issued returns true if the passed issuer issued the passed token
	Issued(tms *token.ManagementService, issuer view.Identity, tok *token2.Token) bool
}

type RWSetProcessor struct {
	network   net
	nss       []string
	sp        view2.ServiceProvider
	ownership Ownership
	issued    Issued
}

func NewTokenRWSetProcessor(network net, ns string, sp view2.ServiceProvider, ownership Ownership, issued Issued) *RWSetProcessor {
	return &RWSetProcessor{
		network:   network,
		nss:       []string{ns},
		sp:        sp,
		ownership: ownership,
		issued:    issued,
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
	case "setup":
		return r.setup(req, tx, rws, ns)
	default:
		return r.tokenRequest(req, tx, rws, ns)
	}
}

func (r *RWSetProcessor) setup(req fabric.Request, tx fabric.ProcessTransaction, rws *fabric.RWSet, ns string) error {
	logger.Debugf("[setup] store setup bundle")
	key, err := keys.CreateSetupBundleKey()
	if err != nil {
		return err
	}
	logger.Debugf("[setup] store setup bundle [%s,%s]", key, req.ID())
	err = rws.SetState(ns, key, []byte(req.ID()))
	if err != nil {
		logger.Errorf("failed setting setup bundle state [%s,%s]", key, req.ID())
		return errors.Wrapf(err, "failed setting setup bundle state [%s,%s]", key, req.ID())
	}
	logger.Debugf("[setup] store setup bundle done")

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
	if !transientMap.Exists("zkat") {
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
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("transaction [%s on (%s)] is known, extract tokens", txID, tms.ID())
	}
	metadata, err := tms.NewMetadataFromBytes(transientMap.Get("zkat"))
	if err != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction [%s], failed getting zkat state from transient map [%s]", txID, err)
		}
		return err
	}

	if tms.PublicParametersManager().GraphHiding() {
		// Delete inputs
		for _, id := range metadata.SpentTokenID() {
			if err := r.deleteFabToken(ns, id.TxId, id.Index, rws); err != nil {
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
			panic(err)
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
		case keys.SignaturePrefix:
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("expected key without the sig metadata, skipping")
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
			if err := r.deleteFabToken(ns, components[0], index, rws); err != nil {
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
			if err := r.storeFabToken(ns, txID, index, tok, rws, tokenInfoRaw, ids); err != nil {
				return err
			}
		} else {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("transaction [%s], found a token and I must be the auditor", txID)
			}
			if err := r.storeAuditToken(ns, txID, index, tok, rws, tokenInfoRaw); err != nil {
				return err
			}
		}

		if !issuer.IsNone() && r.issued.Issued(tms, issuer, tok) {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("transaction [%s], found a token and I have issued it", txID)
			}
			if err := r.storeIssuedHistoryToken(ns, txID, index, tok, rws, tokenInfoRaw, issuer); err != nil {
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
