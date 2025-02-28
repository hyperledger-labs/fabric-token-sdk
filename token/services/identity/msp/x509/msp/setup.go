/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"encoding/pem"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	msp2 "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric/msp"
	"github.com/pkg/errors"
)

func SerializeFromMSP(conf *msp2.MSPConfig, mspID string) ([]byte, error) {
	msp, err := loadVerifyingMSPAt(conf)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load msp at [%s]", mspID)
	}
	certRaw, err := loadLocalMSPSignerCert(conf)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load certificate at [%s]", mspID)
	}
	serRaw, err := serializeRaw(mspID, certRaw)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to generate msp serailization at [%s]", mspID)
	}
	id, err := msp.DeserializeIdentity(serRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to deserialize certificate at [%s]", mspID)
	}
	return id.Serialize()
}

// GetSigningIdentity retrieves a signing identity from the passed arguments.
// If keyStorePath is empty, then it is assumed that the key is at mspConfigPath/keystore
func GetSigningIdentity(conf *msp2.MSPConfig, bccspConfig *BCCSP) (driver.FullIdentity, error) {
	mspInstance, err := loadLocalMSPAt(conf, bccspConfig)
	if err != nil {
		return nil, err
	}

	signingIdentity, err := mspInstance.GetDefaultSigningIdentity()
	if err != nil {
		return nil, err
	}

	return signingIdentity, nil
}

// loadVerifyingMSPAt loads a verifying MSP whose configuration is stored at 'dir', and whose
// id and type are the passed as arguments.
func loadVerifyingMSPAt(conf *msp2.MSPConfig) (msp.MSP, error) {
	cp, _, err := GetBCCSPFromConf(nil)
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
		return nil, errors.WithMessagef(err, "failed to create new BCCSPMSP instance")
	}
	err = thisMSP.Setup(conf)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to setup the new BCCSPMSP instance")
	}
	return thisMSP, nil
}

func serializeRaw(mspID string, raw []byte) ([]byte, error) {
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

// loadLocalMSPAt loads an MSP whose configuration is stored at 'dir', and whose
// id and type are the passed as arguments.
func loadLocalMSPAt(conf *msp2.MSPConfig, bccspConfig *BCCSP) (msp.MSP, error) {
	cp, _, err := GetBCCSPFromConf(bccspConfig)
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
		return nil, errors.WithMessagef(err, "failed to create new BCCSPMSP instance")
	}
	err = thisMSP.Setup(conf)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to setup the new BCCSPMSP instance")
	}
	return thisMSP, nil
}

func loadLocalMSPSignerCert(conf *msp2.MSPConfig) ([]byte, error) {
	c := &msp2.FabricMSPConfig{}
	if err := proto.Unmarshal(conf.Config, c); err != nil {
		return nil, errors.WithMessagef(err, "failed to load provider config [%v]", conf.Config)
	}
	if c.SigningIdentity == nil {
		return nil, errors.Errorf("signing identity is missing")
	}
	return c.SigningIdentity.PublicSigner, nil
}
