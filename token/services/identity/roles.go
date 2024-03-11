/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// Role is a container of long-term identities.
// A long-term identity is then used to construct a wallet.
type Role interface {
	// MapToID returns the long-term identity and its identifier for the given index.
	// The index can be an identity or a label (string).
	MapToID(v interface{}) (view.Identity, string, error)
	// GetIdentityInfo returns the long-term identity info associated to the passed id
	GetIdentityInfo(id string) driver.IdentityInfo
	// RegisterIdentity registers the given identity
	RegisterIdentity(id string, path string) error
	// IDs returns the identifiers contained in this role
	IDs() ([]string, error)
	// Reload the roles with the respect to the passed public parameters
	Reload(pp driver.PublicParameters) error
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

func (m Roles) Reload(pp driver.PublicParameters) error {
	logger.Debugf("reload roles...")
	for roleID, role := range m {
		logger.Debugf("reload role [%d]...", roleID)
		if err := role.Reload(pp); err != nil {
			return err
		}
		logger.Debugf("reload role [%d]...done", roleID)
	}
	logger.Debugf("reload roles...done")
	return nil
}
