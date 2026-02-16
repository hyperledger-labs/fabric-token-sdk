/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorser

import (
	"testing"

	dig2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/sdk/dig"
	fabricsdk "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk/dig"
	sdk "github.com/hyperledger-labs/fabric-smart-client/platform/view/sdk/dig"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
	"github.com/test-go/testify/require"
)

func TestAllWiring(t *testing.T) {
	require.NoError(t, sdk.DryRunWiring(
		func(sdk dig2.SDK) *SDK { return NewFrom(tokensdk.NewFrom(fabricsdk.NewFrom(sdk))) },
		sdk.WithBool("token.enabled", true),
	))
}
