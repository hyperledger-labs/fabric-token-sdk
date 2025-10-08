/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fxdlog

import (
	"testing"

	dig2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/sdk/dig"
	fabricsdk "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk/dig"
	fabricx "github.com/hyperledger-labs/fabric-smart-client/platform/fabricx/sdk/dig"
	sdk "github.com/hyperledger-labs/fabric-smart-client/platform/view/sdk/dig"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/common/sdk/fdlog"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
	"github.com/stretchr/testify/assert"
)

func TestFabricWiring(t *testing.T) {
	assert.NoError(t, sdk.DryRunWiring(
		func(sdk dig2.SDK) *SDK {
			return NewFrom(fabricx.NewFrom(&fdlog.SDK{SDK: &tokensdk.SDK{SDK: &fabricsdk.SDK{SDK: sdk}}}))
		},
		sdk.WithBool("token.enabled", true),
		sdk.WithBool("fabric.enabled", true),
	))
}
