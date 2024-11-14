/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ofabtoken

import (
	"errors"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	orionsdk "github.com/hyperledger-labs/fabric-smart-client/platform/orion/sdk/dig"
	fabtoken "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/driver"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
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
		p.Container().Provide(orion.NewOrionDriver, dig.Group("network-drivers")),
		p.Container().Provide(fabtoken.NewDriver, dig.Group("token-drivers")),
	)
	if err != nil {
		return err
	}

	return p.SDK.Install()
}
