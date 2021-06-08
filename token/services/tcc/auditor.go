/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package tcc

import (
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/chaincode"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

type RegisterAuditorView struct {
	Network   string
	Channel   string
	Namespace string
	Id        view.Identity
}

func NewRegisterAuditorView(network string, channel string, namespace string, id view.Identity) *RegisterAuditorView {
	return &RegisterAuditorView{Network: network, Channel: channel, Namespace: namespace, Id: id}
}

func (r *RegisterAuditorView) Call(context view.Context) (interface{}, error) {
	var set bool
	key := "token-sdk.tcc.auditor.registered"
	if kvs.GetService(context).Exists(key) {
		if err := kvs.GetService(context).Get(key, &set); err != nil {
			logger.Errorf("failed checking auditor has been registered to the chaincode [%s]", err)
			set = false
		}
	}

	if !set {
		tms := token.GetManagementService(
			context,
			token.WithNetwork(r.Network),
			token.WithChannel(r.Channel),
			token.WithNamespace(r.Namespace),
		)
		_, err := context.RunView(chaincode.NewInvokeView(
			tms.Namespace(), AddAuditorFunction, r.Id.Bytes(),
		).WithNetwork(tms.Network()).WithChannel(tms.Channel()))
		if err != nil {
			return nil, errors.WithMessagef(err, "failed auditor registration")
		}

		if err := kvs.GetService(context).Put(key, true); err != nil {
			logger.Errorf("failed recording auditor has been registered to the chaincode [%s]", err)
		}

		if err := tms.PublicParametersManager().ForceFetch(); err != nil {
			logger.Warnf("failed fetching parameters [%s]", err)
		}
	}
	return nil, nil
}
