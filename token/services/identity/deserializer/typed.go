/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package deserializer

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var logger = logging.MustGetLogger("token-sdk.services.identity.deserializer")

type TypedVerifierDeserializer interface {
	DeserializeVerifier(typ string, raw []byte) (driver.Verifier, error)
	Recipients(id driver.Identity, typ string, raw []byte) ([]driver.Identity, error)
	GetOwnerAuditInfo(id driver.Identity, typ string, raw []byte, p driver.AuditInfoProvider) ([][]byte, error)
	GetOwnerMatcher(owner driver.Identity, auditInfo []byte) (driver.Matcher, error)
}

// AuditMatcherDeserializer deserializes raw bytes into a matcher, which allows an auditor to match an identity to an enrollment ID
type AuditMatcherDeserializer interface {
	GetOwnerMatcher(owner driver.Identity, auditInfo []byte) (driver.Matcher, error)
}

type TypedVerifierDeserializerMultiplex struct {
	deserializers map[string]TypedVerifierDeserializer
}

func NewTypedVerifierDeserializerMultiplex() *TypedVerifierDeserializerMultiplex {
	return &TypedVerifierDeserializerMultiplex{deserializers: map[string]TypedVerifierDeserializer{}}
}

func (v *TypedVerifierDeserializerMultiplex) AddTypedVerifierDeserializer(typ string, d TypedVerifierDeserializer) {
	v.deserializers[typ] = d
}

func (v *TypedVerifierDeserializerMultiplex) DeserializeVerifier(id driver.Identity) (driver.Verifier, error) {
	si, err := identity.UnmarshalTypedIdentity(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to TypedIdentity")
	}
	d, ok := v.deserializers[si.Type]
	if !ok {
		return nil, errors.Errorf("no deserializer found for [%s]", si.Type)
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("Deserializing [%s] with type [%s]", id, si.Type)
	}
	verifier, err := d.DeserializeVerifier(si.Type, si.Identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed deserializing [%s]", si.Type)
	}
	return verifier, nil
}

func (v *TypedVerifierDeserializerMultiplex) Recipients(id driver.Identity) ([]driver.Identity, error) {
	if id.IsNone() {
		return nil, nil
	}
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

func (v *TypedVerifierDeserializerMultiplex) GetOwnerMatcher(id driver.Identity, auditInfo []byte) (driver.Matcher, error) {
	if id.IsNone() {
		return nil, nil
	}
	si, err := identity.UnmarshalTypedIdentity(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to TypedIdentity")
	}
	return v.getOwnerMatcher(si.Type, id, auditInfo)
}

func (v *TypedVerifierDeserializerMultiplex) getOwnerMatcher(idType string, id driver.Identity, auditInfo []byte) (driver.Matcher, error) {
	d, ok := v.deserializers[idType]
	if !ok {
		return nil, errors.Errorf("no deserializer found for [%s]", idType)
	}
	return d.GetOwnerMatcher(id, auditInfo)
}

func (v *TypedVerifierDeserializerMultiplex) MatchOwnerIdentity(id driver.Identity, ai []byte) error {
	// match identity and audit info
	recipient, err := identity.UnmarshalTypedIdentity(id)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal identity [%s]", id)
	}
	// if recipient.Type != v.identityType {
	//	return errors.Errorf("expected serialized identity type, got [%s]", recipient.Type)
	// }

	matcher, err := v.getOwnerMatcher(recipient.Type, id, ai)
	if err != nil {
		return errors.Wrapf(err, "failed getting audit info matcher for [%s]", id)
	}
	err = matcher.Match(recipient.Identity)
	if err != nil {
		return errors.Wrapf(err, "failed to match identity to audit infor for [%s:%s]", id, hash.Hashable(ai))
	}
	return nil
}

func (v *TypedVerifierDeserializerMultiplex) GetOwnerAuditInfo(id driver.Identity, p driver.AuditInfoProvider) ([][]byte, error) {
	si, err := identity.UnmarshalTypedIdentity(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to TypedIdentity")
	}
	d, ok := v.deserializers[si.Type]
	if !ok {
		return nil, errors.Errorf("no deserializer found for [%s]", si.Type)
	}
	res, err := d.GetOwnerAuditInfo(id, si.Type, si.Identity, p)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get owner audit info, type [%s]", si.Type)
	}
	return res, nil
}

type TypedIdentityVerifierDeserializer struct {
	common.VerifierDeserializer
	common.MatcherDeserializer
}

func NewTypedIdentityVerifierDeserializer(verifierDeserializer common.VerifierDeserializer, matcherDeserializer common.MatcherDeserializer) *TypedIdentityVerifierDeserializer {
	return &TypedIdentityVerifierDeserializer{VerifierDeserializer: verifierDeserializer, MatcherDeserializer: matcherDeserializer}
}

func (t *TypedIdentityVerifierDeserializer) DeserializeVerifier(typ string, raw []byte) (driver.Verifier, error) {
	return t.VerifierDeserializer.DeserializeVerifier(raw)
}

func (t *TypedIdentityVerifierDeserializer) Recipients(id driver.Identity, typ string, raw []byte) ([]driver.Identity, error) {
	return []driver.Identity{id}, nil
}

func (t *TypedIdentityVerifierDeserializer) GetOwnerAuditInfo(id driver.Identity, typ string, raw []byte, p driver.AuditInfoProvider) ([][]byte, error) {
	auditInfo, err := p.GetAuditInfo(id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for recipient identity [%s]", id)
	}
	return [][]byte{auditInfo}, nil
}

func (t *TypedIdentityVerifierDeserializer) GetOwnerMatcher(owner driver.Identity, auditInfo []byte) (driver.Matcher, error) {
	return t.MatcherDeserializer.GetOwnerMatcher(owner, auditInfo)
}
