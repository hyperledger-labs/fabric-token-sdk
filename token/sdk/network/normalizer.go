/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package network

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	orion2 "github.com/hyperledger-labs/fabric-smart-client/platform/orion"
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
		if fns := fabric.GetDefaultFNS(n.sp); fns != nil {
			opt.Network = fns.Name()
		} else if ons := orion2.GetDefaultONS(n.sp); ons != nil {
			opt.Network = ons.Name()
		} else {
			panic("no network found")
		}
	}

	if len(opt.Channel) == 0 {
		if fns := fabric.GetFabricNetworkService(n.sp, opt.Network); fns != nil {
			opt.Channel = fns.DefaultChannel()
		} else if ons := orion2.GetOrionNetworkService(n.sp, opt.Network); ons != nil {
			// Nothing to do here
		} else {
			panic("no network found for " + opt.Network)
		}
	}

	if len(opt.Namespace) == 0 {
		opt.Namespace = keys.TokenNameSpace
	}
	if opt.PublicParamsFetcher == nil {
		opt.PublicParamsFetcher = tcc.NewPublicParamsFetcher(n.sp, opt.Network, opt.Channel, opt.Namespace)
	}
	return opt
}
