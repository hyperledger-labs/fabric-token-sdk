/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type WalletRegistry interface {
	WalletIDs() ([]string, error)
	RegisterIdentity(config driver.IdentityConfiguration) error
	Lookup(id driver.WalletLookupID) (driver.Wallet, IdentityInfo, string, error)
	RegisterWallet(id string, wallet driver.Wallet) error
	BindIdentity(identity driver.Identity, eID string, wID string, meta any) error
	ContainsIdentity(i driver.Identity, id string) bool
	GetIdentityMetadata(identity driver.Identity, wID string, meta any) error
}
