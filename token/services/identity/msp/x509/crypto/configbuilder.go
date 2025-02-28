/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"os"
	"path/filepath"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/pkg/errors"
)

// ProviderType indicates the type of identity provider
type ProviderType int

const (
	// FABRIC The ProviderType of the default MSP provider
	FABRIC ProviderType = iota // MSP is of FABRIC type
)

const (
	CACerts   = "cacerts"
	SignCerts = "signcerts"
	KeyStore  = "keystore"
	PrivSK    = "priv_sk"
)

func LoadConfig(dir string, keyStoreDirName string, ID string) (*Config, error) {
	signcertDir := filepath.Join(dir, SignCerts)
	signcert, err := getPemMaterialFromDir(signcertDir)
	if err != nil || len(signcert) == 0 {
		return nil, errors.Wrapf(err, "could not load a valid signer certificate from directory %s", signcertDir)
	}
	// load secret key, if available. If not available, the public's key SKI will be used to load the secret key from the key store
	if len(keyStoreDirName) == 0 {
		keyStoreDirName = KeyStore
	}
	secretKeyFile := filepath.Join(dir, keyStoreDirName, PrivSK)
	var skRaw []byte
	if _, err := os.Stat(secretKeyFile); err == nil {
		skRaw, err = readPemFile(secretKeyFile)
		if err != nil {
			return nil, errors.Wrapf(err, "could not load private key from file %s", secretKeyFile)
		}
	}
	return LoadConfigWithIdentityInfo(
		dir,
		ID,
		&SigningIdentityInfo{
			PublicSigner: signcert[0],
			PrivateSigner: &KeyInfo{
				KeyIdentifier: "",
				KeyMaterial:   skRaw,
			},
		},
	)
}

func LoadConfigWithIdentityInfo(dir string, ID string, signingIdentityInfo *SigningIdentityInfo) (*Config, error) {
	cacertDir := filepath.Join(dir, CACerts)
	cacerts, err := getPemMaterialFromDir(cacertDir)
	if err != nil || len(cacerts) == 0 {
		return nil, errors.WithMessagef(err, "could not load a valid ca certificate from directory %s", cacertDir)
	}

	// Set FabricCryptoConfig
	cryptoConfig := &FabricCryptoConfig{
		SignatureHashFamily:            bccsp.SHA2,
		IdentityIdentifierHashFunction: bccsp.SHA256,
	}

	// Compose FabricMSPConfig
	fmspconf := &FabricMSPConfig{
		RootCerts:       cacerts,
		SigningIdentity: signingIdentityInfo,
		Name:            ID,
		CryptoConfig:    cryptoConfig,
	}

	fmpsjs, err := proto.Marshal(fmspconf)
	if err != nil {
		return nil, err
	}

	return &Config{Config: fmpsjs, Type: int32(FABRIC)}, nil
}

func RemoveSigningIdentityInfo(c *Config) (*Config, error) {
	fabricMSPConfig := &FabricMSPConfig{}
	if err := proto.Unmarshal(c.Config, fabricMSPConfig); err != nil {
		return nil, err
	}
	fabricMSPConfig.SigningIdentity = nil

	raw, err := proto.Marshal(fabricMSPConfig)
	if err != nil {
		return nil, err
	}
	return &Config{Config: raw, Type: int32(FABRIC)}, nil
}
