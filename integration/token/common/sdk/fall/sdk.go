/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fall

import (
	"context"
	"errors"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	dig2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/sdk/dig"
	digutils "github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/weaver"
	fabtoken "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/driver"
	fabric3 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/driver/interop/state/fabric"
	dlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/driver"
	fabric4 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/driver/interop/state/fabric"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/state"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/state/driver"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/state/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	"go.uber.org/dig"
)

type SDK struct {
	dig2.SDK
}

func NewSDK(registry node.Registry) *SDK {
	return &SDK{SDK: tokensdk.NewSDK(registry)}
}

func NewFrom(sdk dig2.SDK) *SDK {
	return &SDK{SDK: sdk}
}

func (p *SDK) Install() error {
	err := errors.Join(
		p.Container().Provide(fabric.NewGenericDriver, dig.Group("network-drivers")),
		p.Container().Provide(fabtoken.NewDriver, dig.Group("token-drivers")),
		p.Container().Provide(dlog.NewDriver, dig.Group("token-drivers")),
	)
	if err != nil {
		return err
	}

	fabricEnabled := p.ConfigService().GetBool("fabric.enabled")
	if fabricEnabled {
		err := errors.Join(
			// weaver
			p.Container().Provide(weaver.NewProvider, dig.As(new(fabric2.RelayProvider))),
			// state provider
			p.Container().Provide(state.NewServiceProvider),
			p.Container().Provide(fabric3.NewStateDriver, dig.Group("fabric-ssp-state-drivers")),
			p.Container().Provide(fabric4.NewStateDriver, dig.Group("fabric-ssp-state-drivers")),
			p.Container().Provide(fabric2.NewSSPDriver, dig.Group("ssp-drivers")),
			p.Container().Provide(pledge.NewVaultStore),
		)
		if err != nil {
			return err
		}
	}

	if err := p.SDK.Install(); err != nil {
		return err
	}

	if fabricEnabled {
		return errors.Join(
			digutils.Register[fabric2.RelayProvider](p.Container()),
			digutils.Register[*pledge.VaultStore](p.Container()),
			digutils.Register[*state.ServiceProvider](p.Container()),
		)
	}

	return nil
}

func (p *SDK) Start(ctx context.Context) error {
	if err := p.SDK.Start(ctx); err != nil {
		return err
	}

	fabricEnabled := p.ConfigService().GetBool("fabric.enabled")
	if fabricEnabled {
		return errors.Join(
			p.Container().Invoke(registerInteropStateDrivers),
		)
	}
	return nil
}

func registerInteropStateDrivers(in struct {
	dig.In
	StateServiceProvider *state.ServiceProvider
	Drivers              []driver.NamedSSPDriver `group:"ssp-drivers"`
}) {
	for _, d := range in.Drivers {
		in.StateServiceProvider.RegisterDriver(d)
	}
}
