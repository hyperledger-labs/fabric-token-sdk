/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package network

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/pkg/errors"
)

type publicParamsFetcher struct {
	networkProvider *network.Provider
	network         string
	channel         string
	namespace       string
}

func NewPublicParamsFetcher(networkProvider *network.Provider, network string, channel string, namespace string) *publicParamsFetcher {
	return &publicParamsFetcher{
		networkProvider: networkProvider,
		network:         network,
		channel:         channel,
		namespace:       namespace,
	}
}

func (c *publicParamsFetcher) Fetch() ([]byte, error) {
	logger.Debugf("retrieve public params for [%s:%s:%s]", c.network, c.channel, c.namespace)
	n, err := c.networkProvider.GetNetwork(c.network, c.channel)
	if n == nil || err != nil {
		return nil, errors.Errorf("network [%s:%s] does not exist: %v", c.network, c.channel, err)
	}

	return n.FetchPublicParameters(c.namespace)
}
