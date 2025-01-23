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
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
)

type KeyManagerProvider interface {
	Get(identityConfig *driver.IdentityConfiguration) (KeyManager, error)
}

type KeyManager interface {
	idriver.Deserializer
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
	config                 idriver.Config
	defaultNetworkIdentity driver.Identity
	signerService          idriver.SigService
	deserializerManager    idriver.DeserializerManager
	identityDB             driver3.IdentityDB
	binderService          idriver.BinderService
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
	config idriver.Config,
	defaultNetworkIdentity driver.Identity,
	signerService idriver.SigService,
	deserializerManager idriver.DeserializerManager,
	identityDB driver3.IdentityDB,
	binderService idriver.BinderService,
	identityType string,
	defaultAnonymous bool,
	keyManagerProviders ...KeyManagerProvider,
) *LocalMembership {
	return &LocalMembership{
		logger:                    logger.Named(identityType),
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

func (l *LocalMembership) GetIdentityInfo(label string, auditInfo []byte) (idriver.IdentityInfo, error) {
	l.localIdentitiesMutex.RLock()
	defer l.localIdentitiesMutex.RUnlock()

	if l.logger.IsEnabledFor(zapcore.DebugLevel) {
		l.logger.Debugf("get identity info by label [%s][%s]", label, hash.Hashable(label))
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

func (l *LocalMembership) Load(identities []*idriver.ConfiguredIdentity, targets []view.Identity) error {
	l.localIdentitiesMutex.Lock()
	defer l.localIdentitiesMutex.Unlock()

	l.logger.Debugf("load identities [%s][%+q]", l.IdentityType, identities)

	// init fields
	l.targetIdentities = targets
	l.localIdentities = make([]*LocalIdentity, 0)
	l.localIdentitiesByName = make(map[string][]LocalIdentityWithPriority, 0)

	// prepare all identity configurations
	identityConfigurations, defaults, err := l.toIdentityConfiguration(identities)
	if err != nil {
		return errors.Wrap(err, "failed to prepare identity configurations")
	}
	storedIdentityConfigurations, err := l.storedIdentityConfigurations()
	if err != nil {
		return errors.Wrap(err, "failed to load stored identity configurations")
	}

	// merge identityConfigurations and storedIdentityConfigurations
	// filter out stored configuration that are already in identityConfigurations
	var filtered []driver.IdentityConfiguration
	if len(storedIdentityConfigurations) != 0 {
		for _, stored := range storedIdentityConfigurations {
			found := false
			// if stored is in identityConfigurations, skip it
			for _, identityConfiguration := range identityConfigurations {
				if stored.ID == identityConfiguration.ID && stored.URL == identityConfiguration.URL {
					// we don't need this configuration
					found = true
				}
			}
			if !found {
				// keep this
				filtered = append(filtered, stored)
				defaults = append(defaults, false)
			}
		}
	}

	ics := append(identityConfigurations, filtered...)

	// load identities from configuration
	for i, identityConfiguration := range ics {
		l.logger.Debugf("load identity configuration [%+v]", identityConfiguration)
		if err := l.registerIdentityConfiguration(&identityConfiguration, defaults[i]); err != nil {
			// we log the error so the user can fix it but it shouldn't stop the loading of the service.
			l.logger.Errorf("failed loading identity with err [%s]", err)
		} else {
			l.logger.Debugf("load wallet for identity [%+v] done.", identityConfiguration)
		}
	}

	// if no default identity, use the first one
	defaultIdentifier := l.getDefaultIdentifier()
	if len(defaultIdentifier) == 0 {
		l.logger.Warnf("no default identity, use the first one available")
		if len(l.localIdentities) > 0 {
			defaultIdentity := l.firstDefaultIdentifier()
			if defaultIdentity == nil {
				l.logger.Warnf("no default identity can be set among the available identities [%d]", len(l.localIdentities))
			} else {
				defaultIdentity.Default = true
			}
			l.logger.Warnf("default identity is [%s]", l.getDefaultIdentifier())

		} else {
			l.logger.Warnf("cannot set default identity, no identity available")
		}
	} else {
		l.logger.Debugf("default identifier is [%s]", defaultIdentifier)
	}

	l.logger.Debugf("load identities [%s] done", l.IdentityType)

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

func (l *LocalMembership) firstDefaultIdentifier() *LocalIdentity {
	for _, identity := range l.localIdentities {
		if l.DefaultAnonymous && !identity.Anonymous {
			continue
		}
		return identity
	}
	return nil
}

func (l *LocalMembership) toIdentityConfiguration(identities []*config.Identity) ([]driver.IdentityConfiguration, []bool, error) {
	ics := make([]driver.IdentityConfiguration, len(identities))
	defaults := make([]bool, len(identities))
	for i, identity := range identities {
		optsRaw, err := yaml.Marshal(identity.Opts)
		if err != nil {
			return nil, nil, errors.WithMessage(err, "failed to marshal identity options")
		}
		ics[i] = driver.IdentityConfiguration{
			ID:     identity.ID,
			URL:    identity.Path,
			Type:   l.IdentityType,
			Config: optsRaw,
			Raw:    nil,
		}
		defaults[i] = identity.Default
	}
	return ics, defaults, nil
}

func (l *LocalMembership) registerLocalIdentity(identityConfig *driver.IdentityConfiguration, defaultIdentity bool) error {
	var errs []error
	var keyManager KeyManager
	var priority int
	l.logger.Debugf("try to load identity with [%d] key managers [%v]", len(l.KeyManagerProviders), l.KeyManagerProviders)
	for i, p := range l.KeyManagerProviders {
		var err error
		keyManager, err = p.Get(identityConfig)
		if err == nil && keyManager != nil && len(keyManager.EnrollmentID()) != 0 {
			priority = i
			break
		}
		keyManager = nil
		errs = append(errs, err)
	}
	if keyManager == nil {
		return errors2.Wrapf(
			errors3.Join(errs...),
			"failed to get a key manager for the passed identity config for [%s:%s]",
			identityConfig.ID,
			identityConfig.URL,
		)
	}

	l.logger.Debugf("append local identity for [%s]", identityConfig.ID)
	if err := l.addLocalIdentity(identityConfig, keyManager, defaultIdentity, priority); err != nil {
		return errors.Wrapf(err, "failed to add local identity for [%s]", identityConfig.ID)
	}

	if exists, _ := l.identityDB.ConfigurationExists(identityConfig.ID, l.IdentityType, identityConfig.URL); !exists {
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
	var errs []error
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
			errs = append(errs, err)
			l.logger.Errorf("failed registering local identity [%s]: [%s]", id, err)
			continue
		}
		found++
	}
	if found == 0 {
		return errors.Errorf("no valid identities found in [%s], errs [%v]", configuration.URL, errs)
	}
	return nil
}

func (l *LocalMembership) addLocalIdentity(config *driver.IdentityConfiguration, keyManager KeyManager, defaultID bool, priority int) error {
	// check for duplicates
	name := config.ID
	if keyManager.Anonymous() || len(l.targetIdentities) == 0 {
		l.logger.Debugf("no target identity check needed, skip it")
	} else if identity, _, err := keyManager.Identity(nil); err != nil {
		return err
	} else if found := slices.ContainsFunc(l.targetIdentities, identity.Equal); !found {
		l.logger.Debugf("identity [%s:%s] not in target identities, ignore it", name, config.URL)
		return nil
	} else {
		l.logger.Debugf("identity [%s:%s][%s] in target identities", name, config.URL, identity)
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
	l.logger.Debugf("new local identity for [%s:%s] - [%v]", name, eID, localIdentity)

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

	l.logger.Debugf("new local identity for [%s:%s] - [%d][%v]", name, eID, len(list), list)

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
		if l.logger.IsEnabledFor(zapcore.DebugLevel) {
			l.logger.Debugf("get local identity by name found with label [%s]", hash.Hashable(label))
		}
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

func (l *LocalMembership) storedIdentityConfigurations() ([]driver3.IdentityConfiguration, error) {
	it, err := l.identityDB.IteratorConfigurations(l.IdentityType)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get registered identities from kvs")
	}
	defer it.Close()
	// copy the iterator
	items := make([]driver3.IdentityConfiguration, 0)
	for it.HasNext() {
		item, err := it.Next()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
