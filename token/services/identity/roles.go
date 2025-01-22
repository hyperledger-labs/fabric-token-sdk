/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
)

// RoleType is the role of an identity
type RoleType = driver.IdentityRoleType

// Role is a container of long-term identities.
// A long-term identity is then used to construct a wallet.
type Role = idriver.Role

// Roles is a map of Role, one for each identity role
type Roles map[RoleType]Role

// NewRoles returns a new Roles maps
func NewRoles() Roles {
	return make(Roles)
}

// Register associates an instance of Role to a given identifier
func (m Roles) Register(usage RoleType, role Role) {
	m[usage] = role
}
