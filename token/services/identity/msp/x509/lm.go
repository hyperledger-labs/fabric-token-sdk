/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/deserializer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/common"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/config"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

const (
	KeystoreFullFolder        = "keystoreFull"
	PrivateKeyFileName        = "priv_sk"
	KeystoreFolder            = "keystore"
	IdentityConfigurationType = "x509"
)

type LocalMembership struct {
	config                 config2.Config
	defaultNetworkIdentity view.Identity
	signerService          common.SigService
	binderService          common.BinderService
	deserializerManager    deserializer.Manager
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
	defaultNetworkIdentity view.Identity,
	signerService common.SigService,
	binderService common.BinderService,
	deserializerManager deserializer.Manager,
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
		if err := lm.registerIdentity(identityConfig, identityConfig.Default); err != nil {
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
	defaultIdentifier := lm.GetDefaultIdentifier()
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

func (lm *LocalMembership) DefaultNetworkIdentity() view.Identity {
	return lm.defaultNetworkIdentity
}

func (lm *LocalMembership) IsMe(id view.Identity) bool {
	return lm.signerService.IsMe(id)
}

func (lm *LocalMembership) GetIdentifier(id view.Identity) (string, error) {
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
		func() (view.Identity, []byte, error) {
			return r.GetIdentity(nil)
		},
	), nil
}

func (lm *LocalMembership) RegisterIdentity(id string, path string) error {
	return lm.registerIdentity(&config.Identity{
		ID:   id,
		Path: path,
	}, lm.GetDefaultIdentifier() == "")
}

func (lm *LocalMembership) IDs() ([]string, error) {
	var ids []string
	for _, resolver := range lm.resolvers {
		ids = append(ids, resolver.Name)
	}
	return ids, nil
}

func (lm *LocalMembership) registerIdentity(c *config.Identity, setDefault bool) error {
	// Try to register the MSP provider
	translatedPath := lm.config.TranslatePath(c.Path)
	if err := lm.registerProvider(c, translatedPath, setDefault); err != nil {
		// Does path correspond to a holder containing multiple MSP identities?
		if err := lm.registerProviders(c, translatedPath); err != nil {
			return errors.WithMessage(err, "failed to register MSP provider")
		}
	}
	return nil
}

func (lm *LocalMembership) registerProvider(identityConfig *config.Identity, translatedPath string, setDefault bool) error {
	// Try without "msp"
	opts, err := ToBCCSPOpts(identityConfig.Opts)
	if err != nil {
		return errors.WithMessage(err, "failed to extract BCCSP options")
	}
	if opts == nil {
		logger.Debugf("no BCCSP options set for [%s]: [%v]", identityConfig.ID, identityConfig.Opts)
	} else {
		logger.Debugf("BCCSP options set for [%s] to [%v:%v:%v]", identityConfig.ID, opts, opts.PKCS11, opts.SW)
	}

	keyStorePath := ""
	if lm.ignoreVerifyOnlyWallet {
		// check if there is the folder keystoreFull, if yes then use it
		path := filepath.Join(translatedPath, KeystoreFullFolder)
		_, err := os.Stat(path)
		if err == nil {
			keyStorePath = path
		} else {
			path := filepath.Join(translatedPath, "msp", KeystoreFullFolder)
			_, err := os.Stat(path)
			if err == nil {
				keyStorePath = path
			}
		}
	}

	logger.Debugf("load provider at [%s][%s]", translatedPath, keyStorePath)
	provider, err := NewProviderWithBCCSPConfig(translatedPath, keyStorePath, lm.mspID, lm.signerService, opts)
	if err != nil {
		logger.Debugf("failed reading bccsp msp configuration from [%s]: [%s]", filepath.Join(translatedPath), err)
		// Try with "msp"
		provider, err = NewProviderWithBCCSPConfig(filepath.Join(translatedPath, "msp"), keyStorePath, lm.mspID, lm.signerService, opts)
		if err != nil {
			logger.Warnf("failed reading bccsp msp configuration from [%s and %s]: [%s]",
				filepath.Join(translatedPath), filepath.Join(translatedPath, "msp"), err,
			)
			return err
		}
	}

	walletId, _, err := provider.Identity(nil)
	if err != nil {
		return errors.WithMessagef(err, "failed to get wallet identity from [%s:%s]", identityConfig.ID, translatedPath)
	}

	logger.Debugf("Adding x509 wallet resolver [%s:%s:%s]", identityConfig.ID, provider.EnrollmentID(), walletId.String())

	lm.deserializerManager.AddDeserializer(provider)
	if err := lm.addResolver(identityConfig.ID, provider.EnrollmentID(), setDefault, provider.IsRemote(), provider.Identity); err != nil {
		return err
	}

	if exists, _ := lm.identityDB.ConfigurationExists(identityConfig.ID, IdentityConfigurationType); !exists {
		identityConfigRaw, err := json.Marshal(identityConfig)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal config [%v]", identityConfig)
		}
		if err := lm.identityDB.AddConfiguration(driver2.IdentityConfiguration{
			ID:     identityConfig.ID,
			Type:   IdentityConfigurationType,
			URL:    identityConfig.Path,
			Config: identityConfigRaw,
			Raw:    nil,
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
		if err := lm.registerProvider(c, filepath.Join(translatedPath, id), false); err != nil {
			logger.Errorf("failed registering msp provider [%s]: [%s]", id, err)
		}
	}
	return nil
}

func (lm *LocalMembership) addResolver(id string, eID string, defaultID bool, remote bool, IdentityGetter common.GetIdentityFunc) error {
	logger.Debugf("Adding resolver [%s:%s]", id, eID)
	lm.resolversMutex.Lock()
	defer lm.resolversMutex.Unlock()

	if lm.binderService != nil {
		id, _, err := IdentityGetter(nil)
		if err != nil {
			return errors.WithMessagef(err, "cannot get identity for [%s,%s]", id, eID)
		}
		if err := lm.binderService.Bind(lm.defaultNetworkIdentity, id); err != nil {
			return errors.WithMessagef(err, "cannot bing identity for [%s,%s]", id, eID)
		}
	}

	resolver := &common.Resolver{
		Name:         id,
		Default:      defaultID,
		EnrollmentID: eID,
		GetIdentity:  IdentityGetter,
		Remote:       remote,
	}
	identity, _, err := IdentityGetter(nil)
	if err != nil {
		return errors.WithMessagef(err, "cannot get identity for [%s,%s]", id, eID)
	}
	lm.bccspResolversByIdentity[identity.String()] = resolver
	lm.resolversByName[id] = resolver
	if len(eID) != 0 {
		lm.resolversByEnrollmentID[eID] = resolver
	}
	lm.resolvers = append(lm.resolvers, resolver)

	logger.Debugf("Adding resolver [%s:%s] done", id, eID)
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

func (lm *LocalMembership) loadFromStorage() error {
	it, err := lm.identityDB.IteratorConfigurations(IdentityConfigurationType)
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
		if lm.getResolver(id) != nil {
			continue
		}
		identityConfig := &config.Identity{
			ID:   id,
			Path: entry.URL,
			Type: IdentityConfigurationType,
		}
		if len(entry.Config) != 0 {
			if err := json.Unmarshal([]byte(entry.Config), identityConfig); err != nil {
				logger.Errorf("failed to load configuration for entry [%s]", entry.ID)
				continue
			}
			if identityConfig.ID != id || identityConfig.Path != entry.URL || identityConfig.Type != IdentityConfigurationType {
				logger.Errorf("invalid configuration for entry [%s], it does not match the expected values [%v][%v]", entry.ID, entry, identityConfig)
				continue
			}
		}
		if err := lm.registerIdentity(identityConfig, lm.GetDefaultIdentifier() == ""); err != nil {
			return err
		}
	}
	return nil
}
