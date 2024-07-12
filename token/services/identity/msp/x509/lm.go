/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"os"
	"path/filepath"
	"sync"

	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/common"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509/msp"
	m "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
)

const (
	KeystoreFullFolder        = "keystoreFull"
	PrivateKeyFileName        = "priv_sk"
	KeystoreFolder            = "keystore"
	IdentityConfigurationType = "x509"
)

type LocalMembership struct {
	config                 config2.Config
	defaultNetworkIdentity driver.Identity
	signerService          common.SigService
	binderService          common.BinderService
	deserializerManager    driver3.DeserializerManager
	identityDB             driver2.IdentityDB
	mspID                  string

	resolversMutex           sync.RWMutex
	resolvers                []*common.Resolver
	resolversByName          map[string]*common.Resolver
	resolversByEnrollmentID  map[string]*common.Resolver
	bccspResolversByIdentity map[string]*common.Resolver
	// ignoreVerifyOnlyWallet when set to true, for each wallet the service will force the load of the secrets
	ignoreVerifyOnlyWallet bool
}

func NewLocalMembership(
	config config2.Config,
	defaultNetworkIdentity driver.Identity,
	signerService common.SigService,
	binderService common.BinderService,
	deserializerManager driver3.DeserializerManager,
	identityDB driver2.IdentityDB,
	mspID string,
	ignoreVerifyOnlyWallet bool,
) *LocalMembership {
	return &LocalMembership{
		config:                   config,
		defaultNetworkIdentity:   defaultNetworkIdentity,
		signerService:            signerService,
		binderService:            binderService,
		deserializerManager:      deserializerManager,
		identityDB:               identityDB,
		mspID:                    mspID,
		bccspResolversByIdentity: map[string]*common.Resolver{},
		resolversByEnrollmentID:  map[string]*common.Resolver{},
		resolversByName:          map[string]*common.Resolver{},
		ignoreVerifyOnlyWallet:   ignoreVerifyOnlyWallet,
	}
}

func (lm *LocalMembership) Load(identities []*config.Identity) error {
	logger.Debugf("Load x509 Wallets: [%+q]", identities)

	// load identities from configuration
	for _, identityConfig := range identities {
		logger.Debugf("Load x509 Wallet: [%v]", identityConfig)
		if err := lm.registerIdentity(nil, identityConfig, identityConfig.Default); err != nil {
			return errors.WithMessage(err, "failed to load identity")
		}
	}

	// load identities from storage
	logger.Debugf("load identities from storage...")
	if err := lm.loadFromStorage(); err != nil {
		return errors.Wrapf(err, "failed to load identities from storage")
	}
	logger.Debugf("load identities from storage...done")

	// if no default identity, use the first one
	defaultIdentifier := lm.getDefaultIdentifier()
	if len(defaultIdentifier) == 0 {
		logger.Warnf("no default identity, use the first one available")
		if len(lm.resolvers) > 0 {
			logger.Warnf("set default identity to %s", lm.resolvers[0].Name)
			lm.resolvers[0].Default = true
		} else {
			logger.Warnf("cannot set default identity, no identity available")
		}
	} else {
		logger.Debugf("default identifier is [%s]", defaultIdentifier)
	}
	return nil
}

func (lm *LocalMembership) DefaultNetworkIdentity() driver.Identity {
	return lm.defaultNetworkIdentity
}

func (lm *LocalMembership) IsMe(id driver.Identity) bool {
	return lm.signerService.IsMe(id)
}

func (lm *LocalMembership) GetIdentifier(id driver.Identity) (string, error) {
	label := id.String()
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get identity info by label [%s]", label)
	}
	r := lm.getResolver(label)
	if r == nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("identity info not found for label [%s][%v]", label, lm.resolversByName)
		}
		return "", errors.New("not found")
	}
	return r.Name, nil
}

func (lm *LocalMembership) GetDefaultIdentifier() string {
	lm.resolversMutex.RLock()
	defer lm.resolversMutex.RUnlock()
	return lm.getDefaultIdentifier()
}

func (lm *LocalMembership) getDefaultIdentifier() string {
	for _, resolver := range lm.resolvers {
		if resolver.Default {
			return resolver.Name
		}
	}
	return ""
}

func (lm *LocalMembership) GetIdentityInfo(label string, auditInfo []byte) (driver.IdentityInfo, error) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get identity info by label [%s]", label)
	}
	r := lm.getResolver(label)
	if r == nil {
		return nil, errors.Errorf("identity info not found for label [%s][%v]", label, lm.resolversByName)
	}

	return common.NewIdentityInfo(
		r.Name,
		r.EnrollmentID,
		r.Remote,
		func() (driver.Identity, []byte, error) {
			return r.GetIdentity(nil)
		},
	), nil
}

func (lm *LocalMembership) RegisterIdentity(idConfig driver.IdentityConfiguration) error {
	identityConfig := &config.Identity{
		ID:   idConfig.ID,
		Path: idConfig.URL,
	}
	if len(idConfig.Config) != 0 {
		// load opts as yaml
		if err := yaml.Unmarshal(idConfig.Config, &identityConfig.Opts); err != nil {
			return errors.Wrapf(err, "failed to load options for [%s]", idConfig.ID)
		}
	}
	var mspConfig *m.MSPConfig
	if len(idConfig.Raw) != 0 {
		// load raw as mspConfig
		mspConfig = &m.MSPConfig{}
		if err := proto.Unmarshal(idConfig.Raw, mspConfig); err != nil {
			return errors.Wrapf(err, "failed to load msp config [%s]", idConfig.ID)
		}
	}
	return lm.registerIdentity(mspConfig, identityConfig, lm.getDefaultIdentifier() == "")
}

func (lm *LocalMembership) IDs() ([]string, error) {
	var ids []string
	for _, resolver := range lm.resolvers {
		ids = append(ids, resolver.Name)
	}
	return ids, nil
}

func (lm *LocalMembership) registerIdentity(conf *m.MSPConfig, c *config.Identity, setDefault bool) error {
	// Try to register the MSP provider
	translatedPath := lm.config.TranslatePath(c.Path)
	if err := lm.registerProvider(conf, c, translatedPath, setDefault); err != nil {
		// Does path correspond to a holder containing multiple MSP identities?
		if err := lm.registerProviders(c, translatedPath); err != nil {
			return errors.WithMessage(err, "failed to register MSP provider")
		}
	}
	return nil
}

func (lm *LocalMembership) registerProvider(conf *m.MSPConfig, identityConfig *config.Identity, translatedPath string, setDefault bool) error {
	opts, err := msp.ToBCCSPOpts(identityConfig.Opts)
	if err != nil {
		return errors.WithMessage(err, "failed to extract BCCSP options")
	}
	if opts == nil {
		logger.Debugf("no BCCSP options set for [%s], opts [%v]", identityConfig.ID, identityConfig.Opts)
	} else {
		logger.Debugf("BCCSP options set for [%s] to [%v:%v:%v]", identityConfig.ID, opts, opts.PKCS11, opts.SW)
	}

	keyStorePath := lm.keyStorePath(translatedPath)
	logger.Debugf("load provider at [%s][%s]", translatedPath, keyStorePath)
	// Try without "msp"
	provider, conf, err := NewProviderFromConf(conf, translatedPath, keyStorePath, lm.mspID, lm.signerService, opts)
	if err != nil {
		logger.Debugf("failed loading provider at [%s]: [%s]", translatedPath, err)
		// Try with "msp"
		provider, conf, err = NewProviderFromConf(conf, filepath.Join(translatedPath, "msp"), keyStorePath, lm.mspID, lm.signerService, opts)
		if err != nil {
			logger.Debugf("failed loading provider at [%s]: [%s]", filepath.Join(translatedPath, "msp"), err)
			return err
		}
	}

	if err := lm.addResolver(identityConfig, provider, setDefault); err != nil {
		return err
	}

	if exists, _ := lm.identityDB.ConfigurationExists(identityConfig.ID, IdentityConfigurationType); !exists {
		optsRaw, err := yaml.Marshal(identityConfig.Opts)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal config [%v]", identityConfig)
		}
		confRaw, err := proto.Marshal(conf)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal msp config [%v]", identityConfig)
		}
		if err := lm.identityDB.AddConfiguration(driver2.IdentityConfiguration{
			ID:     identityConfig.ID,
			Type:   IdentityConfigurationType,
			URL:    identityConfig.Path,
			Config: optsRaw,
			Raw:    confRaw,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (lm *LocalMembership) registerProviders(c *config.Identity, translatedPath string) error {
	entries, err := os.ReadDir(translatedPath)
	if err != nil {
		logger.Warnf("failed reading from [%s]: [%s]", translatedPath, err)
		return nil
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		if err := lm.registerProvider(nil, c, filepath.Join(translatedPath, id), false); err != nil {
			logger.Errorf("failed registering msp provider [%s]: [%s]", id, err)
		}
	}
	return nil
}

func (lm *LocalMembership) addResolver(identityConfig *config.Identity, provider *Provider, defaultID bool) error {
	lm.resolversMutex.Lock()
	defer lm.resolversMutex.Unlock()

	eID := provider.EnrollmentID()
	walletId, _, err := provider.Identity(nil)
	if err != nil {
		return errors.WithMessagef(err, "failed to get wallet identity from [%s]", identityConfig.ID)
	}
	logger.Debugf("adding x509 wallet resolver [%s:%s:%s]", identityConfig.ID, eID, walletId)

	// deserializer
	lm.deserializerManager.AddDeserializer(provider)

	// bind
	if lm.binderService != nil {
		if err := lm.binderService.Bind(lm.defaultNetworkIdentity, walletId, false); err != nil {
			return errors.WithMessagef(err, "cannot bind identity for [%s,%s]", walletId, eID)
		}
	}

	resolver := &common.Resolver{
		Name:         identityConfig.ID,
		Default:      defaultID,
		EnrollmentID: eID,
		GetIdentity:  provider.Identity,
		Remote:       provider.IsRemote(),
	}
	lm.bccspResolversByIdentity[walletId.String()] = resolver
	lm.resolversByName[identityConfig.ID] = resolver
	if len(eID) != 0 {
		lm.resolversByEnrollmentID[eID] = resolver
	}
	lm.resolvers = append(lm.resolvers, resolver)

	logger.Debugf("adding x509 wallet resolver [%s:%s:%s] done", identityConfig.ID, eID, walletId)
	return nil
}

func (lm *LocalMembership) getResolver(label string) *common.Resolver {
	lm.resolversMutex.RLock()
	defer lm.resolversMutex.RUnlock()

	r, ok := lm.resolversByName[label]
	if ok {
		return r
	}

	r, ok = lm.bccspResolversByIdentity[label]
	if ok {
		return r
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("identity info not found for label [%s][%v]", label, lm.bccspResolversByIdentity)
	}
	return nil
}

func (lm *LocalMembership) keyStorePath(translatedPath string) string {
	if !lm.ignoreVerifyOnlyWallet {
		return ""
	}

	path := filepath.Join(translatedPath, KeystoreFullFolder)
	if _, err := os.Stat(path); err == nil {
		return path
	}

	path = filepath.Join(translatedPath, "msp", KeystoreFullFolder)
	if _, err := os.Stat(path); err == nil {
		return path
	}

	return ""
}

func (lm *LocalMembership) loadFromStorage() error {
	it, err := lm.identityDB.IteratorConfigurations(IdentityConfigurationType)
	if err != nil {
		return errors.WithMessage(err, "failed to get registered identities from kvs")
	}
	// copy the iterator
	items := make([]driver2.IdentityConfiguration, 0)
	for it.HasNext() {
		item, err := it.Next()
		if err != nil {
			return err
		}
		items = append(items, item)
	}
	it.Close()
	for _, entry := range items {
		id := entry.ID
		if lm.getResolver(id) != nil {
			continue
		}
		if err := lm.RegisterIdentity(driver.IdentityConfiguration{
			ID:     id,
			URL:    entry.URL,
			Config: entry.Config,
			Raw:    entry.Raw,
		}); err != nil {
			return err
		}
	}
	return nil
}
