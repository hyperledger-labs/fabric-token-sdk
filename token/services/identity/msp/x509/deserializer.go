/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	ecdsa2 "crypto/ecdsa"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"
)

// MSPIdentityDeserializer takes as MSP identity and returns an ECDSA verifier
type MSPIdentityDeserializer struct{}

func (deserializer *MSPIdentityDeserializer) DeserializeVerifier(id view.Identity) (driver.Verifier, error) {
	si := &msp.SerializedIdentity{}
	err := proto.Unmarshal(id, si)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to msp.SerializedIdentity{}")
	}
	genericPublicKey, err := PemDecodeKey(si.IdBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed parsing received public key")
	}
	publicKey, ok := genericPublicKey.(*ecdsa2.PublicKey)
	if !ok {
		return nil, errors.New("expected *ecdsa.PublicKey")
	}
	return NewECDSAVerifier(publicKey), nil
}

type AuditInfoDeserializer struct {
	CommonName string
}

func (a *AuditInfoDeserializer) Match(id []byte) error {
	si := &msp.SerializedIdentity{}
	err := proto.Unmarshal(id, si)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal to msp.SerializedIdentity{}")
	}

	cert, err := PemDecodeCert(si.IdBytes)
	if err != nil {
		return errors.Wrap(err, "failed to decode certificate")
	}

	if cert.Subject.CommonName != a.CommonName {
		return errors.Errorf("expected [%s], got [%s]", a.CommonName, cert.Subject.CommonName)
	}

	return nil
}
