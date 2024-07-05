/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	"testing"

	dig2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/sdk/dig"
	fabricsdk "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk/dig"
	orionsdk "github.com/hyperledger-labs/fabric-smart-client/platform/orion/sdk/dig"
	sdk "github.com/hyperledger-labs/fabric-smart-client/platform/view/sdk/dig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
)

func TestFabricWiring(t *testing.T) {
	assert.NoError(sdk.DryRunWiring(
		func(sdk dig2.SDK) *SDK { return NewFrom(fabricsdk.NewFrom(sdk)) },
		sdk.WithBool("token.enabled", true),
		sdk.WithBool("fabric.enabled", true),
	))
}

func TestOrionWiring(t *testing.T) {
	assert.NoError(sdk.DryRunWiring(
		func(sdk dig2.SDK) *SDK { return NewFrom(orionsdk.NewFrom(sdk)) },
		sdk.WithBool("token.enabled", true),
		sdk.WithBool("orion.enabled", true),
	))
}

func TestFabricOrionWiring(t *testing.T) {
	assert.NoError(sdk.DryRunWiring(
		func(sdk dig2.SDK) *SDK { return NewFrom(fabricsdk.NewFrom(orionsdk.NewFrom(sdk))) },
		sdk.WithBool("token.enabled", true),
		sdk.WithBool("fabric.enabled", true),
		sdk.WithBool("orion.enabled", true),
	))
}

func TestOrionFabricWiring(t *testing.T) {
	assert.NoError(sdk.DryRunWiring(
		func(sdk dig2.SDK) *SDK { return NewFrom(orionsdk.NewFrom(fabricsdk.NewFrom(sdk))) },
		sdk.WithBool("token.enabled", true),
		sdk.WithBool("orion.enabled", true),
		sdk.WithBool("fabric.enabled", true),
	))
}
