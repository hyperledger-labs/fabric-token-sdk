/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/pkg/errors"
)

type VerifierDES interface {
	DeserializeVerifier(id driver.Identity) (driver.Verifier, error)
}

type AuditInfoMatcher interface {
	GetOwnerMatcher(owner driver.Identity, auditInfo []byte) (driver.Matcher, error)
}

type TypedIdentityDeserializer struct {
	VerifierDeserializer VerifierDES
	AuditInfoMatcher     AuditInfoMatcher
}

func NewTypedIdentityDeserializer(verifierDeserializer VerifierDES, auditInfoDeserializer AuditInfoMatcher) *TypedIdentityDeserializer {
	return &TypedIdentityDeserializer{VerifierDeserializer: verifierDeserializer, AuditInfoMatcher: auditInfoDeserializer}
}

func (d *TypedIdentityDeserializer) GetOwnerAuditInfo(id driver.Identity, typ identity.Type, rawIdentity []byte, p driver.AuditInfoProvider) ([]byte, error) {
	if typ != Multisig {
		return nil, errors.Errorf("invalid type, got [%s], expected [%s]", typ, Multisig)
	}

	// if there is already some audit info for id, return it
	auditInfoRaw, err := p.GetAuditInfo(id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for id [%s]", id.String())
	}
	if len(auditInfoRaw) != 0 {
		return auditInfoRaw, nil
	}

	// otherwise, build it
	mid := MultiIdentity{}
	err = mid.Deserialize(rawIdentity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal mid")
	}
	auditInfo := &AuditInfo{}
	auditInfo.IdentityAuditInfos = make([]IdentityAuditInfo, len(mid.Identities))
	for k, identity := range mid.Identities {
		auditInfo.IdentityAuditInfos[k].AuditInfo, err = p.GetAuditInfo(identity)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting audit info for mid [%s]", id.String())
		}
	}
	return auditInfo.Bytes()
}

func (d *TypedIdentityDeserializer) GetOwnerMatcher(owner driver.Identity, auditInfo []byte) (driver.Matcher, error) {
	ei := &AuditInfo{}
	err := json.Unmarshal(auditInfo, ei)
	if err != nil {
		return nil, err
	}
	id, err := identity.UnmarshalTypedIdentity(owner)
	if err != nil {
		return nil, err
	}
	mid := MultiIdentity{}
	err = mid.Deserialize(id.Identity)
	if err != nil {
		return nil, err
	}
	if len(mid.Identities) != len(ei.IdentityAuditInfos) {
		return nil, errors.Errorf("expected %d audit info but received %d", len(mid.Identities), len(ei.IdentityAuditInfos))
	}
	matchers := make([]driver.Matcher, len(ei.IdentityAuditInfos))
	for k, info := range ei.IdentityAuditInfos {
		matchers[k], err = d.AuditInfoMatcher.GetOwnerMatcher(mid.Identities[k], info.AuditInfo)
		if err != nil {
			return nil, err
		}
	}
	return &InfoMatcher{AuditInfoMatcher: matchers}, nil
}

func (d *TypedIdentityDeserializer) DeserializeVerifier(typ identity.Type, id []byte) (driver.Verifier, error) {
	multisigIdentity := &MultiIdentity{}
	err := multisigIdentity.Deserialize(id)
	if err != nil {
		return nil, errors.New("failed to unmarshal multisig identity")
	}
	verifier := &Verifier{}
	verifier.Verifiers = make([]driver.Verifier, len(multisigIdentity.Identities))
	for k, i := range multisigIdentity.Identities {
		verifier.Verifiers[k], err = d.VerifierDeserializer.DeserializeVerifier(i)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal multisig identity")
		}
	}
	return verifier, nil
}

func (d *TypedIdentityDeserializer) Recipients(id driver.Identity, typ identity.Type, raw []byte) ([]driver.Identity, error) {
	mid := &MultiIdentity{}
	err := mid.Deserialize(raw)
	if err != nil {
		return nil, err
	}
	return mid.Identities, nil
}

type AuditInfoDeserializer struct {
}

func (a *AuditInfoDeserializer) DeserializeAuditInfo(raw []byte) (driver2.AuditInfo, error) {
	ei := &AuditInfo{}
	err := json.Unmarshal(raw, ei)
	if err != nil {
		return nil, err
	}
	return ei, nil
}
