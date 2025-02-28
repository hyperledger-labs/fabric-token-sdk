/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"path/filepath"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/membership"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509/crypto"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	KeystoreFullFolder = "keystoreFull"
	PrivateKeyFileName = "priv_sk"
	KeystoreFolder     = "keystore"
)

type KeyManagerProvider struct {
	config        idriver.Config
	mspID         string
	signerService idriver.SigService
	keyStore      identity.Keystore
	// ignoreVerifyOnlyWallet when set to true, for each wallet the service will force the load of the secrets
	ignoreVerifyOnlyWallet bool
}

func NewKeyManagerProvider(
	config idriver.Config,
	mspID string,
	signerService idriver.SigService,
	keyStore identity.Keystore,
	ignoreVerifyOnlyWallet bool,
) *KeyManagerProvider {
	return &KeyManagerProvider{
		config:                 config,
		mspID:                  mspID,
		signerService:          signerService,
		keyStore:               keyStore,
		ignoreVerifyOnlyWallet: ignoreVerifyOnlyWallet,
	}
}

func (k *KeyManagerProvider) Get(idConfig *driver.IdentityConfiguration) (membership.KeyManager, error) {
	identityConfig := &idriver.ConfiguredIdentity{
		ID:   idConfig.ID,
		Path: idConfig.URL,
	}
	if len(idConfig.Config) != 0 {
		// load opts as yaml
		if err := yaml.Unmarshal(idConfig.Config, &identityConfig.Opts); err != nil {
			return nil, errors.Wrapf(err, "failed to load options for [%s]", idConfig.ID)
		}
	}
	var mspConfig *crypto.Config
	if len(idConfig.Raw) != 0 {
		// load raw as mspConfig
		var err error
		mspConfig, err = crypto.UnmarshalConfig(idConfig.Raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load msp config [%s]", idConfig.ID)
		}
	}
	return k.registerIdentity(mspConfig, identityConfig, idConfig)
}

func (k *KeyManagerProvider) registerIdentity(conf *crypto.Config, identityConfig *idriver.ConfiguredIdentity, idConfig *driver.IdentityConfiguration) (membership.KeyManager, error) {
	p, err := k.registerProvider(conf, identityConfig, idConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to register MSP provider")
	}
	return p, nil
}

func (k *KeyManagerProvider) registerProvider(conf *crypto.Config, identityConfig *idriver.ConfiguredIdentity, idConfig *driver.IdentityConfiguration) (membership.KeyManager, error) {
	opts, err := crypto.ToBCCSPOpts(identityConfig.Opts)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to extract BCCSP options")
	}
	if opts == nil {
		logger.Debugf("no BCCSP options set for [%s], opts [%v]", identityConfig.ID, identityConfig.Opts)
	} else {
		logger.Debugf("BCCSP options set for [%s] to [%v:%v:%v]", identityConfig.ID, opts, opts.PKCS11, opts.SW)
	}

	translatedPath := k.config.TranslatePath(identityConfig.Path)
	keyStorePath := k.keyStorePath()
	logger.Debugf("load provider at [%s][%s]", translatedPath, keyStorePath)
	// Try without "msp"
	provider, conf, err := NewKeyManagerFromConf(conf, translatedPath, keyStorePath, k.mspID, k.signerService, opts, nil)
	if err != nil {
		logger.Debugf("failed loading provider at [%s]: [%s]", translatedPath, err)
		// Try with "msp"
		provider, conf, err = NewKeyManagerFromConf(conf, filepath.Join(translatedPath, "msp"), keyStorePath, k.mspID, k.signerService, opts, nil)
		if err != nil {
			logger.Debugf("failed loading provider at [%s]: [%s]", filepath.Join(translatedPath, "msp"), err)
			return nil, err
		}
	}

	optsRaw, err := yaml.Marshal(identityConfig.Opts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal config [%v]", identityConfig)
	}
	confRaw, err := crypto.MarshalConfig(conf)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal msp config [%v]", identityConfig)
	}
	idConfig.Config = optsRaw
	idConfig.Raw = confRaw

	return provider, nil
}

func (k *KeyManagerProvider) keyStorePath() string {
	if !k.ignoreVerifyOnlyWallet {
		return ""
	}
	return KeystoreFullFolder
}
