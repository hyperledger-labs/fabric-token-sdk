/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"crypto/x509"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"
)

func GetEnrollmentID(id []byte) (string, error) {
	si := &msp.SerializedIdentity{}
	err := proto.Unmarshal(id, si)
	if err != nil {
		return "", errors.Wrap(err, "failed to unmarshal to msp.SerializedIdentity{}")
	}
	cert, err := PemDecodeCert(si.IdBytes)
	if err != nil {
		return "", err
	}
	return cert.Subject.CommonName, nil
}

func GetRevocationHandle(id []byte) ([]byte, error) {
	si := &msp.SerializedIdentity{}
	err := proto.Unmarshal(id, si)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to msp.SerializedIdentity{}")
	}
	cert, err := PemDecodeCert(si.IdBytes)
	if err != nil {
		return nil, err
	}
	encoded, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to marshal PKI public key")
	}
	return []byte(hash.Hashable(encoded).String()), nil
}
