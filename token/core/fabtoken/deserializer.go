/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/owner"
	"github.com/pkg/errors"
)

// VerifierDES is the interface for verifiers' deserializer
// A verifier checks the validity of a signature against the identity
// associated with the verifier
type VerifierDES interface {
	DeserializeVerifier(id view.Identity) (driver.Verifier, error)
}

// Deserializer deserializes verifiers associated with issuers, owners, and auditors
type Deserializer struct {
	auditorDeserializer VerifierDES
	ownerDeserializer   VerifierDES
	issuerDeserializer  VerifierDES
}

// NewDeserializer returns a deserializer
func NewDeserializer() *Deserializer {
	return &Deserializer{
		auditorDeserializer: &x509.MSPIdentityDeserializer{},
		issuerDeserializer:  &x509.MSPIdentityDeserializer{},
		ownerDeserializer:   interop.NewDeserializer(owner.NewTypedIdentityDeserializer(&x509.MSPIdentityDeserializer{})),
	}
}

// GetOwnerVerifier deserializes the verifier for the passed owner identity
func (d *Deserializer) GetOwnerVerifier(id view.Identity) (driver.Verifier, error) {
	return d.ownerDeserializer.DeserializeVerifier(id)
}

// GetIssuerVerifier deserializes the verifier for the passed issuer identity
func (d *Deserializer) GetIssuerVerifier(id view.Identity) (driver.Verifier, error) {
	return d.issuerDeserializer.DeserializeVerifier(id)
}

// GetAuditorVerifier deserializes the verifier for the passed auditor identity
func (d *Deserializer) GetAuditorVerifier(id view.Identity) (driver.Verifier, error) {
	return d.auditorDeserializer.DeserializeVerifier(id)
}

// GetOwnerMatcher is not needed in fabtoken, as identities are in the clear
func (d *Deserializer) GetOwnerMatcher(raw []byte) (driver.Matcher, error) {
	ai := &x509.AuditInfo{}
	err := ai.FromBytes(raw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal")
	}
	return &x509.AuditInfoDeserializer{CommonName: string(ai.EnrollmentId)}, nil
}

// EnrollmentService returns enrollment IDs behind the owners of token
type EnrollmentService struct {
}

// NewEnrollmentIDDeserializer returns an enrollmentService
func NewEnrollmentIDDeserializer() *EnrollmentService {
	return &EnrollmentService{}
}

func (e *EnrollmentService) GetEnrollmentID(auditInfo []byte) (string, error) {
	if len(auditInfo) == 0 {
		return "", nil
	}

	// Try to unmarshal it as ScriptInfo
	si := &interop.ScriptInfo{}
	err := json.Unmarshal(auditInfo, si)
	if err == nil && (len(si.Sender) != 0 || len(si.Recipient) != 0) {
		if len(si.Recipient) != 0 {
			ai := &x509.AuditInfo{}
			if err := ai.FromBytes(si.Recipient); err != nil {
				return "", errors.Wrapf(err, "failed unmarshalling audit info [%s]", auditInfo)
			}
			return ai.EnrollmentId, nil
		}

		return "", nil
	}

	ai := &x509.AuditInfo{}
	if err := ai.FromBytes(auditInfo); err != nil {
		return "", errors.Wrapf(err, "failed unmarshalling audit info [%s]", auditInfo)
	}
	return ai.EnrollmentId, nil
}

// GetRevocationHandler returns the revocation handler associated with the identity matched to the passed auditInfo
func (e *EnrollmentService) GetRevocationHandler(auditInfo []byte) (string, error) {
	if len(auditInfo) == 0 {
		return "", nil
	}

	// Try to unmarshal it as ScriptInfo
	si := &interop.ScriptInfo{}
	err := json.Unmarshal(auditInfo, si)
	if err == nil && (len(si.Sender) != 0 || len(si.Recipient) != 0) {
		if len(si.Recipient) != 0 {
			ai := &x509.AuditInfo{}
			if err := ai.FromBytes(si.Recipient); err != nil {
				return "", errors.Wrapf(err, "failed unamrshalling audit info [%s]", auditInfo)
			}
			return ai.RevocationHandle, nil
		}

		return "", nil
	}

	ai := &x509.AuditInfo{}
	if err := ai.FromBytes(auditInfo); err != nil {
		return "", errors.Wrapf(err, "failed unmarshalling audit info [%s]", auditInfo)
	}
	return ai.RevocationHandle, nil
}
