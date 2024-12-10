/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/config"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
)

type KeyManagerProvider interface {
	Get(identityConfig *driver.IdentityConfiguration) (KeyManager, error)
}

type KeyManager interface {
	driver2.Deserializer
	EnrollmentID() string
	IsRemote() bool
	Anonymous() bool
	Identity([]byte) (driver.Identity, []byte, error)
}

type LocalMembership struct {
	config                 driver2.Config
	defaultNetworkIdentity driver.Identity
	signerService          driver2.SigService
	deserializerManager    driver2.DeserializerManager
	identityDB             driver3.IdentityDB
	binderService          driver2.BinderService
	KeyManagerProvider     KeyManagerProvider
	IdentityType           string
	logger                 logging.Logger

	localIdentitiesMutex          sync.RWMutex
	localIdentities               []*LocalIdentity
	localIdentitiesByName         map[string]*LocalIdentity
	localIdentitiesByEnrollmentID map[string]*LocalIdentity
	localIdentitiesByIdentity     map[string]*LocalIdentity
}

func NewLocalMembership(
	logger logging.Logger,
	config driver2.Config,
	defaultNetworkIdentity driver.Identity,
	signerService driver2.SigService,
	deserializerManager driver2.DeserializerManager,
	identityDB driver3.IdentityDB,
	binderService driver2.BinderService,
	identityType string,
	KeyManagerProvider KeyManagerProvider,
) *LocalMembership {
	return &LocalMembership{
		logger:                        logger,
		config:                        config,
		defaultNetworkIdentity:        defaultNetworkIdentity,
		signerService:                 signerService,
		deserializerManager:           deserializerManager,
		identityDB:                    identityDB,
		localIdentitiesByEnrollmentID: map[string]*LocalIdentity{},
		localIdentitiesByName:         map[string]*LocalIdentity{},
		localIdentitiesByIdentity:     map[string]*LocalIdentity{},
		binderService:                 binderService,
		IdentityType:                  identityType,
		KeyManagerProvider:            KeyManagerProvider,
	}
}

func (l *LocalMembership) DefaultNetworkIdentity() driver.Identity {
	return l.defaultNetworkIdentity
}

func (l *LocalMembership) IsMe(id driver.Identity) bool {
	return l.signerService.IsMe(id)
}

func (l *LocalMembership) GetIdentifier(id driver.Identity) (string, error) {
	l.localIdentitiesMutex.RLock()
	defer l.localIdentitiesMutex.RUnlock()

	for _, label := range []string{string(id), id.String()} {
		if l.logger.IsEnabledFor(zapcore.DebugLevel) {
			l.logger.Debugf("get local identity by label [%s]", label)
		}
		r := l.getLocalIdentity(label)
		if r == nil {
			if l.logger.IsEnabledFor(zapcore.DebugLevel) {
				l.logger.Debugf("local identity not found for label [%s][%v]", collections.Keys(l.localIdentitiesByName))
			}
			continue
		}
		return r.Name, nil
	}
	return "", errors.Errorf("identifier not found for id [%s]", id)
}

func (l *LocalMembership) GetDefaultIdentifier() string {
	l.localIdentitiesMutex.RLock()
	defer l.localIdentitiesMutex.RUnlock()

	return l.getDefaultIdentifier()
}

func (l *LocalMembership) GetIdentityInfo(label string, auditInfo []byte) (driver.IdentityInfo, error) {
	l.localIdentitiesMutex.RLock()
	defer l.localIdentitiesMutex.RUnlock()

	if l.logger.IsEnabledFor(zapcore.DebugLevel) {
		l.logger.Debugf("get identity info by label [%s]", hash.Hashable(label))
	}
	localIdentity := l.getLocalIdentity(label)
	if localIdentity == nil {
		return nil, errors.Errorf("local identity not found for label [%s][%v]", hash.Hashable(label), l.localIdentitiesByName)
	}
	return NewIdentityInfo(localIdentity, func() (driver.Identity, []byte, error) {
		return localIdentity.GetIdentity(auditInfo)
	}), nil
}

func (l *LocalMembership) RegisterIdentity(idConfig driver.IdentityConfiguration) error {
	l.localIdentitiesMutex.Lock()
	defer l.localIdentitiesMutex.Unlock()

	return l.registerIdentityConfiguration(&idConfig, l.getDefaultIdentifier() == "")
}

func (l *LocalMembership) IDs() ([]string, error) {
	l.localIdentitiesMutex.RLock()
	defer l.localIdentitiesMutex.RUnlock()

	var ids []string
	for _, identity := range l.localIdentities {
		ids = append(ids, identity.Name)
	}
	return ids, nil
}

func (l *LocalMembership) Load(identities []*config.Identity) error {
	l.localIdentitiesMutex.Lock()
	defer l.localIdentitiesMutex.Unlock()

	l.logger.Debugf("load identities [%+q]", identities)

	// cleanup tables
	l.localIdentities = make([]*LocalIdentity, 0)
	l.localIdentitiesByName = make(map[string]*LocalIdentity)
	l.localIdentitiesByEnrollmentID = make(map[string]*LocalIdentity)

	// load identities from configuration
	for _, identityConfig := range identities {
		l.logger.Debugf("load wallet for identity [%+v]", identityConfig)
		if err := l.registerIdentity(*identityConfig); err != nil {
			return errors.WithMessage(err, "failed to load identity")
		}
		l.logger.Debugf("load wallet for identity [%+v] done.", identityConfig)
	}

	// load identities from storage
	l.logger.Debugf("load identities from storage...")
	if err := l.loadFromStorage(); err != nil {
		return errors.Wrapf(err, "failed to load identities from identityDB")
	}
	l.logger.Debugf("load identities from storage...done")

	// if no default identity, use the first one
	defaultIdentifier := l.getDefaultIdentifier()
	if len(defaultIdentifier) == 0 {
		l.logger.Warnf("no default identity, use the first one available")
		if len(l.localIdentities) > 0 {
			l.logger.Warnf("set default identity to %s", l.localIdentities[0].Name)
			l.localIdentities[0].Default = true
		} else {
			l.logger.Warnf("cannot set default identity, no identity available")
		}
	} else {
		l.logger.Debugf("default identifier is [%s]", defaultIdentifier)
	}

	return nil
}

func (l *LocalMembership) getDefaultIdentifier() string {
	for _, identity := range l.localIdentities {
		if identity.Default {
			return identity.Name
		}
	}
	return ""
}

func (l *LocalMembership) registerIdentity(identity config.Identity) error {
	// marshal opts
	optsRaw, err := yaml.Marshal(identity.Opts)
	if err != nil {
		return errors.WithMessage(err, "failed to marshal identity options")
	}
	return l.registerIdentityConfiguration(&driver.IdentityConfiguration{
		ID:     identity.ID,
		URL:    identity.Path,
		Config: optsRaw,
		Raw:    nil,
	}, identity.Default)
}

func (l *LocalMembership) registerLocalIdentity(identityConfig *driver.IdentityConfiguration, defaultIdentity bool) error {
	provider, err := l.KeyManagerProvider.Get(identityConfig)
	if err != nil {
		return errors.Wrapf(err, "failed to get identity provider for [%s]", identityConfig.ID)
	}

	l.logger.Debugf("append local identity for [%s]", identityConfig.ID)
	if err := l.addLocalIdentity(identityConfig, provider, defaultIdentity); err != nil {
		return errors.Wrapf(err, "failed to add local identity for [%s]", identityConfig.ID)
	}

	l.logger.Debugf("does the configuration already exists for [%s]?", identityConfig.ID)
	if exists, _ := l.identityDB.ConfigurationExists(identityConfig.ID, l.IdentityType); !exists {
		l.logger.Debugf("does the configuration already exists for [%s]? no, add it", identityConfig.ID)
		if err := l.identityDB.AddConfiguration(driver3.IdentityConfiguration{
			ID:     identityConfig.ID,
			Type:   l.IdentityType,
			URL:    identityConfig.URL,
			Config: identityConfig.Config,
			Raw:    identityConfig.Raw,
		}); err != nil {
			return err
		}
	}
	l.logger.Debugf("added local identity for id [%s], remote [%v]", identityConfig.ID+"@"+provider.EnrollmentID(), provider.IsRemote())
	return nil
}

func (l *LocalMembership) registerIdentityConfiguration(identity *driver.IdentityConfiguration, defaultIdentity bool) error {
	// Try to register the local identity
	identity.URL = l.config.TranslatePath(identity.URL)
	if err := l.registerLocalIdentity(identity, defaultIdentity); err != nil {
		l.logger.Warnf("failed to load local identity at [%s]:[%s]", identity.URL, err)
		// Does path correspond to a folder containing multiple identities?
		if err := l.registerLocalIdentities(identity); err != nil {
			// we don't return the error so that the token manager can still load
			l.logger.Errorf("failed to register local identity from folder: %s", err.Error())
		}
	}
	return nil
}

func (l *LocalMembership) registerLocalIdentities(configuration *driver.IdentityConfiguration) error {
	entries, err := os.ReadDir(configuration.URL)
	if err != nil {
		l.logger.Warnf("failed reading from [%s]: [%s]", configuration.URL, err)
		return nil
	}
	found := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		if err := l.registerLocalIdentity(&driver.IdentityConfiguration{
			ID:     id,
			URL:    filepath.Join(configuration.URL, id),
			Config: configuration.Config,
		}, false); err != nil {
			l.logger.Errorf("failed registering local identity [%s]: [%s]", id, err)
			continue
		}
		found++
	}
	if found == 0 {
		return errors.Errorf("no valid identities found in [%s]", configuration.URL)
	}
	return nil
}

func (l *LocalMembership) addLocalIdentity(config *driver.IdentityConfiguration, keyManager KeyManager, defaultID bool) error {
	eID := keyManager.EnrollmentID()
	name := config.ID
	localIdentity := &LocalIdentity{
		Name:         name,
		Default:      defaultID,
		EnrollmentID: eID,
		GetIdentity:  keyManager.Identity,
		Remote:       keyManager.IsRemote(),
	}
	l.localIdentitiesByName[name] = localIdentity
	if len(eID) != 0 {
		l.localIdentitiesByEnrollmentID[eID] = localIdentity
	}

	l.logger.Debugf("adding identity mapping for [%s] by name and eID [%s]", name, eID)

	// deserializer
	l.deserializerManager.AddDeserializer(keyManager)

	// if the keyManager is not anonymous
	if !keyManager.Anonymous() {
		identity, _, err := keyManager.Identity(nil)
		if err != nil {
			return errors.WithMessagef(err, "failed to get wallet identity from [%s]", name)
		}
		if l.logger.IsEnabledFor(zapcore.DebugLevel) {
			l.logger.Debugf("adding identity mapping for [%s]", identity)
		}
		l.localIdentitiesByIdentity[identity.String()] = localIdentity
		if l.binderService != nil {
			if err := l.binderService.Bind(l.defaultNetworkIdentity, identity, false); err != nil {
				return errors.WithMessagef(err, "cannot bind identity for [%s,%s]", identity, eID)
			}
		}
	}

	l.localIdentities = append(l.localIdentities, localIdentity)
	return nil
}

func (l *LocalMembership) getLocalIdentity(label string) *LocalIdentity {
	if l.logger.IsEnabledFor(zapcore.DebugLevel) {
		l.logger.Debugf("get local identity by label [%s]", hash.Hashable(label))
	}
	r, ok := l.localIdentitiesByName[label]
	if ok {
		return r
	}
	r, ok = l.localIdentitiesByIdentity[label]
	if ok {
		return r
	}

	if l.logger.IsEnabledFor(zapcore.DebugLevel) {
		l.logger.Debugf("local identity not found for label [%s][%v]", hash.Hashable(label), l.localIdentitiesByName)
	}
	return nil
}

func (l *LocalMembership) loadFromStorage() error {
	it, err := l.identityDB.IteratorConfigurations(l.IdentityType)
	if err != nil {
		return errors.WithMessage(err, "failed to get registered identities from kvs")
	}
	// copy the iterator
	items := make([]driver3.IdentityConfiguration, 0)
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
		if l.getLocalIdentity(id) != nil {
			l.logger.Debugf("from storage: id [%s] already exists", id)
			continue
		}
		l.logger.Debugf("from storage: id [%s] does no exist, register it", id)
		if err := l.registerIdentityConfiguration(&driver.IdentityConfiguration{
			ID:     entry.ID,
			URL:    entry.URL,
			Config: entry.Config,
			Raw:    entry.Raw,
		}, l.getDefaultIdentifier() == ""); err != nil {
			return err
		}
	}
	return nil
}
