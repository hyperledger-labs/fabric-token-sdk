/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/config"
)

type Config interface {
	// CacheSizeForOwnerID returns the cache size to be used for the given owner wallet.
	// If not defined, the function returns -1
	CacheSizeForOwnerID(id string) int
	TranslatePath(path string) string
	IdentitiesForRole(role driver.IdentityRole) ([]*config.Identity, error)
}
