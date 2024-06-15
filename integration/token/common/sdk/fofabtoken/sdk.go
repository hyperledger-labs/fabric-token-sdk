/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fofabtoken

import (
	fabricsdk "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk/dig"
	orionsdk "github.com/hyperledger-labs/fabric-smart-client/platform/orion/sdk/dig"
	viewsdk "github.com/hyperledger-labs/fabric-smart-client/platform/view/sdk/dig"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/driver"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb/db/sql"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/driver/unity"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb/db/sql"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb/db/sql"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokenlockdb/db/sql"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/sql"
	_ "modernc.org/sqlite"
)

type SDK struct {
	*tokensdk.SDK
}

func NewSDK(registry sdk.Registry) *SDK {
	return &SDK{SDK: tokensdk.NewFrom(fabricsdk.NewFrom(orionsdk.NewFrom(viewsdk.NewSDK(registry))))}
}
