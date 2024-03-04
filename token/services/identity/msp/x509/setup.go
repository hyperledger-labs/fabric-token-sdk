/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	x5092 "crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"os"
	"path/filepath"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/config"
	pkcs112 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509/pkcs11"
	msp2 "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/pkcs11"
	"github.com/hyperledger/fabric/bccsp/sw"
	"github.com/hyperledger/fabric/msp"
	"github.com/pkg/errors"
)

const (
	BCCSPType = "bccsp"
	SignCerts = "signcerts"
)

// GetSigningIdentity retrieves a signing identity from the passed arguments.
// If keyStorePath is empty, then it is assumed that the key is at mspConfigPath/keystore
func GetSigningIdentity(mspConfigPath, keyStorePath, mspID string, bccspConfig *config.BCCSP) (driver.FullIdentity, error) {
	mspInstance, err := LoadLocalMSPAt(mspConfigPath, keyStorePath, mspID, BCCSPType, bccspConfig)
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
func LoadLocalMSPAt(dir, keyStorePath, id, mspType string, bccspConfig *config.BCCSP) (msp.MSP, error) {
	if mspType != BCCSPType {
		return nil, errors.Errorf("invalid msp type, expected 'bccsp', got %s", mspType)
	}
	conf, err := msp.GetLocalMspConfig(dir, nil, id)
	if err != nil {
		return nil, errors.WithMessagef(err, "could not get msp config from dir [%s]", dir)
	}

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

// LoadVerifyingMSPAt loads a verifying MSP whose configuration is stored at 'dir', and whose
// id and type are the passed as arguments.
func LoadVerifyingMSPAt(dir, id, mspType string) (msp.MSP, error) {
	if mspType != BCCSPType {
		return nil, errors.Errorf("invalid msp type, expected 'bccsp', got %s", mspType)
	}
	conf, err := msp.GetVerifyingMspConfig(dir, id, mspType)
	if err != nil {
		return nil, errors.WithMessagef(err, "could not get msp config from dir [%s]", dir)
	}

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

// GetBCCSPFromConf returns a BCCSP instance and its relative key store from the passed configuration.
// If no configuration is passed, the default one is used, namely the `SW` provider.
func GetBCCSPFromConf(dir string, keyStorePath string, conf *config.BCCSP) (bccsp.BCCSP, bccsp.KeyStore, error) {
	if len(keyStorePath) == 0 {
		keyStorePath = filepath.Join(dir, "keystore")
	}
	if conf == nil {
		return GetSWBCCSP(keyStorePath)
	}

	switch conf.Default {
	case "SW":
		return GetSWBCCSP(keyStorePath)
	case "PKCS11":
		return GetPKCS11BCCSP(conf)
	default:
		return nil, nil, errors.Errorf("invalid config.BCCSP.Default.%s", conf.Default)
	}
}

// GetPKCS11BCCSP returns a new instance of the HSM-based BCCSP
func GetPKCS11BCCSP(conf *config.BCCSP) (bccsp.BCCSP, bccsp.KeyStore, error) {
	if conf.PKCS11 == nil {
		return nil, nil, errors.New("invalid config.BCCSP.PKCS11. missing configuration")
	}

	p11Opts := *conf.PKCS11
	ks := sw.NewDummyKeyStore()
	mapper := skiMapper(p11Opts)
	csp, err := pkcs11.New(*pkcs112.ToPKCS11Opts(&p11Opts), ks, pkcs11.WithKeyMapper(mapper))
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "Failed initializing PKCS11 library with config [%+v]", p11Opts)
	}
	return csp, ks, nil
}

func skiMapper(p11Opts config.PKCS11) func([]byte) []byte {
	keyMap := map[string]string{}
	for _, k := range p11Opts.KeyIDs {
		keyMap[k.SKI] = k.ID
	}

	return func(ski []byte) []byte {
		keyID := hex.EncodeToString(ski)
		if id, ok := keyMap[keyID]; ok {
			return []byte(id)
		}
		if p11Opts.AltID != "" {
			return []byte(p11Opts.AltID)
		}
		return ski
	}
}

// GetSWBCCSP returns a new instance of the software-based BCCSP
func GetSWBCCSP(dir string) (bccsp.BCCSP, bccsp.KeyStore, error) {
	ks, err := sw.NewFileBasedKeyStore(nil, dir, true)
	if err != nil {
		return nil, nil, err
	}
	cryptoProvider, err := sw.NewDefaultSecurityLevelWithKeystore(ks)
	if err != nil {
		return nil, nil, err
	}
	return cryptoProvider, ks, nil
}

func GetEnrollmentID(id []byte) (string, error) {
	si := &msp2.SerializedIdentity{}
	err := proto.Unmarshal(id, si)
	if err != nil {
		return "", errors.Wrap(err, "failed to unmarshal to msp.SerializedIdentity{}")
	}
	block, _ := pem.Decode(si.IdBytes)
	if block == nil {
		return "", errors.New("bytes are not PEM encoded")
	}
	switch block.Type {
	case "CERTIFICATE":
		cert, err := x5092.ParseCertificate(block.Bytes)
		if err != nil {
			return "", errors.WithMessage(err, "pem bytes are not cert encoded ")
		}
		return cert.Subject.CommonName, nil
	default:
		return "", errors.Errorf("bad block type %s, expected CERTIFICATE", block.Type)
	}
}

func GetRevocationHandle(id []byte) ([]byte, error) {
	si := &msp2.SerializedIdentity{}
	err := proto.Unmarshal(id, si)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to msp.SerializedIdentity{}")
	}
	block, _ := pem.Decode(si.IdBytes)
	if block == nil {
		return nil, errors.New("bytes are not PEM encoded")
	}
	switch block.Type {
	case "CERTIFICATE":
		cert, err := x5092.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, errors.WithMessage(err, "pem bytes are not cert encoded ")
		}
		encoded, err := x5092.MarshalPKIXPublicKey(cert.PublicKey)
		if err != nil {
			return nil, errors.WithMessage(err, "Failed to marshal PKI public key")
		}
		return []byte(hash.Hashable(encoded).String()), nil
	default:
		return nil, errors.Errorf("bad block type %s, expected CERTIFICATE", block.Type)
	}
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
