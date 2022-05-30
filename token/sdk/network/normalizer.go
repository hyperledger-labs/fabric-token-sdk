/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package network

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	orion2 "github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	tcc "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/tcc/fetcher"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
)

var (
	logger = flogging.MustGetLogger("token-sdk.network")
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
			logger.Debugf("No network specified, using default FNS: %s", fns.Name())
			opt.Network = fns.Name()
		} else if ons := orion2.GetDefaultONS(n.sp); ons != nil {
			logger.Debugf("No network specified, using default ONS: %s", ons.Name())
			opt.Network = ons.Name()
		} else {
			logger.Errorf("No network specified, and no default FNS or ONS found")
			panic("no network found")
		}
	}

	if len(opt.Channel) == 0 {
		if fns := fabric.GetFabricNetworkService(n.sp, opt.Network); fns != nil {
			logger.Debugf("No channel specified, using default channel: %s", fns.DefaultChannel())
			opt.Channel = fns.DefaultChannel()
		} else if ons := orion2.GetOrionNetworkService(n.sp, opt.Network); ons != nil {
			logger.Debugf("No need to specify channel for orion")
			// Nothing to do here
		} else {
			logger.Errorf("No channel specified, and no default channel found")
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
