/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	"encoding/asn1"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/pkg/errors"
)

const Escrow = "ms"

type MultiIdentity struct {
	Identities []token.Identity
}

func (m *MultiIdentity) Serialize() ([]byte, error) {
	return asn1.Marshal(*m)
}

func (m *MultiIdentity) Deserialize(raw []byte) error {
	_, err := asn1.Unmarshal(raw, m)
	return err
}

func (m *MultiIdentity) Bytes() ([]byte, error) {
	return asn1.Marshal(*m)
}

func WrapIdentities(ids ...token.Identity) (token.Identity, error) {
	mi := &MultiIdentity{Identities: ids}
	raw, err := mi.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "failed marshalling multi identity")
	}
	typedIdentity, err := (&identity.TypedIdentity{Type: Escrow, Identity: raw}).Bytes()
	if err != nil {
		return nil, err
	}
	return typedIdentity, nil
}

func Unwrap(raw []byte) (bool, []token.Identity, error) {
	ti, err := identity.UnmarshalTypedIdentity(raw)
	if err != nil {
		return false, nil, errors.Wrap(err, "failed unmarshalling typed identity")
	}
	if ti.Type != Escrow {
		return false, nil, nil
	}
	mi := &MultiIdentity{}
	err = mi.Deserialize(ti.Identity)
	if err != nil {
		return false, nil, errors.Wrap(err, "failed unmarshalling multi identity")
	}
	return true, mi.Identities, nil
}

type InfoMatcher struct {
	AuditInfoMatcher []driver.Matcher
}

func (e *InfoMatcher) Match(raw []byte) error {
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

type AuditInfoIdentity struct {
	AuditInfo []byte
}

type MultiIdentityAuditInfo struct {
	AuditInfoIdentities []AuditInfoIdentity
}

func WrapAuditInfo(recipients [][]byte) ([]byte, error) {
	mi := &MultiIdentityAuditInfo{
		AuditInfoIdentities: make([]AuditInfoIdentity, len(recipients)),
	}
	for k, recipient := range recipients {
		mi.AuditInfoIdentities[k] = AuditInfoIdentity{
			AuditInfo: recipient,
		}
	}
	return mi.Bytes()
}

func UnwrapAuditInfo(info []byte) (bool, [][]byte, error) {
	mi := &MultiIdentityAuditInfo{}
	err := json.Unmarshal(info, mi)
	if err != nil {
		return false, nil, err
	}
	recipients := make([][]byte, len(mi.AuditInfoIdentities))
	for k, identity := range mi.AuditInfoIdentities {
		recipients[k] = identity.AuditInfo
	}
	return true, recipients, nil
}

func (ei *MultiIdentityAuditInfo) EnrollmentID() string {
	return ""
}

func (ei *MultiIdentityAuditInfo) RevocationHandle() string {
	return ""
}

func (ei *MultiIdentityAuditInfo) Bytes() ([]byte, error) {
	return json.Marshal(ei)
}
