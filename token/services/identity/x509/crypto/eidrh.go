/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"crypto/x509"
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/hash"
)

func GetEnrollmentID(id []byte) (string, error) {
	cert, err := PemDecodeCert(id)
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
	cert, err := PemDecodeCert(id)
	if err != nil {
		return nil, err
	}
	encoded, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to marshal PKI public key")
	}
	return []byte(hash.Hashable(encoded).String()), nil
}
