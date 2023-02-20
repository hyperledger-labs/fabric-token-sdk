/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"io/ioutil"
	"path/filepath"
	"sync"

	math3 "github.com/IBM/mathlib"
	idemix2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/idemix"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	"github.com/hyperledger/fabric/msp"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

const (
	MSP = "idemix"
)

type LocalMembership struct {
	sp                     view2.ServiceProvider
	configManager          config.Manager
	defaultNetworkIdentity view.Identity
	signerService          common.SignerService
	binderService          common.BinderService
	deserializerManager    common.DeserializerManager
	kvs                    common.KVS
	mspID                  string
	cacheSize              int

	resolversMutex          sync.RWMutex
	resolvers               []*common.Resolver
	resolversByName         map[string]*common.Resolver
	resolversByEnrollmentID map[string]*common.Resolver
}

func NewLocalMembership(
	sp view2.ServiceProvider,
	configManager config.Manager,
	defaultNetworkIdentity view.Identity,
	signerService common.SignerService,
	binderService common.BinderService,
	deserializerManager common.DeserializerManager,
	kvs common.KVS,
	mspID string,
	cacheSize int,
) *LocalMembership {
	return &LocalMembership{
		sp:                      sp,
		configManager:           configManager,
		defaultNetworkIdentity:  defaultNetworkIdentity,
		signerService:           signerService,
		binderService:           binderService,
		deserializerManager:     deserializerManager,
		kvs:                     kvs,
		mspID:                   mspID,
		cacheSize:               cacheSize,
		resolversByEnrollmentID: map[string]*common.Resolver{},
		resolversByName:         map[string]*common.Resolver{},
	}
}

func (lm *LocalMembership) Load(identities []*config.Identity) error {
	logger.Debugf("Load Idemix Wallets: [%+q]", identities)

	// load identities from configuration
	for _, identityConfig := range identities {
		logger.Debugf("loadWallet: %+v", identityConfig)
		if err := lm.registerIdentity(identityConfig.ID, identityConfig.Path, identityConfig.Default); err != nil {
			return errors.WithMessage(err, "failed to load identity")
		}
	}

	// load identity from KVS
	if err := lm.loadFromKVS(); err != nil {
		return errors.Wrapf(err, "failed to load identity from KVS")
	}

	// if no default identity, use the first one
	if len(lm.GetDefaultIdentifier()) == 0 {
		logger.Warnf("no default identity, use the first one available")
		if len(lm.resolvers) > 0 {
			logger.Warnf("set default identity to %s", lm.resolvers[0].Name)
			lm.resolvers[0].Default = true
		} else {
			logger.Warnf("cannot set default identity, no identity available")
		}
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
	label := string(id)
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get anonymous identity info by label [%s]", label)
	}
	r := lm.getResolver(label)
	if r == nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("anonymous identity info not found for label [%s][%v]", label, lm.resolversByName)
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
		logger.Debugf("get anonymous identity info by label [%s]", label)
	}
	r := lm.getResolver(label)
	if r == nil {
		return nil, errors.Errorf("anonymous identity info not found for label [%s][%v]", label, lm.resolversByName)
	}

	return common.NewIdentityInfo(
		r.Name,
		r.EnrollmentID,
		func() (view.Identity, []byte, error) {
			return r.GetIdentity(&driver2.IdentityOptions{
				EIDExtension: true,
				AuditInfo:    auditInfo,
			})
		},
	), nil
}

func (lm *LocalMembership) RegisterIdentity(id string, path string) error {
	if err := lm.storeEntryInKVS(id, path); err != nil {
		return err
	}
	return lm.registerIdentity(id, path, lm.GetDefaultIdentifier() == "")
}

func (lm *LocalMembership) registerIdentity(id string, path string, setDefault bool) error {
	// Try to register the MSP provider
	translatedPath := lm.configManager.TranslatePath(path)
	curveID := math3.BN254
	if err := lm.registerMSPProvider(id, translatedPath, curveID, setDefault); err != nil {
		// Does path correspond to a holder containing multiple MSP identities?
		if err := lm.registerMSPProviders(translatedPath, curveID); err != nil {
			return errors.WithMessage(err, "failed to register MSP provider")
		}
	}
	return nil
}

func (lm *LocalMembership) registerMSPProvider(id, translatedPath string, curveID math3.CurveID, setDefault bool) error {
	conf, err := msp.GetLocalMspConfigWithType(translatedPath, nil, lm.mspID, MSP)
	if err != nil {
		return errors.Wrapf(err, "failed reading idemix msp configuration from [%s]", translatedPath)
	}
	// TODO: remove the need for ServiceProvider
	provider, err := idemix2.NewAnyProviderWithCurve(conf, lm.sp, curveID)
	if err != nil {
		return errors.Wrapf(err, "failed instantiating idemix msp provider from [%s]", translatedPath)
	}

	cacheSize, err := lm.cacheSizeForID(id)
	if err != nil {
		return err
	}

	lm.deserializerManager.AddDeserializer(provider)
	lm.addResolver(id, provider.EnrollmentID(), setDefault, NewIdentityCache(provider.Identity, cacheSize).Identity)
	logger.Debugf("added %s resolver for id %s with cache of size %d", MSP, id+"@"+provider.EnrollmentID(), cacheSize)
	return nil
}

func (lm *LocalMembership) registerMSPProviders(translatedPath string, curveID math3.CurveID) error {
	entries, err := ioutil.ReadDir(translatedPath)
	if err != nil {
		logger.Warnf("failed reading from [%s]: [%s]", translatedPath, err)
		return nil
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		if err := lm.registerMSPProvider(id, filepath.Join(translatedPath, id), curveID, false); err != nil {
			logger.Errorf("failed registering msp provider [%s]: [%s]", id, err)
		}
	}
	return nil
}

func (lm *LocalMembership) addResolver(Name string, EnrollmentID string, defaultID bool, IdentityGetter common.GetIdentityFunc) {
	lm.resolversMutex.Lock()
	defer lm.resolversMutex.Unlock()

	resolver := &common.Resolver{
		Name:         Name,
		Default:      defaultID,
		EnrollmentID: EnrollmentID,
		GetIdentity:  IdentityGetter,
	}
	lm.resolversByName[Name] = resolver
	if len(EnrollmentID) != 0 {
		lm.resolversByEnrollmentID[EnrollmentID] = resolver
	}
	lm.resolvers = append(lm.resolvers, resolver)
}

func (lm *LocalMembership) getResolver(label string) *common.Resolver {
	lm.resolversMutex.RLock()
	defer lm.resolversMutex.RUnlock()

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get anonymous identity info by label [%s]", label)
	}
	r, ok := lm.resolversByName[label]
	if ok {
		return r
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("anonymous identity info not found for label [%s][%v]", label, lm.resolversByName)
	}
	return nil
}

func (lm *LocalMembership) cacheSizeForID(id string) (int, error) {
	tmss, err := config2.NewTokenSDK(view2.GetConfigService(lm.sp)).GetTMSs()
	if err != nil {
		return 0, errors.WithMessage(err, "failed to obtain token management system instances")
	}

	for _, tms := range tmss {
		for _, owner := range tms.TMS().Wallets.Owners {
			if owner.ID == id {
				logger.Debugf("Cache size for %s is set to be %d", id, owner.CacheSize)
				return owner.CacheSize, nil
			}
		}
	}

	logger.Debugf("cache size for %s not configured, using default (%d)", id, lm.cacheSize)

	return lm.cacheSize, nil
}

func (lm *LocalMembership) storeEntryInKVS(id string, path string) error {
	k, err := kvs.CreateCompositeKey("fabric-sdk", []string{"msp", "idemix", "registeredIdentity", id})
	if err != nil {
		return errors.Wrapf(err, "failed to create identity key")
	}
	return lm.kvs.Put(k, path)
}

func (lm *LocalMembership) loadFromKVS() error {
	it, err := lm.kvs.GetByPartialCompositeID("fabric-sdk", []string{"msp", "idemix", "registeredIdentity"})
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

		if err := lm.registerIdentity(id, path, lm.GetDefaultIdentifier() == ""); err != nil {
			return err
		}
	}
	return nil
}
