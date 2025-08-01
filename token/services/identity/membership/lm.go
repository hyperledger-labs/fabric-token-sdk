/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package membership

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"

	errors2 "github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"gopkg.in/yaml.v2"
)

const (
	MaxPriority = -1 // smaller numbers, higher priority
)

var logger = logging.MustGetLogger()

type KeyManagerProvider interface {
	Get(identityConfig *driver.IdentityConfiguration) (KeyManager, error)
}

type KeyManager interface {
	idriver.Deserializer
	EnrollmentID() string
	IsRemote() bool
	Anonymous() bool
	IdentityType() identity.Type
	Identity(ctx context.Context, auditInfo []byte) (driver.Identity, []byte, error)
}

type LocalIdentityWithPriority struct {
	Identity *LocalIdentity
	Priority int
}

// PriorityComparison gives higher priority to smaller numbers
var PriorityComparison = func(a, b LocalIdentityWithPriority) int {
	if a.Priority < b.Priority {
		return -1
	} else if a.Priority > b.Priority {
		return 1
	}
	return 0
}

type LocalMembership struct {
	config                 idriver.Config
	defaultNetworkIdentity driver.Identity
	signerService          idriver.SigService
	deserializerManager    idriver.DeserializerManager
	identityDB             idriver.IdentityStoreService
	binderService          idriver.BinderService
	KeyManagerProviders    []KeyManagerProvider
	IdentityType           string
	IdentityProvider       idriver.IdentityProvider
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
	identityDB idriver.IdentityStoreService,
	binderService idriver.BinderService,
	identityType string,
	defaultAnonymous bool,
	identityProvider idriver.IdentityProvider,
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
		IdentityProvider:          identityProvider,
	}
}

func (l *LocalMembership) DefaultNetworkIdentity() driver.Identity {
	return l.defaultNetworkIdentity
}

func (l *LocalMembership) IsMe(ctx context.Context, id driver.Identity) bool {
	return l.signerService.IsMe(ctx, id)
}

func (l *LocalMembership) GetIdentifier(id driver.Identity) (string, error) {
	l.localIdentitiesMutex.RLock()
	defer l.localIdentitiesMutex.RUnlock()

	for _, label := range []string{string(id), id.String()} {
		l.logger.Debugf("get local identity by label [%s]", hash.Hashable(label))
		r := l.getLocalIdentity(label)
		if r == nil {
			l.logger.Debugf(
				"local identity not found for label [%s] [%v]",
				logging.Keys(l.localIdentitiesByName),
				logging.Printable(label),
			)
			continue
		}
		return r.Name, nil
	}
	return "", errors2.Errorf("identifier not found for id [%s]", id)
}

func (l *LocalMembership) GetDefaultIdentifier() string {
	l.localIdentitiesMutex.RLock()
	defer l.localIdentitiesMutex.RUnlock()

	return l.getDefaultIdentifier()
}

func (l *LocalMembership) GetIdentityInfo(label string, auditInfo []byte) (idriver.IdentityInfo, error) {
	l.localIdentitiesMutex.RLock()
	defer l.localIdentitiesMutex.RUnlock()

	l.logger.Debugf("get identity info by label [%s][%s]", label, hash.Hashable(label))
	localIdentity := l.getLocalIdentity(label)
	if localIdentity == nil {
		return nil, errors2.Errorf("local identity not found for label [%s][%v]", hash.Hashable(label), l.localIdentitiesByName)
	}
	return NewIdentityInfo(localIdentity, func(ctx context.Context) (driver.Identity, []byte, error) {
		return localIdentity.GetIdentity(ctx, auditInfo)
	}), nil
}

func (l *LocalMembership) RegisterIdentity(ctx context.Context, idConfig driver.IdentityConfiguration) error {
	l.localIdentitiesMutex.Lock()
	defer l.localIdentitiesMutex.Unlock()

	return l.registerIdentityConfiguration(ctx, &idConfig, l.getDefaultIdentifier() == "")
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
		return errors2.Wrap(err, "failed to prepare identity configurations")
	}
	storedIdentityConfigurations, err := l.storedIdentityConfigurations(context.Background())
	if err != nil {
		return errors2.Wrap(err, "failed to load stored identity configurations")
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
		l.logger.Infof("load identity configuration [%+v]", identityConfiguration)
		if err := l.registerIdentityConfiguration(context.Background(), &identityConfiguration, defaults[i]); err != nil {
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

func (l *LocalMembership) toIdentityConfiguration(identities []*idriver.ConfiguredIdentity) ([]driver.IdentityConfiguration, []bool, error) {
	ics := make([]driver.IdentityConfiguration, len(identities))
	defaults := make([]bool, len(identities))
	for i, identity := range identities {
		optsRaw, err := yaml.Marshal(identity.Opts)
		if err != nil {
			return nil, nil, errors2.WithMessagef(err, "failed to marshal identity options")
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

func (l *LocalMembership) registerLocalIdentity(ctx context.Context, identityConfig *driver.IdentityConfiguration, defaultIdentity bool) error {
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
			errors.Join(errs...),
			"failed to get a key manager for the passed identity config for [%s:%s]",
			identityConfig.ID,
			identityConfig.URL,
		)
	}

	l.logger.Debugf("append local identity for [%s]", identityConfig.ID)
	if err := l.addLocalIdentity(identityConfig, keyManager, defaultIdentity, priority); err != nil {
		return errors2.Wrapf(err, "failed to add local identity for [%s]", identityConfig.ID)
	}

	if exists, _ := l.identityDB.ConfigurationExists(ctx, identityConfig.ID, l.IdentityType, identityConfig.URL); !exists {
		l.logger.Debugf("does the configuration already exists for [%s]? no, add it", identityConfig.ID)
		// enforce type
		identityConfig.Type = l.IdentityType
		if err := l.identityDB.AddConfiguration(ctx, *identityConfig); err != nil {
			return err
		}
	}
	l.logger.Debugf("added local identity for id [%s], remote [%v]", identityConfig.ID+"@"+keyManager.EnrollmentID(), keyManager.IsRemote())
	return nil
}

func (l *LocalMembership) registerIdentityConfiguration(ctx context.Context, identity *driver.IdentityConfiguration, defaultIdentity bool) error {
	// Try to register the local identity
	identity.URL = l.config.TranslatePath(identity.URL)
	if err := l.registerLocalIdentity(ctx, identity, defaultIdentity); err != nil {
		l.logger.Warnf("failed to load local identity at [%s]:[%s]", identity.URL, err)
		// Does path correspond to a folder containing multiple identities?
		if err := l.registerLocalIdentities(ctx, identity); err != nil {
			return errors2.WithMessagef(err, "failed to register local identity")
		}
	}
	return nil
}

func (l *LocalMembership) registerLocalIdentities(ctx context.Context, configuration *driver.IdentityConfiguration) error {
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
		if err := l.registerLocalIdentity(ctx, &driver.IdentityConfiguration{
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
		return errors2.Errorf("no valid identities found in [%s], errs [%v]", configuration.URL, errs)
	}
	return nil
}

func (l *LocalMembership) addLocalIdentity(config *driver.IdentityConfiguration, keyManager KeyManager, defaultID bool, priority int) error {
	var getIdentity GetIdentityFunc
	var identity driver.Identity

	typedIdentityInfo := &TypedIdentityInfo{
		GetIdentity:      keyManager.Identity,
		IdentityType:     keyManager.IdentityType(),
		EnrollmentID:     keyManager.EnrollmentID(),
		RootIdentity:     l.defaultNetworkIdentity,
		IdentityProvider: l.IdentityProvider,
		BinderService:    l.binderService,
	}
	if keyManager.Anonymous() {
		getIdentity = typedIdentityInfo.Get
	} else {
		var auditInfo []byte
		var err error
		identity, auditInfo, err = typedIdentityInfo.Get(context.Background(), nil)
		if err != nil {
			return errors2.WithMessagef(err, "failed to get identity")
		}
		getIdentity = func(context.Context, []byte) (driver.Identity, []byte, error) {
			return identity, auditInfo, nil
		}
	}

	// check for duplicates
	name := config.ID
	if keyManager.Anonymous() || len(l.targetIdentities) == 0 {
		l.logger.Debugf("no target identity check needed, skip it")
	} else if found := slices.ContainsFunc(l.targetIdentities, identity.Equal); !found {
		// the identity is not in the target identities, we should give it a lower priority
		l.logger.Debugf("identity [%s:%s] not in target identities", name, config.URL)
	} else {
		// give it high priority
		priority = MaxPriority
		l.logger.Debugf("identity [%s:%s][%s] in target identities", name, config.URL, identity)
	}

	eID := keyManager.EnrollmentID()
	localIdentity := &LocalIdentity{
		Name:         name,
		Default:      defaultID,
		EnrollmentID: eID,
		Anonymous:    keyManager.Anonymous(),
		GetIdentity:  getIdentity,
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
	slices.SortFunc(list, PriorityComparison)
	l.localIdentitiesByName[name] = list

	l.logger.Debugf("new local identity for [%s:%s] - [%d][%v]", name, eID, len(list), list)

	// deserializer
	l.deserializerManager.AddDeserializer(keyManager)

	// if the keyManager is not anonymous
	if !keyManager.Anonymous() {
		l.logger.Debugf("adding identity mapping for [%s]", identity)
		l.localIdentitiesByIdentity[identity.String()] = localIdentity
		if l.binderService != nil {
			if err := l.binderService.Bind(context.Background(), l.defaultNetworkIdentity, identity, false); err != nil {
				return errors2.WithMessagef(err, "cannot bind identity for [%s,%s]", identity, eID)
			}
		}
	}

	l.localIdentities = append(l.localIdentities, localIdentity)
	return nil
}

func (l *LocalMembership) getLocalIdentity(label string) *LocalIdentity {
	l.logger.Debugf("get local identity by label [%s]", hash.Hashable(label))
	identities, ok := l.localIdentitiesByName[label]
	if ok {
		l.logger.Debugf("get local identity by name found with label [%s]", hash.Hashable(label))
		return identities[0].Identity
	}
	identity, ok := l.localIdentitiesByIdentity[label]
	if ok {
		return identity
	}

	l.logger.Debugf("local identity not found for label [%s][%v]", hash.Hashable(label), l.localIdentitiesByName)
	return nil
}

func (l *LocalMembership) storedIdentityConfigurations(ctx context.Context) ([]idriver.IdentityConfiguration, error) {
	it, err := l.identityDB.IteratorConfigurations(ctx, l.IdentityType)
	if err != nil {
		return nil, errors2.WithMessagef(err, "failed to get registered identities from kvs")
	}
	return collections.ReadAll[idriver.IdentityConfiguration](it)
}

type TypedIdentityInfo struct {
	GetIdentity  func(context.Context, []byte) (driver.Identity, []byte, error)
	IdentityType identity.Type

	EnrollmentID     string
	RootIdentity     driver.Identity
	IdentityProvider idriver.IdentityProvider
	BinderService    idriver.BinderService
}

func (i *TypedIdentityInfo) Get(ctx context.Context, auditInfo []byte) (driver.Identity, []byte, error) {
	// get the identity
	logger.DebugfContext(ctx, "fetch identity")

	id, ai, err := i.GetIdentity(ctx, auditInfo)
	if err != nil {
		return nil, nil, errors2.Wrapf(err, "failed to get root identity for [%s]", i.EnrollmentID)
	}
	// register the audit info
	logger.DebugfContext(ctx, "register audit info")
	if err := i.IdentityProvider.RegisterAuditInfo(ctx, id, ai); err != nil {
		return nil, nil, errors2.Wrapf(err, "failed to register audit info for identity [%s]", id)
	}
	// bind the identity to the default FSC node identity
	if i.BinderService != nil {
		logger.DebugfContext(ctx, "bind to root identity")
		if err := i.BinderService.Bind(ctx, i.RootIdentity, id, false); err != nil {
			return nil, nil, errors2.Wrapf(err, "failed to bind identity [%s] to [%s]", id, i.RootIdentity)
		}
	}
	// wrap the backend identity, and bind it
	if len(i.IdentityType) != 0 {
		logger.DebugfContext(ctx, "wrap and bind as [%s]", i.IdentityType)
		typedIdentity, err := identity.WrapWithType(i.IdentityType, id)
		if err != nil {
			return nil, nil, errors2.Wrapf(err, "failed to wrap identity [%s]", i.IdentityType)
		}
		if i.BinderService != nil {
			logger.DebugfContext(ctx, "bind wrapped")
			if err := i.BinderService.Bind(ctx, id, typedIdentity, true); err != nil {
				return nil, nil, errors2.Wrapf(err, "failed to bind identity [%s] to [%s]", typedIdentity, id)
			}
			if err := i.BinderService.Bind(ctx, i.RootIdentity, typedIdentity, false); err != nil {
				return nil, nil, errors2.Wrapf(err, "failed to bind identity [%s] to [%s]", typedIdentity, i.RootIdentity)
			}
		} else {
			// register at the list the audit info
			logger.DebugfContext(ctx, "register audit infor for wrapped identity")
			if err := i.IdentityProvider.RegisterAuditInfo(ctx, typedIdentity, ai); err != nil {
				return nil, nil, errors2.Wrapf(err, "failed to register audit info for identity [%s]", id)
			}
		}
		id = typedIdentity
	}
	logger.DebugfContext(ctx, "fetch identity done")
	return id, ai, nil
}
