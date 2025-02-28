/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func readFile(file string) ([]byte, error) {
	fileCont, err := os.ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read file %s", file)
	}

	return fileCont, nil
}

func readPemFile(file string) ([]byte, error) {
	bytes, err := readFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "reading from file %s failed", file)
	}

	b, _ := pem.Decode(bytes)
	if b == nil { // TODO: also check that the type is what we expect (cert vs key..)
		return nil, errors.Errorf("no pem content for file %s", file)
	}

	return bytes, nil
}

func getCertFromPem(idBytes []byte) (*x509.Certificate, error) {
	if idBytes == nil {
		return nil, errors.New("getCertFromPem error: nil idBytes")
	}

	// Decode the pem bytes
	pemCert, _ := pem.Decode(idBytes)
	if pemCert == nil {
		return nil, errors.Errorf("getCertFromPem error: could not decode pem bytes [%v]", idBytes)
	}

	// get a cert
	var cert *x509.Certificate
	cert, err := x509.ParseCertificate(pemCert.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "getCertFromPem error: failed to parse x509 cert")
	}

	return cert, nil
}

func getPemMaterialFromDir(dir string) ([][]byte, error) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil, err
	}

	content := make([][]byte, 0)
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read directory %s", dir)
	}

	for _, f := range files {
		fullName := filepath.Join(dir, f.Name())
		f, err := os.Stat(fullName)
		if err != nil {
			continue
		}
		if f.IsDir() {
			continue
		}
		item, err := readPemFile(fullName)
		if err != nil {
			continue
		}
		content = append(content, item)
	}

	return content, nil
}

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
