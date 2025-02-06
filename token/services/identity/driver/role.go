/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import "github.com/hyperledger-labs/fabric-token-sdk/token/driver"

// IdentityRoleType is the role of an identity
type IdentityRoleType int

const (
	// IssuerRole is the role of an issuer
	IssuerRole = iota
	// AuditorRole is the role of an auditor
	AuditorRole
	// OwnerRole is the role of an owner
	OwnerRole
	// CertifierRole is the role of a certifier
	CertifierRole
)

var (
	IdentityRoleStrings = map[IdentityRoleType]string{
		IssuerRole:    "issuer",
		AuditorRole:   "auditor",
		OwnerRole:     "owner",
		CertifierRole: "certifier",
	}
)

// IdentityInfo models a long-term identity inside the Identity Provider.
// An identity has an identifier (ID) and an Enrollment ID, unique identifier.
// An identity can be remote, meaning that the corresponding secret key is remotely available.
type IdentityInfo interface {
	// ID returns the identifier of the Identity
	ID() string
	// EnrollmentID returns the enrollment ID of the Identity
	EnrollmentID() string
	// Remote is true if this identity info refers to an identify whose corresponding secret key is not known, it is external/remote
	Remote() bool
	// Get returns the identity and it is audit info.
	// Get might return a different identity at each call depending on the implementation.
	Get() (Identity, []byte, error)
	// Anonymous is true if this identity supports anonymity
	Anonymous() bool
}

type (
	// WalletLookupID defines the type of identifiers that can be used to retrieve a given wallet.
	// It can be a string, as the name of the wallet, or an identity contained in that wallet.
	// Ultimately, it is the token driver to decide which types are allowed.
	WalletLookupID        = driver.WalletLookupID
	Identity              = driver.Identity
	IdentityConfiguration = driver.IdentityConfiguration
)

// Role is a container of long-term identities.
// A long-term identity is then used to construct a wallet.
type Role interface {
	// ID returns the identifier of this role
	ID() IdentityRoleType
	// MapToIdentity returns the long-term identity and its identifier for the given index.
	// The index can be an identity or a label (string).
	MapToIdentity(v WalletLookupID) (Identity, string, error)
	// GetIdentityInfo returns the long-term identity info associated to the passed id
	GetIdentityInfo(id string) (IdentityInfo, error)
	// RegisterIdentity registers the given identity
	RegisterIdentity(config IdentityConfiguration) error
	// IdentityIDs returns the identifiers contained in this role
	IdentityIDs() ([]string, error)
}
