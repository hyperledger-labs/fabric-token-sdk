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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/pkg/errors"
)

var (
	logger = flogging.MustGetLogger("token-sdk.network")
)

// TokenSDKConfig is the configuration for the token SDK
type TokenSDKConfig interface {
	// LookupNamespace searches for a TMS configuration that matches the given network and channel, and
	// return its namespace.
	// If no matching configuration is found, an error is returned.
	// If multiple matching configurations are found, an error is returned.
	LookupNamespace(network, channel string) (string, error)
}

// Normalizer is a normalizer for token service options
// Namely, if no network is specified, it will try to find a default network. And so on.
type Normalizer struct {
	sp             view.ServiceProvider
	tokenSDKConfig TokenSDKConfig
}

// NewNormalizer creates a new Normalizer
func NewNormalizer(cp TokenSDKConfig, sp view.ServiceProvider) *Normalizer {
	return &Normalizer{tokenSDKConfig: cp, sp: sp}
}

// Normalize normalizes the passed options.
// If no network is specified, it will try to find a default network. And so on.
func (n *Normalizer) Normalize(opt *token.ServiceOptions) (*token.ServiceOptions, error) {
	if len(opt.Network) == 0 {
		if fns, err := fabric.GetDefaultFNS(n.sp); err == nil {
			logger.Debugf("no network specified, using default FNS: %s", fns.Name())
			opt.Network = fns.Name()
		} else if ons, err := orion2.GetDefaultONS(n.sp); err == nil {
			logger.Debugf("no network specified, using default ONS: %s", ons.Name())
			opt.Network = ons.Name()
		} else {
			return nil, errors.Errorf("No network specified, and no default FNS or ONS found")
		}
	}

	if len(opt.Channel) == 0 {
		if fns, err := fabric.GetFabricNetworkService(n.sp, opt.Network); err == nil {
			logger.Debugf("no channel specified, using default channel: %s", fns.ConfigService().DefaultChannel())
			opt.Channel = fns.ConfigService().DefaultChannel()
		} else if _, err := orion2.GetOrionNetworkService(n.sp, opt.Network); err == nil {
			logger.Debugf("no need to specify channel for orion")
			// Nothing to do here
		} else {
			return nil, errors.Errorf("no channel specified, and no default channel found")
		}
	}

	if len(opt.Namespace) == 0 {
		if ns, err := n.tokenSDKConfig.LookupNamespace(opt.Network, opt.Channel); err == nil {
			logger.Debugf("no namespace specified, found namespace [%s] for [%s:%s]", ns, opt.Network, opt.Channel)
			opt.Namespace = ns
		} else {
			logger.Errorf("no namespace specified, and no default namespace found [%s], use default [%s]", err, ttx.TokenNamespace)
			opt.Namespace = ttx.TokenNamespace
		}
	}
	if opt.PublicParamsFetcher == nil {
		opt.PublicParamsFetcher = NewPublicParamsFetcher(n.sp, opt.Network, opt.Channel, opt.Namespace)
	}
	return opt, nil
}
