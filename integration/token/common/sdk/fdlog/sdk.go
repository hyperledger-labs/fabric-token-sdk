/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fdlog

import (
	"errors"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	dlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/driver"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
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
		p.Container().Provide(dlog.NewDriver, dig.Group("token-drivers")),
	)
	if err != nil {
		return err
	}

	return p.SDK.Install()
}
