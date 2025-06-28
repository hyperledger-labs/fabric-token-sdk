/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fall

import (
	"errors"

	dig2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/sdk/dig"
	digutils "github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/dig"
	fabtoken "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/driver"
	dlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/driver"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	"go.uber.org/dig"
)

type SDK struct {
	dig2.SDK
}

func NewSDK(registry digutils.Registry) *SDK {
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

	return p.SDK.Install()
}
