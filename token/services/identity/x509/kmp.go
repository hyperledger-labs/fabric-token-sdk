/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"path/filepath"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/membership"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509/crypto/csp"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	KeystoreFullFolder = "keystoreFull"
	PrivateKeyFileName = "priv_sk"
	KeystoreFolder     = "keystore"
	ExtraPathElement   = "msp"
)

func NewKeyStore(kvs idriver.Keystore) crypto.KeyStore {
	return csp.NewKVSStore(kvs)
}

type KeyManagerProvider struct {
	config        idriver.Config
	signerService idriver.SigService
	keyStore      crypto.KeyStore
	// ignoreVerifyOnlyWallet when set to true, for each wallet the service will force the load of the secrets
	ignoreVerifyOnlyWallet bool
}

func NewKeyManagerProvider(config idriver.Config, signerService idriver.SigService, keyStore crypto.KeyStore, ignoreVerifyOnlyWallet bool) *KeyManagerProvider {
	return &KeyManagerProvider{
		config:                 config,
		signerService:          signerService,
		ignoreVerifyOnlyWallet: ignoreVerifyOnlyWallet,
		keyStore:               keyStore,
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
	var config *crypto.Config
	if len(idConfig.Raw) != 0 {
		// load raw as config
		var err error
		config, err = crypto.UnmarshalConfig(idConfig.Raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load config [%s]", idConfig.ID)
		}
	}
	return k.registerIdentity(config, identityConfig, idConfig)
}

func (k *KeyManagerProvider) registerIdentity(conf *crypto.Config, identityConfig *idriver.ConfiguredIdentity, idConfig *driver.IdentityConfiguration) (membership.KeyManager, error) {
	p, err := k.registerProvider(conf, identityConfig, idConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to register provider")
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
	// Try without ExtraPathElement
	provider, conf, err := NewKeyManagerFromConf(
		conf,
		translatedPath,
		keyStorePath,
		k.signerService,
		opts,
		k.keyStore,
	)
	if err != nil {
		logger.Debugf("failed loading provider at [%s]: [%s]", translatedPath, err)
		// Try with ExtraPathElement
		provider, conf, err = NewKeyManagerFromConf(
			conf,
			filepath.Join(translatedPath, ExtraPathElement),
			keyStorePath,
			k.signerService,
			opts,
			k.keyStore,
		)
		if err != nil {
			logger.Debugf("failed loading provider at [%s]: [%s]", filepath.Join(translatedPath, ExtraPathElement), err)
			return nil, err
		}
	}

	optsRaw, err := yaml.Marshal(identityConfig.Opts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal config [%v]", identityConfig)
	}
	confRaw, err := crypto.MarshalConfig(conf)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal config [%v]", identityConfig)
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
