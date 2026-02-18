/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// Deserializer deserializes verifiers associated with issuers, owners, and auditors
type Deserializer struct {
	identityType         string
	auditorDeserializer  driver.VerifierDeserializer
	ownerDeserializer    driver.VerifierDeserializer
	issuerDeserializer   driver.VerifierDeserializer
	auditMatcherProvider driver.AuditMatcherProvider
	recipientExtractor   driver.RecipientExtractor
}

func NewDeserializer(
	identityType string,
	auditorDeserializer driver.VerifierDeserializer,
	ownerDeserializer driver.VerifierDeserializer,
	issuerDeserializer driver.VerifierDeserializer,
	auditMatcherProvider driver.AuditMatcherProvider,
	recipientExtractor driver.RecipientExtractor,
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

func (d *Deserializer) GetOwnerVerifier(ctx context.Context, id driver.Identity) (driver.Verifier, error) {
	return d.ownerDeserializer.DeserializeVerifier(ctx, id)
}

func (d *Deserializer) GetIssuerVerifier(ctx context.Context, id driver.Identity) (driver.Verifier, error) {
	return d.issuerDeserializer.DeserializeVerifier(ctx, id)
}

func (d *Deserializer) GetAuditorVerifier(ctx context.Context, id driver.Identity) (driver.Verifier, error) {
	return d.auditorDeserializer.DeserializeVerifier(ctx, id)
}

func (d *Deserializer) Recipients(id driver.Identity) ([]driver.Identity, error) {
	return d.recipientExtractor.Recipients(id)
}

func (d *Deserializer) GetAuditInfoMatcher(ctx context.Context, owner driver.Identity, auditInfo []byte) (driver.Matcher, error) {
	return d.auditMatcherProvider.GetAuditInfoMatcher(ctx, owner, auditInfo)
}

func (d *Deserializer) MatchIdentity(ctx context.Context, id driver.Identity, ai []byte) error {
	return d.auditMatcherProvider.MatchIdentity(ctx, id, ai)
}

func (d *Deserializer) GetAuditInfo(ctx context.Context, id driver.Identity, p driver.AuditInfoProvider) ([]byte, error) {
	return d.auditMatcherProvider.GetAuditInfo(ctx, id, p)
}
