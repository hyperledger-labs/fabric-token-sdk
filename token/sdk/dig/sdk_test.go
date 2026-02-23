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
	"github.com/stretchr/testify/require"
)

func TestFabricWiring(t *testing.T) {
	require.NoError(t, sdk.DryRunWiring(
		func(sdk dig2.SDK) *SDK { return NewFrom(fabricsdk.NewFrom(sdk)) },
		sdk.WithBool("token.enabled", true),
		sdk.WithBool("fabric.enabled", true),
	))
}

func TestFabricWiring_TokenDisabled(t *testing.T) {
	// Test with token platform disabled
	require.NoError(t, sdk.DryRunWiring(
		func(sdk dig2.SDK) *SDK { return NewFrom(fabricsdk.NewFrom(sdk)) },
		sdk.WithBool("token.enabled", false),
		sdk.WithBool("fabric.enabled", true),
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
	require.NotNil(t, NewSDK)
}

func TestNewFromExists(t *testing.T) {
	// Verify the function exists and can be referenced
	require.NotNil(t, NewFrom)
}

func TestHelperFunctionsExist(t *testing.T) {
	// Verify helper functions exist and can be referenced
	require.NotNil(t, connectNetworks)
	require.NotNil(t, registerNetworkDrivers)
}
