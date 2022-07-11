/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/x509"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp/common"
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
	mspID                  string

	resolversMutex           sync.RWMutex
	resolvers                []*common.Resolver
	resolversByName          map[string]*common.Resolver
	resolversByEnrollmentID  map[string]*common.Resolver
	bccspResolversByIdentity map[string]*common.Resolver
}

func NewLocalMembership(
	configManager config.Manager,
	defaultNetworkIdentity view.Identity,
	signerService common.SignerService,
	binderService common.BinderService,
	deserializerManager common.DeserializerManager,
	mspID string,
) *LocalMembership {
	return &LocalMembership{
		configManager:            configManager,
		defaultNetworkIdentity:   defaultNetworkIdentity,
		signerService:            signerService,
		binderService:            binderService,
		deserializerManager:      deserializerManager,
		mspID:                    mspID,
		bccspResolversByIdentity: map[string]*common.Resolver{},
		resolversByEnrollmentID:  map[string]*common.Resolver{},
		resolversByName:          map[string]*common.Resolver{},
	}
}

func (lm *LocalMembership) Load(identities []*config.Identity) error {
	logger.Debugf("Load x509 Wallets: [%+q]", identities)

	type Provider interface {
		EnrollmentID() string
		Identity(opts *fabric.IdentityOption) (view.Identity, []byte, error)
		DeserializeVerifier(raw []byte) (driver.Verifier, error)
		DeserializeSigner(raw []byte) (driver.Signer, error)
		Info(raw []byte, auditInfo []byte) (string, error)
	}

	for _, identityConfig := range identities {
		logger.Debugf("Load x509 Wallet: [%v]", identityConfig)
		if err := lm.registerIdentity(identityConfig.ID, identityConfig.Path, identityConfig.Default); err != nil {
			return errors.WithMessage(err, "failed to load identity")
		}
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
	return lm.registerIdentity(id, path, lm.GetDefaultIdentifier() == "")
}

func (lm *LocalMembership) registerIdentity(id string, path string, setDefault bool) error {
	// Try to register the MSP provider
	translatedPath := lm.configManager.TranslatePath(path)
	if err := lm.registerMSPProvider(id, translatedPath, setDefault); err != nil {
		// Does path correspond to a holder containing multiple MSP identities?
		if err := lm.registerMSPProviders(translatedPath); err != nil {
			return errors.WithMessage(err, "failed to register MSP provider")
		}
	}
	return nil
}

func (lm *LocalMembership) registerMSPProvider(id, translatedPath string, setDefault bool) error {
	// Try without "msp"
	provider, err := x509.NewProvider(filepath.Join(translatedPath), lm.mspID, lm.signerService)
	if err != nil {
		logger.Debugf("failed reading bccsp msp configuration from [%s]: [%s]", filepath.Join(translatedPath), err)
		// Try with "msp"
		provider, err = x509.NewProvider(filepath.Join(translatedPath, "msp"), lm.mspID, lm.signerService)
		if err != nil {
			logger.Warnf("failed reading bccsp msp configuration from [%s and %s]: [%s]",
				filepath.Join(translatedPath), filepath.Join(translatedPath, "msp"), err,
			)
			return err
		}
	}

	walletId, _, err := provider.Identity(nil)
	if err != nil {
		return errors.WithMessagef(err, "failed to get wallet identity from [%s:%s]", id, translatedPath)
	}

	logger.Debugf("Adding x509 wallet resolver [%s:%s:%s]", id, provider.EnrollmentID(), walletId.String())
	lm.deserializerManager.AddDeserializer(provider)
	lm.addResolver(id, provider.EnrollmentID(), setDefault, provider.Identity)

	return nil
}

func (lm *LocalMembership) registerMSPProviders(translatedPath string) error {
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
		if err := lm.registerMSPProvider(id, filepath.Join(translatedPath, id), false); err != nil {
			logger.Errorf("failed registering msp provider [%s]: [%s]", id, err)
		}
	}
	return nil
}

func (lm *LocalMembership) addResolver(id string, eID string, defaultID bool, IdentityGetter common.GetIdentityFunc) {
	logger.Debugf("Adding resolver [%s:%s]", id, eID)
	lm.resolversMutex.Lock()
	defer lm.resolversMutex.Unlock()

	if lm.binderService != nil {
		id, _, err := IdentityGetter(nil)
		if err != nil {
			panic(fmt.Sprintf("cannot get identity for [%s,%s][%s]", id, eID, err))
		}
		if err := lm.binderService.Bind(lm.defaultNetworkIdentity, id); err != nil {
			panic(fmt.Sprintf("cannot bing identity for [%s,%s][%s]", id, eID, err))
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
		panic(fmt.Sprintf("cannot get identity for [%s,%s][%s]", id, eID, err))
	}
	lm.bccspResolversByIdentity[identity.String()] = resolver
	lm.resolversByName[id] = resolver
	if len(eID) != 0 {
		lm.resolversByEnrollmentID[eID] = resolver
	}
	lm.resolvers = append(lm.resolvers, resolver)
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
