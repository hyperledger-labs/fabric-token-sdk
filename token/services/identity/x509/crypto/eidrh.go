/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
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
	h := sha256.Sum256(encoded)
	return []byte(base64.StdEncoding.EncodeToString(h[:])), nil
}
