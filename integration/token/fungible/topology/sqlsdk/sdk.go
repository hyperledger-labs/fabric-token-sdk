/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlsdk

import (
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/driver"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/driver"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/dummy"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/interactive"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/badger"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/sql"
	_ "modernc.org/sqlite"
)

type Registry interface {
	GetService(v interface{}) (interface{}, error)

	RegisterService(service interface{}) error
}

type SDK struct {
	*sdk.SDK
}

func NewSDK(registry Registry) *SDK {
	return &SDK{SDK: sdk.NewSDK(registry)}
}
