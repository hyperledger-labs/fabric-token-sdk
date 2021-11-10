/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package tcc

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
)

var logger = flogging.MustGetLogger("token-sdk.tms.zkat.fetcher")

type publicParamsFetcher struct {
	sp        view.ServiceProvider
	network   string
	channel   string
	namespace string
}

func NewPublicParamsFetcher(sp view.ServiceProvider, network string, channel string, namespace string) *publicParamsFetcher {
	return &publicParamsFetcher{
		sp:        sp,
		network:   network,
		channel:   channel,
		namespace: namespace,
	}
}

func (c *publicParamsFetcher) Fetch() ([]byte, error) {
	logger.Debugf("retrieve public params for [%s:%s:%s]", c.network, c.channel, c.namespace)
	n := network.GetInstance(c.sp, c.network, c.channel)
	if n == nil {
		return nil, errors.Errorf("network [%s:%s] does not exist", c.network, c.channel)
	}

	return n.FetchPublicParameters(c.namespace)
}
