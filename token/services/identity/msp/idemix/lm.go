/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"os"
	"path/filepath"
	"sync"

	bccsp "github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/config"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/common"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix/cache"
	msp2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var logger = logging.MustGetLogger("token-sdk.services.identity.msp.idemix")

type LocalMembership struct {
	issuerPublicKey []byte
	curveID         math.CurveID

	config                 config2.Config
	defaultNetworkIdentity driver.Identity
	signerService          common.SigService
	deserializerManager    driver2.DeserializerManager
	identityDB             driver3.IdentityDB
	keyStore               bccsp.KeyStore
	mspID                  string
	cacheSize              int

	resolversMutex          sync.RWMutex
	resolvers               []*common.Resolver
	resolversByName         map[string]*common.Resolver
	resolversByEnrollmentID map[string]*common.Resolver
	identities              []*config.Identity
	// ignoreVerifyOnlyWallet when set to true, for each wallet the service will force the load of the secrets
	ignoreVerifyOnlyWallet bool
}

func NewLocalMembership(
	issuerPublicKey []byte,
	idemixCurveID math.CurveID,
	config config2.Config,
	defaultNetworkIdentity driver.Identity,
	signerService common.SigService,
	deserializerManager driver2.DeserializerManager,
	identityDB driver3.IdentityDB,
	keyStore bccsp.KeyStore,
	mspID string,
	cacheSize int,
	identities []*config.Identity,
	ignoreVerifyOnlyWallet bool,
) *LocalMembership {
	return &LocalMembership{
		issuerPublicKey:         issuerPublicKey,
		curveID:                 idemixCurveID,
		config:                  config,
		defaultNetworkIdentity:  defaultNetworkIdentity,
		signerService:           signerService,
		deserializerManager:     deserializerManager,
		identityDB:              identityDB,
		keyStore:                keyStore,
		mspID:                   mspID,
		cacheSize:               cacheSize,
		resolversByEnrollmentID: map[string]*common.Resolver{},
		resolversByName:         map[string]*common.Resolver{},
		identities:              identities,
		ignoreVerifyOnlyWallet:  ignoreVerifyOnlyWallet,
	}
}

func (l *LocalMembership) DefaultNetworkIdentity() driver.Identity {
	return l.defaultNetworkIdentity
}

func (l *LocalMembership) IsMe(id driver.Identity) bool {
	return l.signerService.IsMe(id)
}

func (l *LocalMembership) GetIdentifier(id driver.Identity) (string, error) {
	l.resolversMutex.RLock()
	defer l.resolversMutex.RUnlock()

	label := string(id)
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get anonymous identity info by label [%s]", hash.Hashable(label))
	}
	r := l.getResolver(label)
	if r == nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("anonymous identity info not found for label [%s][%v]", hash.Hashable(label), l.resolversByName)
		}
		return "", errors.New("not found")
	}
	return r.Name, nil
}

func (l *LocalMembership) GetDefaultIdentifier() string {
	for _, resolver := range l.resolvers {
		if resolver.Default {
			return resolver.Name
		}
	}
	return ""
}

func (l *LocalMembership) GetIdentityInfo(label string, auditInfo []byte) (driver.IdentityInfo, error) {
	l.resolversMutex.RLock()
	defer l.resolversMutex.RUnlock()

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get anonymous identity info by label [%s]", hash.Hashable(label))
	}
	r := l.getResolver(label)
	if r == nil {
		return nil, errors.Errorf("anonymous identity info not found for label [%s][%v]", hash.Hashable(label), l.resolversByName)
	}

	return common.NewIdentityInfo(
		r.Name,
		r.EnrollmentID,
		r.Remote,
		func() (driver.Identity, []byte, error) {
			return r.GetIdentity(&common.IdentityOptions{
				EIDExtension: true,
				AuditInfo:    auditInfo,
			})
		},
	), nil
}

func (l *LocalMembership) RegisterIdentity(idConfig driver.IdentityConfiguration) error {
	l.resolversMutex.Lock()
	defer l.resolversMutex.Unlock()

	return l.registerIdentityConfiguration(idConfig, l.GetDefaultIdentifier() == "")
}

func (l *LocalMembership) IDs() ([]string, error) {
	var ids []string
	for _, resolver := range l.resolvers {
		ids = append(ids, resolver.Name)
	}
	return ids, nil
}

func (l *LocalMembership) Load() error {
	logger.Debugf("Load Idemix Wallets with the respect to curve [%d], [%+q]", l.curveID, l.identities)

	l.resolversMutex.Lock()
	defer l.resolversMutex.Unlock()

	// cleanup all resolvers
	l.resolvers = make([]*common.Resolver, 0)
	l.resolversByName = make(map[string]*common.Resolver)
	l.resolversByEnrollmentID = make(map[string]*common.Resolver)

	// load identities from configuration
	for _, identityConfig := range l.identities {
		logger.Debugf("load wallet for identity [%+v]", identityConfig)
		if err := l.registerIdentity(*identityConfig); err != nil {
			return errors.WithMessage(err, "failed to load identity")
		}
		logger.Debugf("load wallet for identity [%+v] done.", identityConfig)
	}

	// load identities from storage
	logger.Debugf("load identities from storage...")
	if err := l.loadFromStorage(); err != nil {
		return errors.Wrapf(err, "failed to load identities from identityDB")
	}
	logger.Debugf("load identities from storage...done")

	// if no default identity, use the first one
	defaultIdentifier := l.GetDefaultIdentifier()
	if len(defaultIdentifier) == 0 {
		logger.Warnf("no default identity, use the first one available")
		if len(l.resolvers) > 0 {
			logger.Warnf("set default identity to %s", l.resolvers[0].Name)
			l.resolvers[0].Default = true
		} else {
			logger.Warnf("cannot set default identity, no identity available")
		}
	} else {
		logger.Debugf("default identifier is [%s]", defaultIdentifier)
	}

	return nil
}

func (l *LocalMembership) registerIdentity(identity config.Identity) error {
	return l.registerIdentityConfiguration(driver.IdentityConfiguration{
		ID:     identity.ID,
		URL:    identity.Path,
		Config: nil,
		Raw:    nil,
	}, identity.Default)
}

func (l *LocalMembership) registerIdentityConfiguration(identity driver.IdentityConfiguration, defaultIdentity bool) error {
	// Try to register the MSP provider
	identity.URL = l.config.TranslatePath(identity.URL)
	if err := l.registerProvider(identity, defaultIdentity); err != nil {
		logger.Warnf("failed to load idemix msp provider at [%s]:[%s]", identity.URL, err)
		// Does path correspond to a holder containing multiple MSP identities?
		if err := l.registerProviders(identity); err != nil {
			return errors.WithMessage(err, "failed to register MSP provider")
		}
	}
	return nil
}

func (l *LocalMembership) registerProvider(identityConfig driver.IdentityConfiguration, defaultIdentity bool) error {
	var conf *msp.MSPConfig
	var err error
	if len(identityConfig.Raw) != 0 {
		// load the msp config directly from identityConfig.Raw
		logger.Infof("load the msp config directly from identityConfig.Raw [%s]", hash.Hashable(identityConfig.Raw))
		conf, err = msp2.NewMSPConfigFromRawSigner(l.issuerPublicKey, identityConfig.Raw, l.mspID)
	} else {
		// load from URL
		logger.Infof("load the msp config form identityConfig.URL [%s]", identityConfig.URL)
		conf, err = msp2.NewMSPConfigFromURL(l.issuerPublicKey, identityConfig.URL, l.mspID, l.ignoreVerifyOnlyWallet)
	}
	if err != nil {
		return err
	}

	// instantiate provider from configuration
	cryptoProvider, err := msp2.NewBCCSP(l.keyStore, l.curveID, l.curveID == math.BLS12_381_BBS)
	if err != nil {
		return errors.WithMessage(err, "failed to instantiate crypto provider")
	}
	provider, err := NewProvider(conf, l.signerService, bccsp.EidNymRhNym, cryptoProvider)
	if err != nil {
		return errors.Wrapf(err, "failed instantiating idemix msp provider from [%s]", identityConfig.URL)
	}

	cacheSize, err := l.cacheSizeForID(identityConfig.ID)
	if err != nil {
		return err
	}

	var getIdentityFunc func(opts *common.IdentityOptions) (driver.Identity, []byte, error)
	l.deserializerManager.AddDeserializer(provider)
	if provider.IsRemote() {
		getIdentityFunc = func(opts *common.IdentityOptions) (driver.Identity, []byte, error) {
			return nil, nil, errors.Errorf("cannot invoke this function, remote must register pseudonyms")
		}
	} else {
		getIdentityFunc = cache.NewIdentityCache(
			provider.Identity,
			cacheSize,
			&common.IdentityOptions{
				EIDExtension: true,
			},
		).Identity
	}
	l.addResolver(identityConfig.ID, provider.EnrollmentID(), provider.IsRemote(), defaultIdentity, getIdentityFunc)

	if exists, _ := l.identityDB.ConfigurationExists(identityConfig.ID, msp2.IdentityConfigurationType); !exists {
		if err := l.identityDB.AddConfiguration(driver3.IdentityConfiguration{
			ID:     identityConfig.ID,
			Type:   msp2.IdentityConfigurationType,
			URL:    identityConfig.URL,
			Config: identityConfig.Config,
			Raw:    identityConfig.Raw,
		}); err != nil {
			return err
		}
	}

	logger.Debugf("added idemix resolver for id [%s] with cache of size [%d], remote [%v]", identityConfig.ID+"@"+provider.EnrollmentID(), cacheSize, provider.IsRemote())
	return nil
}

func (l *LocalMembership) registerProviders(configuration driver.IdentityConfiguration) error {
	entries, err := os.ReadDir(configuration.URL)
	if err != nil {
		logger.Warnf("failed reading from [%s]: [%s]", configuration.URL, err)
		return nil
	}
	found := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		if err := l.registerProvider(driver.IdentityConfiguration{
			ID:     id,
			URL:    filepath.Join(configuration.URL, id),
			Config: configuration.Config,
		}, false); err != nil {
			logger.Errorf("failed registering msp provider [%s]: [%s]", id, err)
			continue
		}
		found++
	}
	if found == 0 {
		return errors.Errorf("no valid identities found in [%s]", configuration.URL)
	}
	return nil
}

func (l *LocalMembership) addResolver(Name string, EnrollmentID string, remote bool, defaultID bool, IdentityGetter common.GetIdentityFunc) {
	resolver := &common.Resolver{
		Name:         Name,
		Default:      defaultID,
		EnrollmentID: EnrollmentID,
		GetIdentity:  IdentityGetter,
		Remote:       remote,
	}
	l.resolversByName[Name] = resolver
	if len(EnrollmentID) != 0 {
		l.resolversByEnrollmentID[EnrollmentID] = resolver
	}
	l.resolvers = append(l.resolvers, resolver)
}

func (l *LocalMembership) getResolver(label string) *common.Resolver {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get anonymous identity info by label [%s]", hash.Hashable(label))
	}
	r, ok := l.resolversByName[label]
	if ok {
		return r
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("anonymous identity info not found for label [%s][%v]", hash.Hashable(label), l.resolversByName)
	}
	return nil
}

func (l *LocalMembership) cacheSizeForID(id string) (int, error) {
	cacheSize := l.config.CacheSizeForOwnerID(id)
	if cacheSize <= 0 {
		logger.Debugf("cache size for %s not configured, using default (%d)", id, l.cacheSize)
		cacheSize = l.cacheSize
	}
	return cacheSize, nil
}

func (l *LocalMembership) loadFromStorage() error {
	it, err := l.identityDB.IteratorConfigurations(msp2.IdentityConfigurationType)
	if err != nil {
		return errors.WithMessage(err, "failed to get registered identities from kvs")
	}
	defer it.Close()
	for it.HasNext() {
		entry, err := it.Next()
		if err != nil {
			return errors.WithMessagef(err, "failed to get next registered identities from kvs")
		}
		id := entry.ID
		if l.getResolver(id) != nil {
			continue
		}
		if err := l.registerIdentityConfiguration(driver.IdentityConfiguration{
			ID:     entry.ID,
			URL:    entry.URL,
			Config: entry.Config,
			Raw:    entry.Raw,
		}, l.GetDefaultIdentifier() == ""); err != nil {
			return err
		}
	}
	return nil
}
