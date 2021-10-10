/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package tcc

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/chaincode"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type RegisterCertifierView struct {
	Network   string
	Channel   string
	Namespace string
	Id        view.Identity
}

func NewRegisterCertifierView(network string, channel string, namespace string, id view.Identity) *RegisterCertifierView {
	return &RegisterCertifierView{Network: network, Channel: channel, Namespace: namespace, Id: id}
}

func (r *RegisterCertifierView) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(
		context,
		token.WithNetwork(r.Network),
		token.WithChannel(r.Channel),
		token.WithNamespace(r.Namespace),
	)
	if !tms.PublicParametersManager().GraphHiding() {
		logger.Warnf("the token management system for [%s:%s] does not support graph hiding, skipping certifier registration at the token chaincode", r.Channel, r.Namespace)
		return nil, nil
	}

	var set bool
	key := "token-sdk.tcc.certifier.registered"
	if kvs.GetService(context).Exists(key) {
		if err := kvs.GetService(context).Get(key, &set); err != nil {
			logger.Errorf("failed checking certifier has been registered [%s]", err)
			set = false
		}
	}

	if !set {
		logger.Debugf("register certifier [%s]", r.Id.String())
		_, err := context.RunView(chaincode.NewInvokeView(
			tms.Namespace(), AddCertifierFunction, r.Id.Bytes(),
		).WithNetwork(tms.Network()).WithChannel(tms.Channel()).WithSignerIdentity(
			fabric.GetFabricNetworkService(context, tms.Network()).IdentityProvider().DefaultIdentity(),
		))
		if err != nil {
			return nil, errors.WithMessagef(err, "failed certifier registration")
		}
		if err := kvs.GetService(context).Put(key, true); err != nil {
			logger.Errorf("failed recording auditor has been registered to the chaincode [%s]", err)
		}
	}
	return nil, nil
}

type GetTokenView struct {
	Network   string
	Channel   string
	Namespace string
	IDs       []*token2.Id
}

func NewGetTokensView(channel string, namespace string, ids ...*token2.Id) *GetTokenView {
	if len(ids) == 0 {
		panic("no ids specified")
	}
	return &GetTokenView{Channel: channel, Namespace: namespace, IDs: ids}
}

func (r *GetTokenView) Call(context view.Context) (interface{}, error) {
	idsRaw, err := json.Marshal(r.IDs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling ids")
	}

	tms := token.GetManagementService(
		context,
		token.WithNetwork(r.Network),
		token.WithChannel(r.Channel),
		token.WithNamespace(r.Namespace),
	)
	payloadBoxed, err := context.RunView(chaincode.NewQueryView(
		tms.Namespace(),
		QueryTokensFunctions,
		idsRaw,
	).WithNetwork(tms.Network()).WithChannel(tms.Channel()))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed quering tokens")
	}

	// Unbox
	raw, ok := payloadBoxed.([]byte)
	if !ok {
		return nil, errors.Errorf("expected []byte from TCC, got [%T]", payloadBoxed)
	}
	var tokens [][]byte
	if err := json.Unmarshal(raw, &tokens); err != nil {
		return nil, errors.Wrapf(err, "failed marshalling response")
	}
	return tokens, nil
}
