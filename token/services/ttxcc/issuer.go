/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttxcc

import (
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
)

type registerIssuerIdentityView struct {
	Network   string
	Channel   string
	tokenType string
}

func (r *registerIssuerIdentityView) Call(context view.Context) (interface{}, error) {
	n := network.GetInstance(context, r.Network, r.Channel)

	ts := token.GetManagementService(
		context,
		token.WithNetwork(n.Name()),
		token.WithChannel(n.Channel()),
	)
	sk, pk, err := ts.WalletManager().GenerateIssuerKeyPair(r.tokenType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed generating issuer key pair [%s]", r.tokenType)
	}

	// Register pk to the chaincode
	pkRaw := pk.Bytes()

	if err := r.registerKey(context, pkRaw); err != nil {
		return nil, errors.Wrapf(err, "failed registering issuer key pair [%s]", r.tokenType)
	}

	if err := ts.WalletManager().RegisterIssuer(r.tokenType, sk, pk); err != nil {
		return nil, errors.Wrapf(err, "failed registering issuer key pair locally [%s]", r.tokenType)
	}

	return nil, nil
}

func (r *registerIssuerIdentityView) registerKey(context view.Context, pk []byte) error {
	return network.GetInstance(context, r.Network, r.Channel).AddIssuer(context, pk)
}

func NewRegisterIssuerIdentityView(tokenType string) *registerIssuerIdentityView {
	return &registerIssuerIdentityView{tokenType: tokenType}
}
