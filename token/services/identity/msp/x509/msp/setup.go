/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	msp2 "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/pkg/errors"
)

func SerializeFromMSP(conf *msp2.MSPConfig, mspID string) ([]byte, error) {
	factory, err := getIdentityFactory(nil, nil)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get identity factory")
	}
	identityInfo, err := getSigningIdentityInfo(conf)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get signing identity info")
	}
	signingIdentity, err := factory.GetIdentity(identityInfo)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get signing identity")
	}
	return signingIdentity.Serialize()
}

// GetSigningIdentity retrieves a signing identity from the passed arguments.
// If keyStorePath is empty, then it is assumed that the key is at mspConfigPath/keystore
func GetSigningIdentity(conf *msp2.MSPConfig, bccspConfig *BCCSP, keyStore bccsp.KeyStore) (driver.FullIdentity, error) {
	factory, err := getIdentityFactory(bccspConfig, keyStore)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get identity factory")
	}
	identityInfo, err := getSigningIdentityInfo(conf)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get signing identity info")
	}
	signingIdentity, err := factory.GetFullIdentity(identityInfo)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get signing identity")
	}
	return signingIdentity, nil
}

// getIdentityFactory loads an MSP whose configuration is stored at 'dir', and whose
// id and type are the passed as arguments.
func getIdentityFactory(bccspConfig *BCCSP, keyStore bccsp.KeyStore) (*IdentityFactory, error) {
	csp, err := GetBCCSPFromConf(bccspConfig, keyStore)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get bccsp from config [%v]", bccspConfig)
	}
	return NewIdentityFactory(csp, bccsp.SHA2), nil
}

func getSigningIdentityInfo(conf *msp2.MSPConfig) (*msp2.SigningIdentityInfo, error) {
	c := &msp2.FabricMSPConfig{}
	if err := proto.Unmarshal(conf.Config, c); err != nil {
		return nil, errors.WithMessagef(err, "failed to load provider config [%v]", conf.Config)
	}
	if c.SigningIdentity == nil {
		return nil, errors.Errorf("signing identity is missing")
	}
	return c.SigningIdentity, nil
}
