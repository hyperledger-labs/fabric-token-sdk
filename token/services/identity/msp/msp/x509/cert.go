/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"crypto/x509"
	"encoding/pem"

	"github.com/pkg/errors"
)

func PemDecodeCert(pemBytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("bytes are not PEM encoded")
	}

	switch block.Type {
	case "CERTIFICATE":
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, errors.WithMessage(err, "pem bytes are not cert encoded ")
		}
		return cert, nil
	default:
		return nil, errors.Errorf("bad type %s, expected 'CERTIFICATE", block.Type)
	}
}
