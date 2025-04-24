/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
)

type (
	WalletID              = identity.WalletID
	IdentityConfiguration = driver.IdentityConfiguration
	WalletStore           = driver2.WalletStore
	IdentityStore         = driver2.IdentityStore
)
