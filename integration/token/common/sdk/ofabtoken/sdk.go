/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ofabtoken

import (
	"errors"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	digutils "github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/dig"
	orion2 "github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	core2 "github.com/hyperledger-labs/fabric-smart-client/platform/orion/core"
	orionsdk "github.com/hyperledger-labs/fabric-smart-client/platform/orion/sdk/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/id"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	view3 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/server/view"
	fabtoken "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/driver"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	"go.opentelemetry.io/otel/trace"
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
		p.Container().Provide(orion.NewOrionDriver, dig.Group("network-drivers")),
		p.Container().Provide(fabtoken.NewDriver, dig.Group("token-drivers")),
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
