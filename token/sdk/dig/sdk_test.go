/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	"testing"

	dig2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/sdk/dig"
	fabricsdk "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk/dig"
	sdk "github.com/hyperledger-labs/fabric-smart-client/platform/view/sdk/dig"
	"github.com/test-go/testify/require"
)

func TestFabricWiring(t *testing.T) {
	require.NoError(t, sdk.DryRunWiring(
		func(sdk dig2.SDK) *SDK { return NewFrom(fabricsdk.NewFrom(sdk)) },
		sdk.WithBool("token.enabled", true),
		sdk.WithBool("fabric.enabled", true),
	))
}
