/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/x509"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp/common"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type LocalMembership struct {
	configManager          config.Manager
	defaultNetworkIdentity view.Identity
	signerService          common.SignerService
	binderService          common.BinderService
	deserializerManager    common.DeserializerManager
	kvs                    common.KVS
	mspID                  string

	resolversMutex           sync.RWMutex
	resolvers                []*common.Resolver
	resolversByName          map[string]*common.Resolver
	resolversByEnrollmentID  map[string]*common.Resolver
	bccspResolversByIdentity map[string]*common.Resolver
	ignoreRemote             bool
}

func NewLocalMembership(
	configManager config.Manager,
	defaultNetworkIdentity view.Identity,
	signerService common.SignerService,
	binderService common.BinderService,
	deserializerManager common.DeserializerManager,
	kvs common.KVS,
	mspID string,
	ignoreRemote bool,
) *LocalMembership {
	return &LocalMembership{
		configManager:            configManager,
		defaultNetworkIdentity:   defaultNetworkIdentity,
		signerService:            signerService,
		binderService:            binderService,
		deserializerManager:      deserializerManager,
		kvs:                      kvs,
		mspID:                    mspID,
		bccspResolversByIdentity: map[string]*common.Resolver{},
		resolversByEnrollmentID:  map[string]*common.Resolver{},
		resolversByName:          map[string]*common.Resolver{},
		ignoreRemote:             ignoreRemote,
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

	// load identity from KVS
	if err := lm.loadFromKVS(); err != nil {
		return errors.Wrapf(err, "failed to load identity from KVS")
	}

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

	return common.NewIdentityInfo(r.Name, r.EnrollmentID, func() (view.Identity, []byte, error) {
		return r.GetIdentity(nil)
	}), nil
}

func (lm *LocalMembership) RegisterIdentity(id string, path string) error {
	if err := lm.storeEntryInKVS(id, path); err != nil {
		return err
	}
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
	translatedPath := lm.configManager.TranslatePath(c.Path)
	if err := lm.registerProvider(c, translatedPath, setDefault); err != nil {
		// Does path correspond to a holder containing multiple MSP identities?
		if err := lm.registerProviders(c, translatedPath); err != nil {
			return errors.WithMessage(err, "failed to register MSP provider")
		}
	}
	return nil
}

func (lm *LocalMembership) registerProvider(identity *config.Identity, translatedPath string, setDefault bool) error {
	logger.Infof("register provider with type [%s]", identity.Type)
	switch identity.Type {
	case "remote":
		// do nothing for now
		if lm.ignoreRemote {
			return lm.registerLocalProvider(identity, translatedPath, setDefault)
		}
		return lm.registerRemoteProvider(identity, translatedPath, setDefault)
	default:
		return lm.registerLocalProvider(identity, translatedPath, setDefault)
	}
}

func (lm *LocalMembership) registerLocalProvider(c *config.Identity, translatedPath string, setDefault bool) error {
	// Try without "msp"
	opts, err := config2.ToBCCSPOpts(c.Opts)
	if err != nil {
		return errors.WithMessage(err, "failed to extract BCCSP options")
	}
	if opts == nil {
		logger.Debugf("no BCCSP options set for [%s]: [%v]", c.ID, c.Opts)
	} else {
		logger.Debugf("BCCSP options set for [%s] to [%v:%v:%v]", c.ID, opts, opts.PKCS11, opts.SW)
	}
	provider, err := x509.NewProviderWithBCCSPConfig(filepath.Join(translatedPath), lm.mspID, lm.signerService, opts)
	if err != nil {
		logger.Debugf("failed reading bccsp msp configuration from [%s]: [%s]", filepath.Join(translatedPath), err)
		// Try with "msp"
		provider, err = x509.NewProviderWithBCCSPConfig(filepath.Join(translatedPath, "msp"), lm.mspID, lm.signerService, opts)
		if err != nil {
			logger.Warnf("failed reading bccsp msp configuration from [%s and %s]: [%s]",
				filepath.Join(translatedPath), filepath.Join(translatedPath, "msp"), err,
			)
			return err
		}
	}

	walletId, _, err := provider.Identity(nil)
	if err != nil {
		return errors.WithMessagef(err, "failed to get wallet identity from [%s:%s]", c.ID, translatedPath)
	}

	logger.Debugf("Adding x509 wallet resolver [%s:%s:%s]", c.ID, provider.EnrollmentID(), walletId.String())
	lm.deserializerManager.AddDeserializer(provider)
	if err := lm.addResolver(c.ID, provider.EnrollmentID(), setDefault, provider.Identity); err != nil {
		return err
	}

	return nil
}

func (lm *LocalMembership) registerRemoteProvider(c *config.Identity, translatedPath string, setDefault bool) error {
	// Try without "msp"
	opts, err := config2.ToBCCSPOpts(c.Opts)
	if err != nil {
		return errors.WithMessage(err, "failed to extract BCCSP options")
	}
	if opts == nil {
		logger.Debugf("no BCCSP options set for [%s]: [%v]", c.ID, c.Opts)
	} else {
		logger.Debugf("BCCSP options set for [%s] to [%v:%v:%v]", c.ID, opts, opts.PKCS11, opts.SW)
	}
	provider, err := x509.NewProviderWithBCCSPConfig(filepath.Join(translatedPath), lm.mspID, nil, opts)
	if err != nil {
		logger.Debugf("failed reading bccsp msp configuration from [%s]: [%s]", filepath.Join(translatedPath), err)
		// Try with "msp"
		provider, err = x509.NewProviderWithBCCSPConfig(filepath.Join(translatedPath, "msp"), lm.mspID, nil, opts)
		if err != nil {
			logger.Warnf("failed reading bccsp msp configuration from [%s and %s]: [%s]",
				filepath.Join(translatedPath), filepath.Join(translatedPath, "msp"), err,
			)
			return err
		}
	}

	walletId, _, err := provider.Identity(nil)
	if err != nil {
		return errors.WithMessagef(err, "failed to get wallet identity from [%s:%s]", c.ID, translatedPath)
	}

	logger.Debugf("Adding x509 wallet resolver [%s:%s:%s]", c.ID, provider.EnrollmentID(), walletId.String())
	lm.deserializerManager.AddDeserializer(provider)
	if err := lm.addResolver(c.ID, provider.EnrollmentID(), setDefault, provider.Identity); err != nil {
		return err
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

func (lm *LocalMembership) addResolver(id string, eID string, defaultID bool, IdentityGetter common.GetIdentityFunc) error {
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

func (lm *LocalMembership) storeEntryInKVS(id string, path string) error {
	k, err := kvs.CreateCompositeKey("token-sdk", []string{"msp", "x509", "registeredIdentity", id})
	if err != nil {
		return errors.Wrapf(err, "failed to create identity key")
	}
	return lm.kvs.Put(k, path)
}

func (lm *LocalMembership) loadFromKVS() error {
	it, err := lm.kvs.GetByPartialCompositeID("token-sdk", []string{"msp", "x509", "registeredIdentity"})
	if err != nil {
		return errors.WithMessage(err, "failed to get registered identities from kvs")
	}
	defer it.Close()
	for it.HasNext() {
		var path string
		k, err := it.Next(&path)
		if err != nil {
			return errors.WithMessagef(err, "failed to get next registered identities from kvs")
		}

		_, attrs, err := kvs.SplitCompositeKey(k)
		if err != nil {
			return errors.WithMessagef(err, "failed to split key [%s]", k)
		}

		id := attrs[3]
		if lm.getResolver(id) != nil {
			continue
		}

		if err := lm.registerIdentity(&config.Identity{ID: id, Path: path}, lm.GetDefaultIdentifier() == ""); err != nil {
			return err
		}
	}
	return nil
}
