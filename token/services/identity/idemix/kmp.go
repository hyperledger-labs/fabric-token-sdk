/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	bccsp "github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/cache"
	crypto2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/membership"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

var logger = logging.MustGetLogger()

type KeyManagerProvider struct {
	issuerPublicKey []byte
	curveID         math.CurveID
	keyStore        bccsp.KeyStore
	signerService   SignerService
	config          driver2.Config
	cacheSize       int

	// ignoreVerifyOnlyWallet when set to true, for each wallet the service will force the load of the secrets
	ignoreVerifyOnlyWallet bool
}

func NewKeyManagerProvider(issuerPublicKey []byte, curveID math.CurveID, keyStore bccsp.KeyStore, signerService SignerService, config driver2.Config, cacheSize int, ignoreVerifyOnlyWallet bool) *KeyManagerProvider {
	return &KeyManagerProvider{issuerPublicKey: issuerPublicKey, curveID: curveID, keyStore: keyStore, signerService: signerService, config: config, cacheSize: cacheSize, ignoreVerifyOnlyWallet: ignoreVerifyOnlyWallet}
}

func (l *KeyManagerProvider) Get(identityConfig *driver.IdentityConfiguration) (membership.KeyManager, error) {
	var conf *crypto2.Config
	var err error
	if len(identityConfig.Raw) != 0 {
		// load the config directly from identityConfig.Raw
		logger.Infof("load the config directly from identityConfig.Raw [%s][%s]", identityConfig.ID, hash.Hashable(identityConfig.Raw))
		conf, err = crypto2.NewConfigFromRaw(l.issuerPublicKey, identityConfig.Raw)
	} else {
		// load from URL
		logger.Infof("load the config form identityConfig.URL [%s][%s]", identityConfig.ID, identityConfig.URL)
		conf, err = crypto2.NewConfigWithIPK(l.issuerPublicKey, identityConfig.URL, l.ignoreVerifyOnlyWallet)
	}
	if err != nil {
		return nil, err
	}

	// instantiate provider from configuration
	cryptoProvider, err := crypto2.NewBCCSP(l.keyStore, l.curveID, l.curveID == math.BLS12_381_BBS)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to instantiate crypto provider")
	}
	keyManager, err := NewKeyManager(conf, l.signerService, bccsp.EidNymRhNym, cryptoProvider)
	if err != nil {
		return nil, errors.Wrapf(err, "failed instantiating idemix key manager provider from [%s]", identityConfig.URL)
	}

	cacheSize, err := l.cacheSizeForID(identityConfig.ID)
	if err != nil {
		return nil, err
	}

	var getIdentityFunc func([]byte) (driver.Identity, []byte, error)
	if keyManager.IsRemote() {
		getIdentityFunc = func([]byte) (driver.Identity, []byte, error) {
			return nil, nil, errors.Errorf("cannot invoke this function, remote must register pseudonyms")
		}
	} else {
		getIdentityFunc = cache.NewIdentityCache(
			keyManager.Identity,
			cacheSize,
			nil,
		).Identity
	}

	// finalize identity configuration
	// remove SK and keep only SKI
	conf.Signer.Sk = nil
	identityConfigurationRawField, err := proto.Marshal(conf)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to marshal identity configuration")
	}
	identityConfig.Raw = identityConfigurationRawField

	return &WrappedKeyManager{
		KeyManager:      keyManager,
		getIdentityFunc: getIdentityFunc,
	}, nil
}

func (l *KeyManagerProvider) cacheSizeForID(id string) (int, error) {
	cacheSize := l.config.CacheSizeForOwnerID(id)
	if cacheSize <= 0 {
		logger.Debugf("cache size for %s not configured, using default (%d)", id, l.cacheSize)
		cacheSize = l.cacheSize
	}
	return cacheSize, nil
}

type WrappedKeyManager struct {
	membership.KeyManager
	getIdentityFunc func([]byte) (driver.Identity, []byte, error)
}

func (k *WrappedKeyManager) Identity(auditInfo []byte) (driver.Identity, []byte, error) {
	return k.getIdentityFunc(auditInfo)
}
