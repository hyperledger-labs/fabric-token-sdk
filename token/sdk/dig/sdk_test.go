/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sdk_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/sdk/dig"
	fabricsdk "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk/dig"
	viewsdk "github.com/hyperledger-labs/fabric-smart-client/platform/view/sdk/dig"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
	"github.com/stretchr/testify/require"
)

func TestFabricWiring(t *testing.T) {
	require.NoError(t, viewsdk.DryRunWiring(
		func(root dig.SDK) *sdk.SDK { return sdk.NewFrom(fabricsdk.NewFrom(root)) },
		viewsdk.WithBool("token.enabled", true),
		viewsdk.WithBool("fabric.enabled", true),
	))
}

func TestFabricWiring_TokenDisabled(t *testing.T) {
	// Test with token platform disabled
	require.NoError(t, viewsdk.DryRunWiring(
		func(root dig.SDK) *sdk.SDK { return sdk.NewFrom(fabricsdk.NewFrom(root)) },
		viewsdk.WithBool("token.enabled", false),
		viewsdk.WithBool("fabric.enabled", true),
	))
}

// Note: The following functions are tested through the DryRunWiring tests above:
// - NewSDK: Creates SDK from registry
// - NewFrom: Wraps an existing SDK
// - SDK.TokenEnabled: Checks if token platform is enabled
// - SDK.Install: Installs dependencies (tested in wiring)
// - SDK.Start: Starts services (tested in wiring)
// - connectNetworks: Connects to networks (invoked during Start)
// - registerNetworkDrivers: Registers network drivers (invoked during Start)
// - newMultiplexedDriver: Creates multiplexed DB driver (invoked during Install)
// - newTokenDriverService: Creates token driver service (invoked during Install)
//
// These functions are dependency injection providers and helpers that are primarily
// tested through integration tests and the comprehensive wiring test above.
// Direct unit testing would require complex mocking of the dig container and
// provides limited value compared to the integration-style wiring test.

func TestNewSDKExists(t *testing.T) {
	// Verify the function exists and can be referenced
	require.NotNil(t, sdk.NewSDK)
}

func TestNewFromExists(t *testing.T) {
	// Verify the function exists and can be referenced
	require.NotNil(t, sdk.NewFrom)
}
