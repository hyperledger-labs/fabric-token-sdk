/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"path/filepath"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger/fabric-protos-go/msp"
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
)

func LoadConfig(dir string, ID string) (*msp.MSPConfig, error) {
	signcertDir := filepath.Join(dir, SignCerts)
	signcert, err := getPemMaterialFromDir(signcertDir)
	if err != nil || len(signcert) == 0 {
		return nil, errors.Wrapf(err, "could not load a valid signer certificate from directory %s", signcertDir)
	}
	return LoadConfigWithIdentityInfo(dir, ID, &msp.SigningIdentityInfo{PublicSigner: signcert[0], PrivateSigner: nil})
}

func LoadConfigWithIdentityInfo(dir string, ID string, signingIdentityInfo *msp.SigningIdentityInfo) (*msp.MSPConfig, error) {
	cacertDir := filepath.Join(dir, CACerts)
	cacerts, err := getPemMaterialFromDir(cacertDir)
	if err != nil || len(cacerts) == 0 {
		return nil, errors.WithMessagef(err, "could not load a valid ca certificate from directory %s", cacertDir)
	}

	// Set FabricCryptoConfig
	cryptoConfig := &msp.FabricCryptoConfig{
		SignatureHashFamily:            bccsp.SHA2,
		IdentityIdentifierHashFunction: bccsp.SHA256,
	}

	// Compose FabricMSPConfig
	fmspconf := &msp.FabricMSPConfig{
		RootCerts:       cacerts,
		SigningIdentity: signingIdentityInfo,
		Name:            ID,
		CryptoConfig:    cryptoConfig,
	}

	fmpsjs, err := proto.Marshal(fmspconf)
	if err != nil {
		return nil, err
	}

	return &msp.MSPConfig{Config: fmpsjs, Type: int32(FABRIC)}, nil
}

func RemoveSigningIdentityInfo(c *msp.MSPConfig) (*msp.MSPConfig, error) {
	fabricMSPConfig := &msp.FabricMSPConfig{}
	if err := proto.Unmarshal(c.Config, fabricMSPConfig); err != nil {
		return nil, err
	}
	fabricMSPConfig.SigningIdentity = nil

	raw, err := proto.Marshal(fabricMSPConfig)
	if err != nil {
		return nil, err
	}
	return &msp.MSPConfig{Config: raw, Type: int32(FABRIC)}, nil
}
