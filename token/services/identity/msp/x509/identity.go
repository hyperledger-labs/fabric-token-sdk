/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"encoding/pem"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"
)

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
