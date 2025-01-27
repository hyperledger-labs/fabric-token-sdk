/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"

const (
	// X509Identity identifies an X509-based identity
	X509Identity identity.Type = "x509"
	// IdemixIdentity identifies an idemix identity
	IdemixIdentity identity.Type = "idemix"
)

const (
	// OwnerMSPID is the default MSP ID for the owner wallet
	OwnerMSPID = "OwnerMSPID"
	// IssuerMSPID is the default MSP ID for the issuer wallet
	IssuerMSPID = "IssuerMSPID"
	// AuditorMSPID is the default MSP ID for the auditor wallet
	AuditorMSPID = "AuditorMSPID"
	// CertifierMSPID is the default MSP ID for the certifier wallet
	CertifierMSPID = "CertifierMSPID"
)

// RoleToMSPID maps the role to the MSP ID
var RoleToMSPID = map[identity.RoleType]string{
	identity.OwnerRole:     OwnerMSPID,
	identity.IssuerRole:    IssuerMSPID,
	identity.AuditorRole:   AuditorMSPID,
	identity.CertifierRole: CertifierMSPID,
}
