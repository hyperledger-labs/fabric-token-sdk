/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tcc

import (
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type GetTokenView struct {
	Network   string
	Channel   string
	Namespace string
	IDs       []*token2.ID
}

func NewGetTokensView(channel string, namespace string, ids ...*token2.ID) *GetTokenView {
	return &GetTokenView{Channel: channel, Namespace: namespace, IDs: ids}
}

func (r *GetTokenView) Call(context view.Context) (interface{}, error) {
	if len(r.IDs) == 0 {
		return nil, errors.Errorf("no token ids provided")
	}
	tms := token.GetManagementService(
		context,
		token.WithNetwork(r.Network),
		token.WithChannel(r.Channel),
		token.WithNamespace(r.Namespace),
	)
	tokens, err := network.GetInstance(context, tms.Network(), tms.Channel()).QueryTokens(context, tms.Namespace(), r.IDs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed querying tokens")
	}
	return tokens, nil
}
