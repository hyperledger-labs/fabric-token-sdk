/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common/registry"
)

type platformFactory struct{}

func NewPlatformFactory() *platformFactory {
	return &platformFactory{}
}

func (p *platformFactory) Name() string {
	return "token"
}

func (p *platformFactory) New(registry *registry.Registry, builder integration.Builder) integration.Platform {
	return NewPlatform(registry)
}
