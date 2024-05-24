/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fodlog

import (
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/driver"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb/db/sql"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/driver/unity"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb/db/sql"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb/db/sql"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/sql"
	_ "modernc.org/sqlite"
)

type SDK struct {
	*sdk.SDK
}

func NewSDK(registry sdk.Registry) *SDK {
	return &SDK{SDK: sdk.NewSDK(registry)}
}
