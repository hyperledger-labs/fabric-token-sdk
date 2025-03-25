/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package odlog

import (
	"errors"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	dig2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/sdk/dig"
	digutils "github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/dig"
	orion2 "github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	core2 "github.com/hyperledger-labs/fabric-smart-client/platform/orion/core"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion/driver"
	orionsdk "github.com/hyperledger-labs/fabric-smart-client/platform/orion/sdk/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/id"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	view3 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/server/view"
	dlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/driver"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/dig"
)

type SDK struct {
	dig2.SDK
}

func NewSDK(registry node.Registry) *SDK {
	return NewFrom(tokensdk.NewFrom(orionsdk.NewSDK(registry)))
}

func NewFrom(sdk dig2.SDK) *SDK {
	return &SDK{SDK: sdk}
}

func (p *SDK) Install() error {
	err := errors.Join(
		p.Container().Provide(orion.NewOrionDriver, dig.Group("network-drivers")),
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

	if err := p.SDK.Install(); err != nil {
		return err
	}

	return errors.Join(
		digutils.Register[trace.TracerProvider](p.Container()),
		digutils.Register[driver2.EndpointService](p.Container()),
		digutils.Register[view3.IdentityProvider](p.Container()),
		digutils.Register[node.ViewManager](p.Container()), // Need to add it as a field in the node
		digutils.Register[id.SigService](p.Container()),

		digutils.RegisterOptional[*orion2.NetworkServiceProvider](p.Container()),
		digutils.RegisterOptional[*core2.ONSProvider](p.Container()),
	)
}
