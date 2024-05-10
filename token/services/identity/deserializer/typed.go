/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package deserializer

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/pkg/errors"
)

type TypedVerifierDeserializer interface {
	DeserializeVerifier(typ string, raw []byte) (driver.Verifier, error)
	Recipients(id view.Identity, typ string, raw []byte) ([]view.Identity, error)
}

// AuditMatcherDeserializer deserializes raw bytes into a matcher, which allows an auditor to match an identity to an enrollment ID
type AuditMatcherDeserializer interface {
	GetOwnerMatcher(raw []byte) (driver.Matcher, error)
}

type TypedVerifierDeserializerMultiplex struct {
	deserializers            map[string]TypedVerifierDeserializer
	auditMatcherDeserializer AuditMatcherDeserializer
}

func NewTypedVerifierDeserializerMultiplex(auditMatcherDeserializer AuditMatcherDeserializer) *TypedVerifierDeserializerMultiplex {
	return &TypedVerifierDeserializerMultiplex{deserializers: map[string]TypedVerifierDeserializer{}, auditMatcherDeserializer: auditMatcherDeserializer}
}

func (v *TypedVerifierDeserializerMultiplex) AddTypedVerifierDeserializer(typ string, d TypedVerifierDeserializer) {
	v.deserializers[typ] = d
}

func (v *TypedVerifierDeserializerMultiplex) DeserializeVerifier(id view.Identity) (driver.Verifier, error) {
	si, err := identity.UnmarshalTypedIdentity(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to TypedIdentity")
	}
	d, ok := v.deserializers[si.Type]
	if !ok {
		return nil, errors.Errorf("no deserializer found for [%s]", si.Type)
	}
	return d.DeserializeVerifier(si.Type, si.Identity)
}

func (v *TypedVerifierDeserializerMultiplex) Recipients(id view.Identity) ([]view.Identity, error) {
	si, err := identity.UnmarshalTypedIdentity(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to TypedIdentity")
	}
	d, ok := v.deserializers[si.Type]
	if !ok {
		return nil, errors.Errorf("no deserializer found for [%s]", si.Type)
	}
	return d.Recipients(id, si.Type, si.Identity)
}

func (v *TypedVerifierDeserializerMultiplex) GetOwnerMatcher(raw []byte) (driver.Matcher, error) {
	return v.auditMatcherDeserializer.GetOwnerMatcher(raw)
}

func (v *TypedVerifierDeserializerMultiplex) Match(id view.Identity, ai []byte) error {
	// match identity and audit info
	recipient, err := identity.UnmarshalTypedIdentity(id)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal identity [%s]", id)
	}
	//if recipient.Type != v.identityType {
	//	return errors.Errorf("expected serialized identity type, got [%s]", recipient.Type)
	//}

	matcher, err := v.GetOwnerMatcher(ai)
	if err != nil {
		return errors.Wrapf(err, "failed getting audit info matcher for [%s]", id)
	}
	err = matcher.Match(recipient.Identity)
	if err != nil {
		return errors.Wrapf(err, "failed to match identity to audit infor for [%s:%s]", id, hash.Hashable(ai))
	}
	return nil
}

type TypedIdentityVerifierDeserializer struct {
	VerifierDeserializer
}

func NewTypedIdentityVerifierDeserializer(verifierDeserializer VerifierDeserializer) *TypedIdentityVerifierDeserializer {
	return &TypedIdentityVerifierDeserializer{VerifierDeserializer: verifierDeserializer}
}

func (t *TypedIdentityVerifierDeserializer) DeserializeVerifier(typ string, raw []byte) (driver.Verifier, error) {
	return t.VerifierDeserializer.DeserializeVerifier(raw)
}

func (t *TypedIdentityVerifierDeserializer) Recipients(id view.Identity, typ string, raw []byte) ([]view.Identity, error) {
	return []view.Identity{id}, nil
}
