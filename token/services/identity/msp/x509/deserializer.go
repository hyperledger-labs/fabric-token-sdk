/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	msp2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509/crypto"
	"github.com/pkg/errors"
)

// IdentityDeserializer takes as MSP identity and returns an ECDSA verifier
type IdentityDeserializer struct{}

func (d *IdentityDeserializer) DeserializeVerifier(id driver.Identity) (driver.Verifier, error) {
	return msp2.DeserializeVerifier(id)
}

type AuditMatcherDeserializer struct{}

func (a *AuditMatcherDeserializer) GetOwnerMatcher(owner driver.Identity, auditInfo []byte) (driver.Matcher, error) {
	ai := &AuditInfo{}
	err := ai.FromBytes(auditInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal")
	}
	return &AuditInfoMatcher{EnrollmentID: ai.EID}, nil
}

type AuditInfoMatcher struct {
	EnrollmentID string
}

func (a *AuditInfoMatcher) Match(id []byte) error {
	eid, err := msp2.GetEnrollmentID(id)
	if err != nil {
		return errors.Wrap(err, "failed to get enrollment ID")
	}
	if eid != a.EnrollmentID {
		return errors.Errorf("expected [%s], got [%s]", a.EnrollmentID, eid)
	}

	return nil
}

type AuditInfoDeserializer struct{}

func (a *AuditInfoDeserializer) DeserializeAuditInfo(raw []byte) (driver2.AuditInfo, error) {
	ai := &AuditInfo{}
	err := ai.FromBytes(raw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal")
	}
	logger.Debugf("audit info [%s]", string(raw))
	return ai, nil
}
