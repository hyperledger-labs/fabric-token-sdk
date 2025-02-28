/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/pkg/errors"
)

func SerializeIdentity(conf *Config) ([]byte, error) {
	factory, err := getIdentityFactory(conf, nil, nil)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get identity factory")
	}
	signingIdentity, err := factory.GetIdentity(conf.SigningIdentity)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get signing identity")
	}
	return signingIdentity.Serialize()
}

// GetSigningIdentity retrieves a signing identity from the passed arguments.
// If keyStorePath is empty, then it is assumed that the key is at mspConfigPath/keystore
func GetSigningIdentity(conf *Config, bccspConfig *BCCSP, keyStore bccsp.KeyStore) (driver.FullIdentity, error) {
	factory, err := getIdentityFactory(conf, bccspConfig, keyStore)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get identity factory")
	}
	signingIdentity, err := factory.GetFullIdentity(conf.SigningIdentity)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get signing identity")
	}
	return signingIdentity, nil
}

// getIdentityFactory loads an MSP whose configuration is stored at 'dir', and whose
// id and type are the passed as arguments.
func getIdentityFactory(conf *Config, bccspConfig *BCCSP, keyStore bccsp.KeyStore) (*IdentityFactory, error) {
	csp, err := GetBCCSPFromConf(bccspConfig, keyStore)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get bccsp from config [%v]", bccspConfig)
	}
	return NewIdentityFactory(csp, conf.CryptoConfig.SignatureHashFamily), nil
}
