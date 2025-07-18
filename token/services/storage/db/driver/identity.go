/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
)

type (
	WalletID              = driver2.WalletID
	IdentityConfiguration = driver.IdentityConfiguration
	WalletStore           = driver2.WalletStoreService
	IdentityStore         = driver2.IdentityStoreService
)
