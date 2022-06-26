/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// Mapper helps to find identity identifiers and retrieve the corresponding identities
type Mapper interface {
	// MapToID returns the identity identifier for the given argument
	MapToID(v interface{}) (view.Identity, string)
	// GetIdentityInfo returns the identity information for the given identity identifier
	GetIdentityInfo(id string) driver.IdentityInfo
	// RegisterIdentity registers the given identity
	RegisterIdentity(id string, path string) error
}

// Mappers is a map of Mapper, one for each identity role
type Mappers map[driver.IdentityRole]Mapper

// NewMappers returns a new Mappers
func NewMappers() Mappers {
	return make(Mappers)
}

// Put associated a mapper to a given identity role
func (m Mappers) Put(usage driver.IdentityRole, mapper Mapper) {
	m[usage] = mapper
}

// SetIssuerRole sets the issuer role mapper
func (m Mappers) SetIssuerRole(mapper Mapper) {
	m[driver.IssuerRole] = mapper
}

// SetAuditorRole sets the auditor role mapper
func (m Mappers) SetAuditorRole(mapper Mapper) {
	m[driver.AuditorRole] = mapper
}

// SetOwnerRole sets the owner role mapper
func (m Mappers) SetOwnerRole(mapper Mapper) {
	m[driver.OwnerRole] = mapper
}
