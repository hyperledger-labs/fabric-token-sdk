/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type MatcherDeserializer interface {
	GetAuditInfoMatcher(owner driver.Identity, auditInfo []byte) (driver.Matcher, error)
}

// VerifierDeserializer is the interface for verifiers' deserializer.
// A verifier checks the validity of a signature against the identity associated with the verifier
type VerifierDeserializer interface {
	DeserializeVerifier(id driver.Identity) (driver.Verifier, error)
}

// AuditMatcherProvider provides audit related deserialization functionalities
type AuditMatcherProvider interface {
	MatcherDeserializer
	MatchIdentity(id driver.Identity, ai []byte) error
	GetAuditInfo(id driver.Identity, p driver.AuditInfoProvider) ([]byte, error)
}

// RecipientExtractor extracts the recipients from an identity
type RecipientExtractor interface {
	Recipients(id driver.Identity) ([]driver.Identity, error)
}

// Deserializer deserializes verifiers associated with issuers, owners, and auditors
type Deserializer struct {
	identityType         string
	auditorDeserializer  VerifierDeserializer
	ownerDeserializer    VerifierDeserializer
	issuerDeserializer   VerifierDeserializer
	auditMatcherProvider AuditMatcherProvider
	recipientExtractor   RecipientExtractor
}

func NewDeserializer(
	identityType string,
	auditorDeserializer VerifierDeserializer,
	ownerDeserializer VerifierDeserializer,
	issuerDeserializer VerifierDeserializer,
	auditMatcherProvider AuditMatcherProvider,
	recipientExtractor RecipientExtractor,
) *Deserializer {
	return &Deserializer{
		identityType:         identityType,
		auditorDeserializer:  auditorDeserializer,
		ownerDeserializer:    ownerDeserializer,
		issuerDeserializer:   issuerDeserializer,
		auditMatcherProvider: auditMatcherProvider,
		recipientExtractor:   recipientExtractor,
	}
}

func (d *Deserializer) GetOwnerVerifier(id driver.Identity) (driver.Verifier, error) {
	return d.ownerDeserializer.DeserializeVerifier(id)
}

func (d *Deserializer) GetIssuerVerifier(id driver.Identity) (driver.Verifier, error) {
	return d.issuerDeserializer.DeserializeVerifier(id)
}

func (d *Deserializer) GetAuditorVerifier(id driver.Identity) (driver.Verifier, error) {
	return d.auditorDeserializer.DeserializeVerifier(id)
}

func (d *Deserializer) Recipients(id driver.Identity) ([]driver.Identity, error) {
	return d.recipientExtractor.Recipients(id)
}

func (d *Deserializer) GetAuditInfoMatcher(owner driver.Identity, auditInfo []byte) (driver.Matcher, error) {
	return d.auditMatcherProvider.GetAuditInfoMatcher(owner, auditInfo)
}

func (d *Deserializer) MatchIdentity(id driver.Identity, ai []byte) error {
	return d.auditMatcherProvider.MatchIdentity(id, ai)
}

func (d *Deserializer) GetAuditInfo(id driver.Identity, p driver.AuditInfoProvider) ([]byte, error) {
	return d.auditMatcherProvider.GetAuditInfo(id, p)
}
