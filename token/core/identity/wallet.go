/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// Wallet helps to find identity identifiers and retrieve the corresponding identities
type Wallet interface {
	// MapToID returns the identity for the given argument
	MapToID(v interface{}) (view.Identity, string, error)
	// GetIdentityInfo returns the identity information for the given identity identifier
	GetIdentityInfo(id string) driver.IdentityInfo
	// RegisterIdentity registers the given identity
	RegisterIdentity(id string, path string) error
	// IDs returns the identifiers contains in this wallet
	IDs() ([]string, error)
	// Reload the wallets with the respect to the passed public parameters
	Reload(pp driver.PublicParameters) error
}

// Wallets is a map of Wallet, one for each identity role
type Wallets map[driver.IdentityRole]Wallet

// NewWallets returns a new Wallets maps
func NewWallets() Wallets {
	return make(Wallets)
}

// Put associates a wallet to a given identity role
func (m Wallets) Put(usage driver.IdentityRole, wallet Wallet) {
	m[usage] = wallet
}

func (m Wallets) Reload(pp driver.PublicParameters) error {
	logger.Debugf("reload wallets...")
	for role, wallet := range m {
		logger.Debugf("reload wallet for role [%d]...", role)
		if err := wallet.Reload(pp); err != nil {
			return err
		}
		logger.Debugf("reload wallet for role [%d]...done", role)
	}
	logger.Debugf("reload wallets...done")
	return nil
}
