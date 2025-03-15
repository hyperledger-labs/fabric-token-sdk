/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package deserializer

import (
	errors2 "errors"

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
	DeserializeVerifier(typ identity.Type, raw []byte) (driver.Verifier, error)
	Recipients(id driver.Identity, typ identity.Type, raw []byte) ([]driver.Identity, error)
	GetAuditInfo(id driver.Identity, typ identity.Type, raw []byte, p driver.AuditInfoProvider) ([]byte, error)
	GetAuditInfoMatcher(owner driver.Identity, auditInfo []byte) (driver.Matcher, error)
}

// AuditMatcherDeserializer deserializes raw bytes into a matcher, which allows an auditor to match an identity to an enrollment ID
type AuditMatcherDeserializer interface {
	GetAuditInfoMatcher(owner driver.Identity, auditInfo []byte) (driver.Matcher, error)
}

type TypedVerifierDeserializerMultiplex struct {
	deserializers map[string][]TypedVerifierDeserializer
}

func NewTypedVerifierDeserializerMultiplex() *TypedVerifierDeserializerMultiplex {
	return &TypedVerifierDeserializerMultiplex{deserializers: map[string][]TypedVerifierDeserializer{}}
}

func (v *TypedVerifierDeserializerMultiplex) AddTypedVerifierDeserializer(typ string, d TypedVerifierDeserializer) {
	_, ok := v.deserializers[typ]
	if !ok {
		v.deserializers[typ] = []TypedVerifierDeserializer{d}
		return
	}
	v.deserializers[typ] = append(v.deserializers[typ], d)
}

func (v *TypedVerifierDeserializerMultiplex) DeserializeVerifier(id driver.Identity) (driver.Verifier, error) {
	si, err := identity.UnmarshalTypedIdentity(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to TypedIdentity")
	}
	dess, ok := v.deserializers[si.Type]
	if !ok {
		return nil, errors.Errorf("no deserializer found for [%s]", si.Type)
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("Deserializing [%s] with type [%s]", id, si.Type)
	}
	var errs []error
	for _, deserializer := range dess {
		verifier, err := deserializer.DeserializeVerifier(si.Type, si.Identity)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		return verifier, nil
	}
	return nil, errors.Wrapf(errors2.Join(errs...), "failed to deserialize verifier for [%s]", si.Type)
}

func (v *TypedVerifierDeserializerMultiplex) Recipients(id driver.Identity) ([]driver.Identity, error) {
	if id.IsNone() {
		return nil, nil
	}
	si, err := identity.UnmarshalTypedIdentity(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to TypedIdentity")
	}
	dess, ok := v.deserializers[si.Type]
	if !ok {
		return nil, errors.Errorf("no deserializer found for [%s]", si.Type)
	}

	var errs []error
	for _, deserializer := range dess {
		ids, err := deserializer.Recipients(id, si.Type, si.Identity)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		return ids, nil
	}
	return nil, errors.Wrapf(errors2.Join(errs...), "failed to deserializer recipients for [%s]", si.Type)
}

func (v *TypedVerifierDeserializerMultiplex) GetAuditInfoMatcher(id driver.Identity, auditInfo []byte) (driver.Matcher, error) {
	if id.IsNone() {
		return nil, nil
	}
	si, err := identity.UnmarshalTypedIdentity(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to TypedIdentity")
	}
	matcher, err := v.getMatcher(si.Type, id, auditInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting matcher for [%s]", si.Type)
	}
	return &TypedAuditInfoMatcher{matcher: matcher}, nil
}

func (v *TypedVerifierDeserializerMultiplex) getMatcher(idType string, id driver.Identity, auditInfo []byte) (driver.Matcher, error) {
	dess, ok := v.deserializers[idType]
	if !ok {
		return nil, errors.Errorf("no deserializer found for [%s]", idType)
	}

	var errs []error
	for _, deserializer := range dess {
		matcher, err := deserializer.GetAuditInfoMatcher(id, auditInfo)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		return matcher, nil
	}

	return nil, errors.Wrapf(errors2.Join(errs...), "failed to find a valid owner matcher for [%s]", idType)
}

func (v *TypedVerifierDeserializerMultiplex) MatchIdentity(id driver.Identity, ai []byte) error {
	// match identity and audit info
	recipient, err := identity.UnmarshalTypedIdentity(id)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal identity [%s]", id)
	}
	matcher, err := v.getMatcher(recipient.Type, id, ai)
	if err != nil {
		return errors.Wrapf(err, "failed getting audit info matcher for [%s]", id)
	}
	err = matcher.Match(recipient.Identity)
	if err != nil {
		return errors.Wrapf(err, "failed to match identity to audit infor for [%s]:[%s]", id, hash.Hashable(ai))
	}
	return nil
}

func (v *TypedVerifierDeserializerMultiplex) GetAuditInfo(id driver.Identity, p driver.AuditInfoProvider) ([]byte, error) {
	si, err := identity.UnmarshalTypedIdentity(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to TypedIdentity")
	}
	dess, ok := v.deserializers[si.Type]
	if !ok {
		return nil, errors.Errorf("no deserializer found for [%s]", si.Type)
	}
	var errs []error
	for _, deserializer := range dess {
		info, err := deserializer.GetAuditInfo(id, si.Type, si.Identity, p)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		return info, nil
	}
	return nil, errors.Wrapf(errors2.Join(errs...), "failed to find a valid deserializer for audit info for [%s]", si.Type)
}

type TypedIdentityVerifierDeserializer struct {
	common.VerifierDeserializer
	common.MatcherDeserializer
}

func NewTypedIdentityVerifierDeserializer(verifierDeserializer common.VerifierDeserializer, matcherDeserializer common.MatcherDeserializer) *TypedIdentityVerifierDeserializer {
	return &TypedIdentityVerifierDeserializer{VerifierDeserializer: verifierDeserializer, MatcherDeserializer: matcherDeserializer}
}

func (t *TypedIdentityVerifierDeserializer) DeserializeVerifier(typ identity.Type, raw []byte) (driver.Verifier, error) {
	return t.VerifierDeserializer.DeserializeVerifier(raw)
}

func (t *TypedIdentityVerifierDeserializer) Recipients(id driver.Identity, typ identity.Type, raw []byte) ([]driver.Identity, error) {
	return []driver.Identity{id}, nil
}

func (t *TypedIdentityVerifierDeserializer) GetAuditInfo(id driver.Identity, typ identity.Type, raw []byte, p driver.AuditInfoProvider) ([]byte, error) {
	auditInfo, err := p.GetAuditInfo(id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for recipient identity [%s]", id)
	}
	return auditInfo, nil
}

func (t *TypedIdentityVerifierDeserializer) GetAuditInfoMatcher(owner driver.Identity, auditInfo []byte) (driver.Matcher, error) {
	return t.MatcherDeserializer.GetAuditInfoMatcher(owner, auditInfo)
}

type TypedAuditInfoMatcher struct {
	matcher driver.Matcher
}

func (t *TypedAuditInfoMatcher) Match(id []byte) error {
	// match identity and audit info
	recipient, err := identity.UnmarshalTypedIdentity(id)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal identity [%s]", id)
	}
	err = t.matcher.Match(recipient.Identity)
	if err != nil {
		return errors.Wrapf(err, "failed to match identity [%s] to audit infor", id)
	}
	return nil
}
