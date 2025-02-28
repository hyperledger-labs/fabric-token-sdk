/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"crypto/x509"
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/pkg/errors"
)

func GetEnrollmentID(id []byte) (string, error) {
	si := &SerializedIdentity{}
	err := proto.Unmarshal(id, si)
	if err != nil {
		return "", errors.Wrap(err, "failed to unmarshal to SerializedIdentity{}")
	}
	cert, err := PemDecodeCert(si.IdBytes)
	if err != nil {
		return "", err
	}
	cn := cert.Subject.CommonName
	// if cn contains a @, then return only the left part of the string
	index := strings.Index(cn, "@")
	if index != -1 {
		cn = cn[:index]
	}
	return cn, nil
}

func GetRevocationHandle(id []byte) ([]byte, error) {
	si := &SerializedIdentity{}
	err := proto.Unmarshal(id, si)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to SerializedIdentity{}")
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
