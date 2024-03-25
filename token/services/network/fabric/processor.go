/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"encoding/base64"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/committer"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/rws/keys"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var logger = flogging.MustGetLogger("token-sdk.vault.processor")

type GetTokensFunc = func() (*tokens.Tokens, error)
type GetTMSProviderFunc = func() *token.ManagementServiceProvider
type GetTokenRequestFunc = func(tms *token.ManagementService, txID string) ([]byte, error)

type RWSetProcessor struct {
	network         string
	nss             []string
	GetTokens       GetTokensFunc
	GetTMSProvider  GetTMSProviderFunc
	GetTokenRequest GetTokenRequestFunc
}

func NewTokenRWSetProcessor(network string, ns string, GetTokens GetTokensFunc, GetTMSProvider GetTMSProviderFunc, GetTokenRequest GetTokenRequestFunc) *RWSetProcessor {
	return &RWSetProcessor{
		network:         network,
		nss:             []string{ns},
		GetTokens:       GetTokens,
		GetTMSProvider:  GetTMSProvider,
		GetTokenRequest: GetTokenRequest,
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
		return r.tokenRequest(tx, rws, ns)
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
			tokens, err := r.GetTokens()
			if err != nil {
				return err
			}
			if err := tokens.StorePublicParams(val); err != nil {
				return errors.Wrapf(err, "failed storing public params")
			}
			break
		}
	}
	return nil
}

func (r *RWSetProcessor) tokenRequest(tx fabric.ProcessTransaction, rws *fabric.RWSet, ns string) error {
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

	if err := r.checkTokenRequest(tx.ID(), request, rws, ns); err != nil {
		return err
	}

	tokens, err := r.GetTokens()
	if err != nil {
		return err
	}
	if err := tokens.AppendTransaction(&Transaction{
		id:        txID,
		network:   tms.Network(),
		channel:   tms.Channel(),
		namespace: tms.Namespace(),
		request:   request,
	}); err != nil {
		logger.Errorf("failed to append transaction [%s] to db: [%s]", txID, err)
		return err
	}
	return nil
}

func (r *RWSetProcessor) checkTokenRequest(txID string, request *token.Request, rws *fabric.RWSet, ns string) error {
	key, err := keys.CreateTokenRequestKey(txID)
	if err != nil {
		return errors.Errorf("can't create for token request '%s'", txID)
	}
	rwsTrHash, err := rws.GetState(ns, key, fabric.FromIntermediate)
	if err != nil {
		return errors.Errorf("can't get request has '%s'", txID)
	}
	trToSign, err := request.MarshalToSign()
	if err != nil {
		return errors.Errorf("can't get request hash '%s'", txID)
	}
	if base64.StdEncoding.EncodeToString(rwsTrHash) != hash.Hashable(trToSign).String() {
		logger.Errorf("tx [%s], tr hashes [%s][%s]", txID, base64.StdEncoding.EncodeToString(rwsTrHash), hash.Hashable(trToSign))
		// no further processing of the tokens of these transactions
		return errors.Wrapf(
			committer.ErrDiscardTX,
			"tx [%s], token requests do not match, tr hashes [%s][%s]",
			txID,
			base64.StdEncoding.EncodeToString(rwsTrHash),
			hash.Hashable(trToSign),
		)
	}
	return nil
}

type Transaction struct {
	id        string
	network   string
	channel   string
	namespace string
	request   *token.Request
}

func (t *Transaction) ID() string {
	return t.id
}

func (t *Transaction) Network() string {
	return t.network
}

func (t *Transaction) Channel() string {
	return t.channel
}

func (t *Transaction) Namespace() string {
	return t.namespace
}

func (t *Transaction) Request() *token.Request {
	return t.request
}
