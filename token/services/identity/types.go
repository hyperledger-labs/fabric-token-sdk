/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
)

type (
	Keystore              = driver.Keystore
	StorageProvider       = driver.StorageProvider
	WalletID              = driver.WalletID
	ConfigurationIterator = driver.IdentityConfigurationIterator
	WalletDB              = driver.WalletDB
	IdentityDB            = driver.IdentityDB
)
