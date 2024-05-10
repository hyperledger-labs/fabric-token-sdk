/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package deserializer

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/interop/htlc"
	"github.com/pkg/errors"
)

// VerifierDeserializer is the interface for verifiers' deserializer
// A verifier checks the validity of a signature against the identity
// associated with the verifier
type VerifierDeserializer interface {
	DeserializeVerifier(id view.Identity) (driver.Verifier, error)
}

// AuditMatcherProvider deserializes raw bytes into a matcher, which allows an auditor to match an identity to an enrollment ID
type AuditMatcherProvider interface {
	GetOwnerMatcher(raw []byte) (driver.Matcher, error)
	Match(id view.Identity, ai []byte) error
}

type RecipientExtractor interface {
	Recipients(id view.Identity) ([]view.Identity, error)
}

// Deserializer deserializes verifiers associated with issuers, owners, and auditors
type Deserializer struct {
	identityType             string
	auditorDeserializer      VerifierDeserializer
	ownerDeserializer        VerifierDeserializer
	issuerDeserializer       VerifierDeserializer
	auditMatcherDeserializer AuditMatcherProvider
	recipientExtractor       RecipientExtractor
}

func NewDeserializer(
	identityType string,
	auditorDeserializer VerifierDeserializer,
	ownerDeserializer VerifierDeserializer,
	issuerDeserializer VerifierDeserializer,
	auditMatcherDeserializer AuditMatcherProvider,
	recipientExtractor RecipientExtractor,
) *Deserializer {
	return &Deserializer{
		identityType:             identityType,
		auditorDeserializer:      auditorDeserializer,
		ownerDeserializer:        ownerDeserializer,
		issuerDeserializer:       issuerDeserializer,
		auditMatcherDeserializer: auditMatcherDeserializer,
		recipientExtractor:       recipientExtractor,
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

func (d *Deserializer) Recipients(id view.Identity) ([]view.Identity, error) {
	return d.recipientExtractor.Recipients(id)
}

// GetOwnerMatcher is not needed in fabtoken, as identities are in the clear
func (d *Deserializer) GetOwnerMatcher(raw []byte) (driver.Matcher, error) {
	return d.auditMatcherDeserializer.GetOwnerMatcher(raw)
}

func (d *Deserializer) Match(id view.Identity, ai []byte) error {
	return d.auditMatcherDeserializer.Match(id, ai)
}

func (d *Deserializer) GetOwnerAuditInfo(raw []byte, p driver.AuditInfoProvider) ([][]byte, error) {
	auditInfo, err := htlc.GetOwnerAuditInfo(raw, p)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for recipient identity [%s]", view.Identity(raw).String())
	}
	return [][]byte{auditInfo}, nil
}

type AuditInfo interface {
	EnrollmentID() string
	RevocationHandle() string
}

type AuditInfoDeserializer[T AuditInfo] interface {
	DeserializeAuditInfo([]byte) (T, error)
}

// EIDRHDeserializer returns enrollment IDs behind the owners of token
type EIDRHDeserializer[T AuditInfo] struct {
	AuditInfoDeserializer AuditInfoDeserializer[T]
}

// NewEIDRHDeserializer returns an enrollmentService
func NewEIDRHDeserializer[T AuditInfo](AuditInfoDeserializer AuditInfoDeserializer[T]) *EIDRHDeserializer[T] {
	return &EIDRHDeserializer[T]{
		AuditInfoDeserializer: AuditInfoDeserializer,
	}
}

// GetEnrollmentID returns the enrollmentID associated with the identity matched to the passed auditInfo
func (e *EIDRHDeserializer[T]) GetEnrollmentID(identity view.Identity, auditInfo []byte) (string, error) {
	ai, err := e.getAuditInfo(auditInfo)
	if err != nil {
		return "", err
	}
	return ai.EnrollmentID(), nil
}

// GetRevocationHandler returns the recoatopn handle associated with the identity matched to the passed auditInfo
func (e *EIDRHDeserializer[T]) GetRevocationHandler(identity view.Identity, auditInfo []byte) (string, error) {
	ai, err := e.getAuditInfo(auditInfo)
	if err != nil {
		return "", err
	}
	return ai.RevocationHandle(), nil
}

func (e *EIDRHDeserializer[T]) getAuditInfo(auditInfo []byte) (T, error) {
	var zeroAuditInfo T
	if len(auditInfo) == 0 {
		return zeroAuditInfo, nil
	}

	// Try to unmarshal it as ScriptInfo
	si := &htlc.ScriptInfo{}
	err := json.Unmarshal(auditInfo, si)
	if err == nil && (len(si.Sender) != 0 || len(si.Recipient) != 0) {
		if len(si.Recipient) != 0 {
			ai, err := e.AuditInfoDeserializer.DeserializeAuditInfo(si.Recipient)
			if err != nil {
				return zeroAuditInfo, errors.Wrapf(err, "failed unamrshalling audit info [%s]", auditInfo)
			}
			return ai, nil
		}

		return zeroAuditInfo, nil
	}

	ai, err := e.AuditInfoDeserializer.DeserializeAuditInfo(auditInfo)
	if err != nil {
		return zeroAuditInfo, errors.Wrapf(err, "failed unamrshalling audit info [%s]", auditInfo)
	}
	return ai, nil
}
