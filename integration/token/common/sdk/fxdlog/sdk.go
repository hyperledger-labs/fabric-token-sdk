/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fxdlog

import (
	"errors"

	common "github.com/hyperledger-labs/fabric-smart-client/platform/common/sdk/dig"
	digutils "github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/state"
	fabricxsdk "github.com/hyperledger-labs/fabric-smart-client/platform/fabricx/sdk/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services"
	dlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/driver"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/pp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/tms"
	"go.uber.org/dig"
)

type SDK struct {
	common.SDK
}

func NewSDK(registry services.Registry) *SDK {
	return &SDK{SDK: tokensdk.NewFrom(fabricxsdk.NewSDK(registry))}
}

func NewFrom(sdk common.SDK) *SDK {
	return &SDK{SDK: sdk}
}

func (p *SDK) FabricEnabled() bool {
	return p.ConfigService().GetBool("fabric.enabled")
}

func (p *SDK) Install() error {
	if !p.FabricEnabled() {
		return p.SDK.Install()
	}

	err := errors.Join(
		// token driver
		p.Container().Provide(dlog.NewDriver, dig.Group("token-drivers")),

		// fabricx network driver
		p.Container().Provide(fabricx.NewDriver, dig.Group("network-drivers")),
		p.Container().Provide(tms.NewSubmitterFromFNS, dig.As(new(tms.Submitter))),
		p.Container().Provide(tms.NewTMSDeployerService, dig.As(new(tms.DeployerService))),
		p.Container().Provide(pp.NewPublicParametersService),
		p.Container().Provide(digutils.Identity[*pp.PublicParametersService](), dig.As(new(pp.Loader))),
	)
	if err != nil {
		return err
	}

	if err := p.SDK.Install(); err != nil {
		return err
	}
	return errors.Join(
		digutils.Register[state.VaultService](p.Container()),
		digutils.Register[tms.DeployerService](p.Container()),
		digutils.Register[pp.Loader](p.Container()),
	)
}
