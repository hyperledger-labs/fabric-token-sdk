/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wallet

import (
	"context"
	"encoding/json"

	tdriver "github.com/LFDT-Panurus/panurus/token/driver"
	"github.com/LFDT-Panurus/panurus/token/services/identity"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

// Validation errors
var (
	ErrEmptyIdentity           = errors.New("empty identity")
	ErrEmptyAuditInfo          = errors.New("empty audit info")
	ErrIdentityTooShort        = errors.New("identity too short")
	ErrIdentityTooLarge        = errors.New("identity too large")
	ErrEmptyRawIdentity        = errors.New("empty raw identity data")
	ErrAuditInfoTooLarge       = errors.New("audit info too large")
	ErrInvalidJSON             = errors.New("audit info is not valid JSON")
	ErrEmptyEnrollmentID       = errors.New("empty enrollment ID")
	ErrEnrollmentIDTooLong     = errors.New("enrollment ID too long")
	ErrEmptyRevocationHandle   = errors.New("empty revocation handle")
	ErrRevocationHandleTooLong = errors.New("revocation handle too long")
)

const (
	// MaxEnrollmentIDLength defines the maximum allowed length for enrollment IDs
	MaxEnrollmentIDLength = 256
	// MaxRevocationHandleLength defines the maximum allowed length for revocation handles
	MaxRevocationHandleLength = 512
	// MinIdentityLength defines the minimum allowed length for identity data
	MinIdentityLength = 10
	// MaxIdentityLength defines the maximum allowed length for identity data (prevents DoS).
	// Set generously so legitimate identities are never rejected: composite identities
	// (MultiSig, Policy) aggregate several inner identities and X509 chains / Idemix proofs
	// can run to several KB. The bound exists only to reject pathological multi-MB blobs.
	MaxIdentityLength = 1 << 20 // 1 MiB
	// MaxAuditInfoLength defines the maximum allowed length for audit info (prevents DoS)
	MaxAuditInfoLength = 50000
)

// validateBasicStructure performs nil and empty checks on RecipientData
func validateBasicStructure(data *tdriver.RecipientData) error {
	if data == nil {
		return ErrNilRecipientData
	}
	if len(data.Identity) == 0 {
		return ErrEmptyIdentity
	}
	if len(data.Identity) < MinIdentityLength {
		return errors.Wrapf(ErrIdentityTooShort, "identity is %d bytes (min %d)", len(data.Identity), MinIdentityLength)
	}
	if len(data.Identity) > MaxIdentityLength {
		return errors.Wrapf(ErrIdentityTooLarge, "identity is %d bytes (max %d)", len(data.Identity), MaxIdentityLength)
	}
	if len(data.AuditInfo) == 0 {
		return ErrEmptyAuditInfo
	}
	if len(data.AuditInfo) > MaxAuditInfoLength {
		return errors.Wrapf(ErrAuditInfoTooLarge, "audit info is %d bytes (max %d)", len(data.AuditInfo), MaxAuditInfoLength)
	}

	return nil
}

// validateJSONStructure validates that audit info is valid JSON
func validateJSONStructure(auditInfo []byte) error {
	if !json.Valid(auditInfo) {
		return ErrInvalidJSON
	}

	return nil
}

// validateIdentityStructure validates the decoded typed identity: the type must be
// recognized and the raw identity payload must not be empty.
func validateIdentityStructure(typedID *identity.TypedIdentity) error {
	// Validate the identity type is recognized
	if !isValidIdentityType(typedID.Type) {
		return errors.Errorf("unknown identity type: %d", typedID.Type)
	}

	// Validate raw identity data is not empty
	if len(typedID.Identity) == 0 {
		return ErrEmptyRawIdentity
	}

	return nil
}

// validateEnrollmentID validates the enrollment ID is present and within length bounds.
// The caller is responsible for skipping composite identity types that have no enrollment ID.
func (s *Service) validateEnrollmentID(ctx context.Context, data *tdriver.RecipientData) error {
	// Extract enrollment ID
	eid, err := s.IdentityProvider.GetEnrollmentID(ctx, data.Identity, data.AuditInfo)
	if err != nil {
		return errors.Wrap(err, "failed to extract enrollment ID")
	}

	// Check not empty
	if len(eid) == 0 {
		return ErrEmptyEnrollmentID
	}

	// Check length
	if len(eid) > MaxEnrollmentIDLength {
		return errors.Wrapf(ErrEnrollmentIDTooLong, "enrollment ID is %d bytes (max %d)", len(eid), MaxEnrollmentIDLength)
	}

	return nil
}

// validateRevocationHandle validates the revocation handle is present and within length bounds.
// The caller is responsible for skipping composite identity types that have no revocation handle.
func (s *Service) validateRevocationHandle(ctx context.Context, data *tdriver.RecipientData) error {
	// Extract revocation handle
	rh, err := s.IdentityProvider.GetRevocationHandler(ctx, data.Identity, data.AuditInfo)
	if err != nil {
		return errors.Wrap(err, "failed to extract revocation handle")
	}

	// Check not empty
	if len(rh) == 0 {
		return ErrEmptyRevocationHandle
	}

	// Check reasonable length
	if len(rh) > MaxRevocationHandleLength {
		return errors.Wrapf(ErrRevocationHandleTooLong, "revocation handle is %d bytes (max %d)", len(rh), MaxRevocationHandleLength)
	}

	return nil
}

// isValidIdentityType checks if the identity type is one of the recognized types
func isValidIdentityType(t tdriver.IdentityType) bool {
	switch t {
	case tdriver.IdemixIdentityType,
		tdriver.X509IdentityType,
		tdriver.IdemixNymIdentityType,
		tdriver.HTLCScriptIdentityType,
		tdriver.MultiSigIdentityType,
		tdriver.PolicyIdentityType:
		return true
	default:
		return false
	}
}

// isCompositeIdentityType checks if the identity type is a composite type
// Composite types (MultiSig, HTLC, Policy) don't have enrollment IDs or revocation handles
func isCompositeIdentityType(t tdriver.IdentityType) bool {
	switch t {
	case tdriver.MultiSigIdentityType,
		tdriver.HTLCScriptIdentityType,
		tdriver.PolicyIdentityType:
		return true
	default:
		return false
	}
}
