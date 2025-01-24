/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
)

// RoleType is the role of an identity
type RoleType = driver.IdentityRoleType

const (
	// IssuerRole is the role of an issuer
	IssuerRole = driver.IssuerRole
	// AuditorRole is the role of an auditor
	AuditorRole = driver.AuditorRole
	// OwnerRole is the role of an owner
	OwnerRole = driver.OwnerRole
	// CertifierRole is the role of a certifier
	CertifierRole = driver.CertifierRole
)

var (
	RoleTypeStrings = driver.IdentityRoleStrings
)

type Info = driver.IdentityInfo

// Role is a container of long-term identities.
// A long-term identity is then used to construct a wallet.
type Role = driver.Role
