/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	ecdsa2 "crypto/ecdsa"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	msp2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509/msp"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"
)

// MSPIdentityDeserializer takes as MSP identity and returns an ECDSA verifier
type MSPIdentityDeserializer struct{}

func (deserializer *MSPIdentityDeserializer) DeserializeVerifier(id driver.Identity) (driver.Verifier, error) {
	si := &msp.SerializedIdentity{}
	err := proto.Unmarshal(id, si)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to msp.SerializedIdentity{}")
	}
	genericPublicKey, err := msp2.PemDecodeKey(si.IdBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed parsing received public key")
	}
	publicKey, ok := genericPublicKey.(*ecdsa2.PublicKey)
	if !ok {
		return nil, errors.New("expected *ecdsa.PublicKey")
	}
	return msp2.NewECDSAVerifier(publicKey), nil
}

type AuditMatcherDeserializer struct{}

func (a *AuditMatcherDeserializer) GetOwnerMatcher(raw []byte) (driver.Matcher, error) {
	ai := &AuditInfo{}
	err := ai.FromBytes(raw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal")
	}
	return &AuditInfoMatcher{CommonName: ai.EID}, nil
}

type AuditInfoMatcher struct {
	CommonName string
}

func (a *AuditInfoMatcher) Match(id []byte) error {
	si := &msp.SerializedIdentity{}
	err := proto.Unmarshal(id, si)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal to msp.SerializedIdentity{}")
	}

	cert, err := msp2.PemDecodeCert(si.IdBytes)
	if err != nil {
		return errors.Wrap(err, "failed to decode certificate")
	}

	if cert.Subject.CommonName != a.CommonName {
		return errors.Errorf("expected [%s], got [%s]", a.CommonName, cert.Subject.CommonName)
	}

	return nil
}

type AuditInfoDeserializer struct{}

func (a *AuditInfoDeserializer) DeserializeAuditInfo(raw []byte) (*AuditInfo, error) {
	ai := &AuditInfo{}
	err := ai.FromBytes(raw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal")
	}
	return ai, nil
}
