/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ffabtoken

import (
	"errors"

	dig2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/sdk/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services"
	fabtoken "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	"go.uber.org/dig"
)

type SDK struct {
	dig2.SDK
}

func NewSDK(registry services.Registry) *SDK {
	return &SDK{SDK: tokensdk.NewSDK(registry)}
}

func NewFrom(sdk dig2.SDK) *SDK {
	return &SDK{SDK: sdk}
}

func (p *SDK) Install() error {
	err := errors.Join(
		sdk.RegisterTokenDriverDependencies(p.Container()),
		p.Container().Provide(fabric.NewGenericDriver, dig.Group("network-drivers")),
		p.Container().Provide(fabtoken.NewDriver, dig.Group("token-drivers")),
	)
	if err != nil {
		return err
	}

	return p.SDK.Install()
}
