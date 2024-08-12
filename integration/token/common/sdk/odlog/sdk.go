/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package odlog

import (
	"errors"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion/driver"
	orionsdk "github.com/hyperledger-labs/fabric-smart-client/platform/orion/sdk/dig"
	dlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/driver"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	"go.uber.org/dig"
)

type SDK struct {
	*tokensdk.SDK
}

func NewSDK(registry node.Registry) *SDK {
	return &SDK{SDK: tokensdk.NewFrom(orionsdk.NewSDK(registry))}
}

func (p *SDK) Install() error {
	err := errors.Join(
		p.Container().Provide(orion.NewDriver, dig.Group("network-drivers")),
		p.Container().Provide(dlog.NewDriver, dig.Group("token-drivers")),
	)
	if err != nil {
		return err
	}

	err = errors.Join(
		p.Container().Decorate(func(p driver.ListenerManagerProvider) driver.ListenerManagerProvider {
			return common.NewParallelListenerManagerProvider[driver.ValidationCode](p)
		}),
	)
	if err != nil {
		return err
	}

	return p.SDK.Install()
}
