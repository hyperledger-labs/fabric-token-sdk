/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"context"

	tdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// IdentityType identifies the type of identity.
// It is an alias for tdriver.IdentityType and is used by deserializers to choose the correct
// decoding logic for different identity representations.
type IdentityType = tdriver.IdentityType

// TypedSignerDeserializer converts a raw byte representation into a concrete
// Signer for a given IdentityType.
// Implementations should validate the raw data and return an error on invalid input or decoding failure.
type TypedSignerDeserializer interface {
	// DeserializeSigner deserializes the provided raw bytes into a tdriver.Signer
	// appropriate for the supplied identity type.
	// The context may carry ancillary information required for deserialization.
	DeserializeSigner(ctx context.Context, typ IdentityType, raw []byte) (tdriver.Signer, error)
}

// AuditInfo represents the audit-related information for an identity.
// It exposes the enrollment id and the revocation handle necessary for audit
// and revocation operations.
type AuditInfo interface {
	// EnrollmentID returns the enrollment identifier associated with this audit info.
	EnrollmentID() string
	// RevocationHandle returns the revocation handle associated with this audit info.
	RevocationHandle() string
}

// AuditInfoDeserializer converts a raw byte representation into an AuditInfo instance.
// Implementations should validate and parse the raw bytes and return an error on failure.
//
//go:generate counterfeiter -o mock/aides.go -fake-name AuditInfoDeserializer . AuditInfoDeserializer
type AuditInfoDeserializer interface {
	// DeserializeAuditInfo deserializes the provided raw bytes into an AuditInfo value.
	// The context may carry ancillary information required for deserialization.
	DeserializeAuditInfo(ctx context.Context, raw []byte) (AuditInfo, error)
}
