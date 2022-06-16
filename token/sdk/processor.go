/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	network2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/exchange"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	orion2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	"github.com/pkg/errors"
)

type ProcessorManager struct {
	sp view.ServiceProvider
}

func NewProcessorManager(sp view.ServiceProvider) *ProcessorManager {
	return &ProcessorManager{sp: sp}
}

func (p *ProcessorManager) New(network, channel, namespace string) error {
	n := fabric.GetFabricNetworkService(p.sp, network)
	if n == nil && orion.GetOrionNetworkService(p.sp, network) != nil {
		ons := orion.GetOrionNetworkService(p.sp, network)
		if err := ons.ProcessorManager().AddProcessor(
			namespace,
			orion2.NewTokenRWSetProcessor(
				ons,
				namespace,
				p.sp,
				network2.NewAuthorizationMultiplexer(&network2.TMSAuthorization{}),
				network2.NewIssuedMultiplexer(&network2.WalletIssued{}),
			),
		); err != nil {
			return errors.WithMessagef(err, "failed to add processor to orion network [%s]", network)
		}
		return nil
	}

	if err := n.ProcessorManager().AddProcessor(
		namespace,
		fabric2.NewTokenRWSetProcessor(
			n,
			namespace,
			p.sp,
			network2.NewAuthorizationMultiplexer(&network2.TMSAuthorization{}, &exchange.ScriptOwnership{}),
			network2.NewIssuedMultiplexer(&network2.WalletIssued{}),
		),
	); err != nil {
		return errors.WithMessagef(err, "failed to add processor to fabric network [%s]", network)
	}
	return nil

}
