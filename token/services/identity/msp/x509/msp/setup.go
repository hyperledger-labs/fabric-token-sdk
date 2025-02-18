/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"encoding/pem"
	"path/filepath"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	msp2 "github.com/hyperledger/fabric-protos-go-apiv2/msp"
	"github.com/hyperledger/fabric/msp"
	"github.com/pkg/errors"
)

const (
	SignCerts = "signcerts"
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
	sID := &msp2.SerializedIdentity{Mspid: mspID, IdBytes: pemBytes}
	idBytes, err := proto.Marshal(sID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal a SerializedIdentity structure for identity %s", mspID)
	}

	return idBytes, nil
}

// LoadVerifyingMSPAt loads a verifying MSP whose configuration is stored at 'dir', and whose
// id and type are the passed as arguments.
func LoadVerifyingMSPAt(conf *msp2.MSPConfig, dir string) (msp.MSP, error) {
	cp, _, err := GetBCCSPFromConf(dir, "", nil)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get bccsp")
	}

	mspOpts := &msp.BCCSPNewOpts{
		NewBaseOpts: msp.NewBaseOpts{
			Version: msp.MSPv1_0,
		},
	}
	thisMSP, err := msp.New(mspOpts, cp)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create new BCCSPMSP instance at [%s]", dir)
	}
	err = thisMSP.Setup(conf)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to setup the new BCCSPMSP instance at [%s]", dir)
	}
	return thisMSP, nil
}

func LoadLocalMSPSignerCert(dir string) ([]byte, error) {
	signCertsPath := filepath.Join(dir, SignCerts)
	signCerts, err := getPemMaterialFromDir(signCertsPath)
	if err != nil || len(signCerts) == 0 {
		return nil, errors.Wrapf(err, "could not load a valid signer certificate from directory %s", signCertsPath)
	}
	return signCerts[0], nil
}

func SerializeFromMSP(conf *msp2.MSPConfig, mspID string, path string) ([]byte, error) {
	msp, err := LoadVerifyingMSPAt(conf, path)
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

// GetSigningIdentity retrieves a signing identity from the passed arguments.
// If keyStorePath is empty, then it is assumed that the key is at mspConfigPath/keystore
func GetSigningIdentity(conf *msp2.MSPConfig, mspConfigPath, keyStorePath string, bccspConfig *BCCSP) (driver.FullIdentity, error) {
	mspInstance, err := LoadLocalMSPAt(conf, mspConfigPath, keyStorePath, bccspConfig)
	if err != nil {
		return nil, err
	}

	signingIdentity, err := mspInstance.GetDefaultSigningIdentity()
	if err != nil {
		return nil, err
	}

	return signingIdentity, nil
}

// LoadLocalMSPAt loads an MSP whose configuration is stored at 'dir', and whose
// id and type are the passed as arguments.
func LoadLocalMSPAt(conf *msp2.MSPConfig, dir, keyStorePath string, bccspConfig *BCCSP) (msp.MSP, error) {
	cp, _, err := GetBCCSPFromConf(dir, keyStorePath, bccspConfig)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get bccsp from config [%v]", bccspConfig)
	}

	mspOpts := &msp.BCCSPNewOpts{
		NewBaseOpts: msp.NewBaseOpts{
			Version: msp.MSPv1_0,
		},
	}
	thisMSP, err := msp.New(mspOpts, cp)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create new BCCSPMSP instance at [%s]", dir)
	}
	err = thisMSP.Setup(conf)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to setup the new BCCSPMSP instance at [%s]", dir)
	}
	return thisMSP, nil
}
