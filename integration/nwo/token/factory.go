/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	fabric2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
)

type platformFactory struct{}

func NewPlatformFactory() *platformFactory {
	return &platformFactory{}
}

func (p *platformFactory) Name() string {
	return "token"
}

func (p *platformFactory) New(ctx api.Context, t api.Topology, builder api.Builder) api.Platform {
	tp := NewPlatform(ctx, t, builder)
	tp.AddNetworkHandler(fabric2.TopologyName, fabric.NewNetworkHandler(tp))
	return tp
}
