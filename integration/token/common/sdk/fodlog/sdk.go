/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fodlog

import (
	"errors"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	fabricsdk "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk/dig"
	orionsdk "github.com/hyperledger-labs/fabric-smart-client/platform/orion/sdk/dig"
	viewsdk "github.com/hyperledger-labs/fabric-smart-client/platform/view/sdk/dig"
	dlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/driver"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	"go.uber.org/dig"
)

type SDK struct {
	*tokensdk.SDK
}

func NewSDK(registry node.Registry) *SDK {
	return &SDK{SDK: tokensdk.NewFrom(orionsdk.NewFrom(fabricsdk.NewFrom(viewsdk.NewSDK(registry))))}
}

func (p *SDK) FabricEnabled() bool {
	return p.ConfigService().GetBool("fabric.enabled")
}
func (p *SDK) OrionEnabled() bool { return p.ConfigService().GetBool("orion.enabled") }

func (p *SDK) Install() error {
	if p.FabricEnabled() {
		if err := p.Container().Provide(fabric.NewDriver, dig.Group("network-drivers")); err != nil {
			return err
		}
	}
	if p.OrionEnabled() {
		if err := p.Container().Provide(orion.NewDriver, dig.Group("network-drivers")); err != nil {
			return err
		}
	}
	err := errors.Join(
		p.Container().Provide(dlog.NewDriver, dig.Group("token-drivers")),
	)
	if err != nil {
		return err
	}

	return p.SDK.Install()
}
