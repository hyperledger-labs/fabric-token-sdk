/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package network

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	tcc "github.com/hyperledger-labs/fabric-token-sdk/token/services/tcc/fetcher"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
)

type Normalizer struct {
	sp view.ServiceProvider
}

func NewNormalizer(sp view.ServiceProvider) *Normalizer {
	return &Normalizer{sp: sp}
}

func (n *Normalizer) Normalize(opt *token.ServiceOptions) *token.ServiceOptions {
	if len(opt.Network) == 0 {
		opt.Network = fabric.GetDefaultFNS(n.sp).Name()
	}
	if len(opt.Channel) == 0 {
		opt.Channel = fabric.GetFabricNetworkService(n.sp, opt.Network).DefaultChannel()
	}
	if len(opt.Namespace) == 0 {
		opt.Namespace = keys.TokenNameSpace
	}
	if opt.PublicParamsFetcher == nil {
		opt.PublicParamsFetcher = tcc.NewPublicParamsFetcher(n.sp, opt.Network, opt.Channel, opt.Namespace)
	}
	return opt
}
