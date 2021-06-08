/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package certifier

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tcc"
)

var logger = flogging.MustGetLogger("token-sdk.certifier")

type RegisterView struct {
	Network   string
	Channel   string
	Namespace string
	Wallet    string
}

func NewRegisterView(network string, channel string, namespace string, wallet string) *RegisterView {
	return &RegisterView{Network: network, Channel: channel, Namespace: namespace, Wallet: wallet}
}

func (r *RegisterView) Call(context view.Context) (interface{}, error) {
	// If the tms does not support graph hiding, skip
	tms := token.GetManagementService(
		context,
		token.WithNetwork(r.Network),
		token.WithChannel(r.Channel),
		token.WithNamespace(r.Namespace),
	)
	if !tms.PublicParametersManager().GraphHiding() {
		logger.Warnf("the token management system for [%s:%s] does not support graph hiding, skipping certifier registration", r.Channel, r.Namespace)
		return nil, nil
	}

	// Register certifier at the Token Chaincode
	w := tms.WalletManager().CertifierWallet(r.Wallet)
	if w == nil {
		return nil, errors.Errorf("failed getting wallet [%s][%s]", tms, r.Wallet)
	}
	id, err := w.GetCertifierIdentity()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting certfier identity [%s][%s]", tms, r.Wallet)
	}
	if _, err := context.RunView(tcc.NewRegisterCertifierView(r.Network, r.Channel, r.Namespace, id)); err != nil {
		return nil, errors.WithMessagef(err, "failed registering certifier [%s][%s]", tms, r.Wallet)
	}

	if err := tms.PublicParametersManager().ForceFetch(); err != nil {
		return nil, errors.WithMessagef(err, "failed registering certifier [%s][%s]", tms, r.Wallet)
	}

	// Start Certifier
	c, err := certifier.NewCertificationService(
		context,
		tms.Network(),
		tms.Channel(),
		tms.Namespace(),
		r.Wallet,
		tms.PublicParametersManager().CertificationDriver(),
	)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed instantiating certifier [%s]", tms)
	}
	if err := c.Start(); err != nil {
		return nil, errors.WithMessagef(err, "failed starting certifier [%s]", tms)
	}

	return nil, nil
}
