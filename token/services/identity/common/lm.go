/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	errors3 "errors"
	"os"
	"path/filepath"
	"slices"
	"sync"

	errors2 "github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
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

type LocalIdentityWithPriority struct {
	Identity *LocalIdentity
	Priority int
}

type LocalMembership struct {
	config                 driver2.Config
	defaultNetworkIdentity driver.Identity
	signerService          driver2.SigService
	deserializerManager    driver2.DeserializerManager
	identityDB             driver3.IdentityDB
	binderService          driver2.BinderService
	KeyManagerProviders    []KeyManagerProvider
	IdentityType           string
	logger                 logging.Logger

	localIdentitiesMutex      sync.RWMutex
	localIdentities           []*LocalIdentity
	localIdentitiesByName     map[string][]LocalIdentityWithPriority
	localIdentitiesByIdentity map[string]*LocalIdentity
	targetIdentities          []view.Identity
	DefaultAnonymous          bool
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
	defaultAnonymous bool,
	keyManagerProviders ...KeyManagerProvider,
) *LocalMembership {
	return &LocalMembership{
		logger:                    logger,
		config:                    config,
		defaultNetworkIdentity:    defaultNetworkIdentity,
		signerService:             signerService,
		deserializerManager:       deserializerManager,
		identityDB:                identityDB,
		localIdentitiesByName:     map[string][]LocalIdentityWithPriority{},
		localIdentitiesByIdentity: map[string]*LocalIdentity{},
		binderService:             binderService,
		IdentityType:              identityType,
		KeyManagerProviders:       keyManagerProviders,
		DefaultAnonymous:          defaultAnonymous,
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

	set := collections.NewSet[string]()
	for _, identity := range l.localIdentities {
		set.Add(identity.Name)
	}
	return set.ToSlice(), nil
}

func (l *LocalMembership) Load(identities []*config.Identity, targets []view.Identity) error {
	l.localIdentitiesMutex.Lock()
	defer l.localIdentitiesMutex.Unlock()

	l.logger.Debugf("load identities [%+q]", identities)

	l.targetIdentities = targets

	// cleanup tables
	l.localIdentities = make([]*LocalIdentity, 0)
	l.localIdentitiesByName = make(map[string][]LocalIdentityWithPriority, 0)

	// load identities from configuration
	for _, identityConfig := range identities {
		l.logger.Debugf("load wallet for identity [%+v]", identityConfig)
		if err := l.registerIdentity(*identityConfig); err != nil {
			// we log the error so the user can fix it but it shouldn't stop the loading of the service.
			l.logger.Errorf("failed loading identity: %s", err.Error())
		} else {
			l.logger.Debugf("load wallet for identity [%+v] done.", identityConfig)
		}
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
		if l.DefaultAnonymous && !identity.Anonymous {
			continue
		}

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
	var errs []error
	var keyManager KeyManager
	var index int
	for i, p := range l.KeyManagerProviders {
		var err error
		keyManager, err = p.Get(identityConfig)
		if err == nil {
			index = i
			break
		}
		errs = append(errs, err)
	}
	if keyManager == nil {
		return errors2.Wrap(errors3.Join(errs...), "failed to get a key manager for the passed identity config")
	}

	l.logger.Debugf("append local identity for [%s]", identityConfig.ID)
	if err := l.addLocalIdentity(identityConfig, keyManager, defaultIdentity, index); err != nil {
		return errors.Wrapf(err, "failed to add local identity for [%s]", identityConfig.ID)
	}

	l.logger.Debugf("does the configuration already exists for [%s]?", identityConfig.ID)
	if err := l.identityDB.AddConfiguration(driver3.IdentityConfiguration{
		ID:     identityConfig.ID,
		Type:   l.IdentityType,
		URL:    identityConfig.URL,
		Config: identityConfig.Config,
		Raw:    identityConfig.Raw,
	}); err != nil {
		return err
	}
	l.logger.Debugf("added local identity for id [%s], remote [%v]", identityConfig.ID+"@"+keyManager.EnrollmentID(), keyManager.IsRemote())
	return nil
}

func (l *LocalMembership) registerIdentityConfiguration(identity *driver.IdentityConfiguration, defaultIdentity bool) error {
	// Try to register the local identity
	identity.URL = l.config.TranslatePath(identity.URL)
	if err := l.registerLocalIdentity(identity, defaultIdentity); err != nil {
		l.logger.Warnf("failed to load local identity at [%s]:[%s]", identity.URL, err)
		// Does path correspond to a folder containing multiple identities?
		if err := l.registerLocalIdentities(identity); err != nil {
			return errors.WithMessage(err, "failed to register local identity")
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

func (l *LocalMembership) addLocalIdentity(config *driver.IdentityConfiguration, keyManager KeyManager, defaultID bool, priority int) error {
	// check for duplicates
	name := config.ID
	if _, ok := l.localIdentitiesByName[config.ID]; !ok || keyManager.Anonymous() || len(l.targetIdentities) == 0 {
		l.logger.Debugf("no target identity check needed, skip it")
	} else if identity, _, err := keyManager.Identity(nil); err != nil {
		return err
	} else if found := slices.ContainsFunc(l.targetIdentities, identity.Equal); !found {
		l.logger.Debugf("identity [%s] not in target identities, ignore it", name)
		return nil
	}

	eID := keyManager.EnrollmentID()
	localIdentity := &LocalIdentity{
		Name:         name,
		Default:      defaultID,
		EnrollmentID: eID,
		Anonymous:    keyManager.Anonymous(),
		GetIdentity:  keyManager.Identity,
		Remote:       keyManager.IsRemote(),
	}
	list, ok := l.localIdentitiesByName[name]
	if !ok {
		list = make([]LocalIdentityWithPriority, 0)
	}
	list = append(list, LocalIdentityWithPriority{
		Identity: localIdentity,
		Priority: priority,
	})
	slices.SortFunc(list, func(a, b LocalIdentityWithPriority) int {
		if a.Priority < b.Priority {
			return -1
		} else if a.Priority > b.Priority {
			return 1
		}
		return 0
	})
	l.localIdentitiesByName[name] = list

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
	identities, ok := l.localIdentitiesByName[label]
	if ok {
		return identities[0].Identity
	}
	identity, ok := l.localIdentitiesByIdentity[label]
	if ok {
		return identity
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
			it.Close()
			return err
		}
		items = append(items, item)
	}
	it.Close()
	noDefault := len(l.getDefaultIdentifier()) == 0

	for _, entry := range items {
		id := entry.ID
		if l.getLocalIdentity(id) != nil {
			l.logger.Debugf("from storage: id [%s] already exists", id)
			continue
		}
		l.logger.Debugf("from storage: id [%s] does no exist, register it", id)
		conf := &driver.IdentityConfiguration{
			ID:     entry.ID,
			URL:    entry.URL,
			Config: entry.Config,
			Raw:    entry.Raw,
		}
		if err := l.registerIdentityConfiguration(conf, noDefault); err != nil {
			// failing to load an identity should not break the flow.
			l.logger.Errorf("failed registering identity from %s: %s", conf.URL, err.Error())
		}
	}
	return nil
}
