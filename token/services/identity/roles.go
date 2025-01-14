/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// Role is a container of long-term identities.
// A long-term identity is then used to construct a wallet.
type Role interface {
	// ID returns the identifier of this role
	ID() driver.IdentityRole
	// MapToIdentity returns the long-term identity and its identifier for the given index.
	// The index can be an identity or a label (string).
	MapToIdentity(v driver.WalletLookupID) (driver.Identity, string, error)
	// GetIdentityInfo returns the long-term identity info associated to the passed id
	GetIdentityInfo(id string) (driver.IdentityInfo, error)
	// RegisterIdentity registers the given identity
	RegisterIdentity(config driver.IdentityConfiguration) error
	// IdentityIDs returns the identifiers contained in this role
	IdentityIDs() ([]string, error)
}

// Roles is a map of Role, one for each identity role
type Roles map[driver.IdentityRole]Role

// NewRoles returns a new Roles maps
func NewRoles() Roles {
	return make(Roles)
}

// Register associates an instance of Role to a given identifier
func (m Roles) Register(usage driver.IdentityRole, role Role) {
	m[usage] = role
}
