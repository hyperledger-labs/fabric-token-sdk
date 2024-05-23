/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ffabtoken

import (
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/driver"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb/db/sql"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/dummy"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/driver"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb/db/sql"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
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
