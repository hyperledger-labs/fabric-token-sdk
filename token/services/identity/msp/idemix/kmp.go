/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	bccsp "github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/membership"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix/cache"
	msp2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger/fabric-protos-go-apiv2/msp"
	"github.com/pkg/errors"
)

var logger = logging.MustGetLogger("token-sdk.services.identity.msp.idemix")

type KeyManagerProvider struct {
	issuerPublicKey []byte
	curveID         math.CurveID
	mspID           string
	keyStore        bccsp.KeyStore
	signerService   SignerService
	config          driver2.Config
	cacheSize       int

	// ignoreVerifyOnlyWallet when set to true, for each wallet the service will force the load of the secrets
	ignoreVerifyOnlyWallet bool
}

func NewKeyManagerProvider(issuerPublicKey []byte, curveID math.CurveID, mspID string, keyStore bccsp.KeyStore, signerService SignerService, config driver2.Config, cacheSize int, ignoreVerifyOnlyWallet bool) *KeyManagerProvider {
	return &KeyManagerProvider{issuerPublicKey: issuerPublicKey, curveID: curveID, mspID: mspID, keyStore: keyStore, signerService: signerService, config: config, cacheSize: cacheSize, ignoreVerifyOnlyWallet: ignoreVerifyOnlyWallet}
}

func (l *KeyManagerProvider) Get(identityConfig *driver.IdentityConfiguration) (membership.KeyManager, error) {
	var conf *msp.MSPConfig
	var err error
	if len(identityConfig.Raw) != 0 {
		// load the msp config directly from identityConfig.Raw
		logger.Infof("load the msp config directly from identityConfig.Raw [%s][%s]", identityConfig.ID, hash.Hashable(identityConfig.Raw))
		conf, err = msp2.NewMSPConfigFromRawSigner(l.issuerPublicKey, identityConfig.Raw, l.mspID)
	} else {
		// load from URL
		logger.Infof("load the msp config form identityConfig.URL [%s][%s]", identityConfig.ID, identityConfig.URL)
		conf, err = msp2.NewMSPConfigFromURL(l.issuerPublicKey, identityConfig.URL, l.mspID, l.ignoreVerifyOnlyWallet)
	}
	if err != nil {
		return nil, err
	}

	// instantiate provider from configuration
	cryptoProvider, err := msp2.NewBCCSP(l.keyStore, l.curveID, l.curveID == math.BLS12_381_BBS)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to instantiate crypto provider")
	}
	provider, err := NewKeyManager(conf, l.signerService, bccsp.EidNymRhNym, cryptoProvider)
	if err != nil {
		return nil, errors.Wrapf(err, "failed instantiating idemix msp provider from [%s]", identityConfig.URL)
	}

	cacheSize, err := l.cacheSizeForID(identityConfig.ID)
	if err != nil {
		return nil, err
	}

	var getIdentityFunc func([]byte) (driver.Identity, []byte, error)
	if provider.IsRemote() {
		getIdentityFunc = func([]byte) (driver.Identity, []byte, error) {
			return nil, nil, errors.Errorf("cannot invoke this function, remote must register pseudonyms")
		}
	} else {
		getIdentityFunc = cache.NewIdentityCache(
			provider.Identity,
			cacheSize,
			nil,
		).Identity
	}

	return &WrappedKeyManager{
		KeyManager:      provider,
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
