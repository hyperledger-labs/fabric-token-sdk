/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"crypto/x509"
	"encoding/pem"
	"os"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"
)

func Serialize(mspID string, certPath string) ([]byte, error) {
	raw, err := readPemFile(certPath)
	if err != nil {
		return nil, err
	}
	return SerializeRaw(mspID, raw)
}

func SerializeRaw(mspID string, raw []byte) ([]byte, error) {
	cert, err := getCertFromPem(raw)
	if err != nil {
		return nil, err
	}

	pb := &pem.Block{Bytes: cert.Raw, Type: "CERTIFICATE"}
	pemBytes := pem.EncodeToMemory(pb)
	if pemBytes == nil {
		return nil, errors.New("encoding of identity failed")
	}

	// We serialize identities by prepending the MSPID and appending the ASN.1 DER content of the cert
	sID := &msp.SerializedIdentity{Mspid: mspID, IdBytes: pemBytes}
	idBytes, err := proto.Marshal(sID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal a SerializedIdentity structure for identity %s", mspID)
	}

	return idBytes, nil
}

func SerializeFromMSP(mspID string, path string) ([]byte, error) {
	msp, err := LoadVerifyingMSPAt(path, mspID, BCCSPType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load msp at [%s:%s]", mspID, path)
	}
	certRaw, err := LoadLocalMSPSignerCert(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load certificate at [%s:%s]", mspID, path)
	}
	serRaw, err := SerializeRaw(mspID, certRaw)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to generate msp serailization at [%s:%s]", mspID, path)
	}
	id, err := msp.DeserializeIdentity(serRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to deserialize certificate at [%s:%s]", mspID, path)
	}
	return id.Serialize()
}

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
