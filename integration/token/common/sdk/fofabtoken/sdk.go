/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fofabtoken

import (
	"errors"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	digutils "github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/dig"
	fabric2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	fabricsdk "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk/dig"
	orion2 "github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	core2 "github.com/hyperledger-labs/fabric-smart-client/platform/orion/core"
	orionsdk "github.com/hyperledger-labs/fabric-smart-client/platform/orion/sdk/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/id"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	viewsdk "github.com/hyperledger-labs/fabric-smart-client/platform/view/sdk/dig"
	view3 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/server/view"
	fabtoken "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/driver"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/dig"
)

type SDK struct {
	*tokensdk.SDK
}

func NewSDK(registry node.Registry) *SDK {
	return &SDK{SDK: tokensdk.NewFrom(fabricsdk.NewFrom(orionsdk.NewFrom(viewsdk.NewSDK(registry))))}
}

func (p *SDK) FabricEnabled() bool {
	return p.ConfigService().GetBool("fabric.enabled")
}
func (p *SDK) OrionEnabled() bool { return p.ConfigService().GetBool("orion.enabled") }

func (p *SDK) Install() error {
	if p.FabricEnabled() {
		if err := p.Container().Provide(fabric.NewGenericDriver, dig.Group("network-drivers")); err != nil {
			return err
		}
	}
	if p.OrionEnabled() {
		if err := p.Container().Provide(orion.NewOrionDriver, dig.Group("network-drivers")); err != nil {
			return err
		}
	}
	err := errors.Join(
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
		digutils.Register[driver.EndpointService](p.Container()),
		digutils.Register[view3.IdentityProvider](p.Container()),
		digutils.Register[node.ViewManager](p.Container()), // Need to add it as a field in the node
		digutils.Register[id.SigService](p.Container()),
		digutils.RegisterOptional[*fabric2.NetworkServiceProvider](p.Container()), // GetFabricNetworkService is used by many components
		digutils.RegisterOptional[*orion2.NetworkServiceProvider](p.Container()),
		digutils.RegisterOptional[*core2.ONSProvider](p.Container()),
	)
}
