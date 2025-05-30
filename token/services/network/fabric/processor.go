/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type GetTokensFunc = func() (*tokens.Service, error)
type GetTMSProviderFunc = func() *token.ManagementServiceProvider

type RWSetProcessor struct {
	network        string
	nss            []string
	GetTokens      GetTokensFunc
	GetTMSProvider GetTMSProviderFunc
	KeyTranslator  translator.KeyTranslator
}

func NewTokenRWSetProcessor(network string, ns string, GetTokens GetTokensFunc, GetTMSProvider GetTMSProviderFunc, KeyTranslator translator.KeyTranslator) *RWSetProcessor {
	return &RWSetProcessor{
		network:        network,
		nss:            []string{ns},
		GetTokens:      GetTokens,
		GetTMSProvider: GetTMSProvider,
		KeyTranslator:  KeyTranslator,
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
		return r.init(context.Background(), tx, rws, ns)
	default:
		return nil
	}
}

// init when invoked extracts the public params from rwset and updates the local version
func (r *RWSetProcessor) init(ctx context.Context, tx fabric.ProcessTransaction, rws *fabric.RWSet, ns string) error {
	tsmProvider := r.GetTMSProvider()
	setUpKey, err := r.KeyTranslator.CreateSetupKey()
	if err != nil {
		return errors.Errorf("failed creating setup key")
	}
	for i := 0; i < rws.NumWrites(ns); i++ {
		key, val, err := rws.GetWriteAt(ns, i)
		if err != nil {
			return err
		}
		if key == setUpKey {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("Parsing write key [%s] with hash value [%s]", key, hash.Hashable(val))
			}
			if err := tsmProvider.Update(token.TMSID{
				Network:   tx.Network(),
				Channel:   tx.Channel(),
				Namespace: ns,
			}, val); err != nil {
				return errors.Wrapf(err, "failed updating public params")
			}
			tokens, err := r.GetTokens()
			if err != nil {
				return err
			}
			if err := tokens.StorePublicParams(ctx, val); err != nil {
				return errors.Wrapf(err, "failed storing public params")
			}
			break
		}
	}
	return nil
}
