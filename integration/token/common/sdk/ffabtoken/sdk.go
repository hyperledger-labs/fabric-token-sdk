/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ffabtoken

import (
	"errors"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	digutils "github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/dig"
	fabric2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/id"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	view3 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/server/view"
	fabtoken "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/driver"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/dig"
)

type SDK struct {
	*tokensdk.SDK
}

func NewSDK(registry node.Registry) *SDK {
	return &SDK{SDK: tokensdk.NewSDK(registry)}
}

func (p *SDK) Install() error {
	err := errors.Join(
		p.Container().Provide(fabric.NewGenericDriver, dig.Group("network-drivers")),
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
	)
}
