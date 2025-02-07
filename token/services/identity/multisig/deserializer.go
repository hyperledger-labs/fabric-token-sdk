/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	"encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/pkg/errors"
)

const Escrow = "Multisig"

type VerifierDES interface {
	DeserializeVerifier(id driver.Identity) (driver.Verifier, error)
}

type AuditInfoMatcher interface {
	GetOwnerMatcher(owner driver.Identity, auditInfo []byte) (driver.Matcher, error)
}

type EscrowInfoMatcher struct {
	AuditInfoMatcher []driver.Matcher
}

func (e *EscrowInfoMatcher) Match(raw []byte) error {
	mid := MultiIdentity{}
	err := mid.Deserialize(raw)
	if err != nil {
		return err
	}
	if len(e.AuditInfoMatcher) != len(mid.Identities) {
		return errors.Errorf("expected [%d] identities, received [%d]", len(e.AuditInfoMatcher), len(mid.Identities))
	}
	for k, id := range mid.Identities {
		tid, err := identity.UnmarshalTypedIdentity(id)
		if err != nil {
			return err
		}
		err = e.AuditInfoMatcher[k].Match(tid.Identity)
		if err != nil {
			return errors.Wrapf(err, "identity at index %d does not match the audit info", k)
		}
	}
	return nil
}

type EscrowInfo struct {
	AuditInfo [][]byte
	EID       string
	RH        string
}

type TypedIdentityDeserializer struct {
	VerifierDeserializer VerifierDES
	AuditInfoMatcher     AuditInfoMatcher
}

func NewTypedIdentityDeserializer(verifierDeserializer VerifierDES, auditInfoDeserializer AuditInfoMatcher) *TypedIdentityDeserializer {
	return &TypedIdentityDeserializer{VerifierDeserializer: verifierDeserializer, AuditInfoMatcher: auditInfoDeserializer}
}

func (d *TypedIdentityDeserializer) GetOwnerAuditInfo(id driver.Identity, typ string, raw []byte, p driver.AuditInfoProvider) ([][]byte, error) {
	if typ != Escrow {
		return nil, errors.Errorf("invalid type, got [%s], expected [%s]", typ, Escrow)
	}
	mid := MultiIdentity{}
	var err error
	err = mid.Deserialize(raw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal mid")
	}
	ei := &EscrowInfo{}
	ei.AuditInfo = make([][]byte, len(mid.Identities))

	for k, identity := range mid.Identities {
		ei.AuditInfo[k], err = p.GetAuditInfo(identity)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting audit info for mid [%s]", id.String())
		}
	}
	auditInfoRaw, err := json.Marshal(ei)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshaling audit info for mid")
	}
	return [][]byte{auditInfoRaw}, nil
}

func (d *TypedIdentityDeserializer) GetOwnerMatcher(owner driver.Identity, auditInfo []byte) (driver.Matcher, error) {
	ei := &EscrowInfo{}
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
	if len(mid.Identities) != len(ei.AuditInfo) {
		return nil, errors.Errorf("expected %d audit info but received %d", len(mid.Identities), len(ei.AuditInfo))
	}
	matchers := make([]driver.Matcher, len(ei.AuditInfo))
	for k, info := range ei.AuditInfo {
		matchers[k], err = d.AuditInfoMatcher.GetOwnerMatcher(mid.Identities[k], info)
		if err != nil {
			return nil, err
		}
	}
	return &EscrowInfoMatcher{AuditInfoMatcher: matchers}, nil
}

func (d *TypedIdentityDeserializer) DeserializeVerifier(typ string, id []byte) (driver.Verifier, error) {
	escrow := &MultiIdentity{}
	err := escrow.Deserialize(id)
	if err != nil {
		return nil, errors.New("failed to unmarshal multisig identity")
	}
	verifier := &MultiVerifier{}
	verifier.Verifiers = make([]driver.Verifier, len(escrow.Identities))
	for k, i := range escrow.Identities {
		verifier.Verifiers[k], err = d.VerifierDeserializer.DeserializeVerifier(i)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal "+
				"multisig identity")
		}
	}
	return verifier, nil
}

func (t *TypedIdentityDeserializer) Recipients(id driver.Identity, typ string, raw []byte) ([]driver.Identity, error) {
	mid := &MultiIdentity{}
	err := mid.Deserialize(raw)
	if err != nil {
		return nil, err
	}
	return mid.Identities, nil
}

type EscrowInfoDeserializer struct {
}

func (a *EscrowInfoDeserializer) DeserializeAuditInfo(raw []byte) (driver2.AuditInfo, error) {
	ei := &EscrowInfo{}
	err := json.Unmarshal(raw, ei)
	if err != nil {
		return nil, err
	}
	return ei, nil
}

func (ei *EscrowInfo) EnrollmentID() string {
	return ei.EID
}
func (ei *EscrowInfo) RevocationHandle() string {
	return ei.RH
}
