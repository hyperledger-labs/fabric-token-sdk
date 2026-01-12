/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package membership

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	tdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"gopkg.in/yaml.v2"
)

const (
	// MaxPriority is used to set a very high priority for identities that match
	// target identities. Smaller numeric values mean higher priority.
	MaxPriority = -1 // smaller numbers, higher priority
)

var logger = logging.MustGetLogger()

// IdentityConfiguration is an alias to the driver-level identity configuration
// structure. LocalMembership expects identity configuration data in this shape.
type IdentityConfiguration = tdriver.IdentityConfiguration

// Config models the part of idriver.Config that LocalMembership needs.
// It is used to translate configured filesystem paths into runtime paths.
//
//go:generate counterfeiter -o mock/config.go -fake-name Config . Config
type Config interface {
	// TranslatePath converts a configured path (may contain ~ or env vars)
	// into an absolute path usable by the runtime.
	TranslatePath(path string) string
	IdentitiesForRole(role idriver.IdentityRoleType) ([]idriver.ConfiguredIdentity, error)
}

// SignerDeserializerManager models the part of idriver.SignerDeserializerManager
// that LocalMembership interacts with. LocalMembership registers typed
// signer-deserializers for key managers so that signatures can be deserialized
// later on when processing tokens.
//
//go:generate counterfeiter -o mock/sdm.go -fake-name SignerDeserializerManager . SignerDeserializerManager
type SignerDeserializerManager interface {
	AddTypedSignerDeserializer(typ idriver.IdentityType, d idriver.TypedSignerDeserializer)
}

//go:generate counterfeiter -o mock/ici.go -fake-name IdentityConfigurationIterator . IdentityConfigurationIterator
type IdentityConfigurationIterator = idriver.IdentityConfigurationIterator

// IdentityStoreService models the part of idriver.IdentityStoreService that
// LocalMembership needs. It provides a persistent place to record which
// identity configurations have been registered so they can be reloaded later.
//
//go:generate counterfeiter -o mock/iss.go -fake-name IdentityStoreService . IdentityStoreService
type IdentityStoreService interface {
	// AddConfiguration stores an identity configuration and the path to the
	// credentials relevant to this identity. The context may carry caller info.
	AddConfiguration(ctx context.Context, wp idriver.IdentityConfiguration) error
	// ConfigurationExists returns true if a configuration with the given id,
	// type and URL already exists in the store.
	ConfigurationExists(ctx context.Context, id, typ, url string) (bool, error)
	// IteratorConfigurations returns an iterator over all configurations of
	// a given type stored in the persistent store.
	IteratorConfigurations(ctx context.Context, configurationType string) (IdentityConfigurationIterator, error)
}

// IdentityProvider is an alias for the driver-level identity provider used to
// register identity descriptors, bind identities and resolve whether an
// identity belongs to this node (IsMe).
//
//go:generate counterfeiter -o mock/ip.go -fake-name IdentityProvider . IdentityProvider
type IdentityProvider = idriver.IdentityProvider

// KeyManagerProvider is responsible for producing a KeyManager for a given
// IdentityConfiguration. Multiple providers can be registered; the first one
// that succeeds is used for that identity.
//
//go:generate counterfeiter -o mock/kmp.go -fake-name KeyManagerProvider . KeyManagerProvider
type KeyManagerProvider interface {
	Get(ctx context.Context, identityConfig *IdentityConfiguration) (KeyManager, error)
}

// KeyManager encapsulates operations over a key material source (local or
// remote). LocalMembership uses KeyManager to deserialize signers, obtain an
// enrollment ID, check whether the key manager is remote/anonymous, and to
// fetch the identity descriptor (Identity + AuditInfo) used for binding and
// registration.
//
//go:generate counterfeiter -o mock/km.go -fake-name KeyManager . KeyManager
type KeyManager interface {
	DeserializeVerifier(ctx context.Context, raw []byte) (tdriver.Verifier, error)
	DeserializeSigner(ctx context.Context, raw []byte) (tdriver.Signer, error)
	EnrollmentID() string
	IsRemote() bool
	Anonymous() bool
	IdentityType() identity.Type
	Identity(ctx context.Context, auditInfo []byte) (*idriver.IdentityDescriptor, error)
}

// LocalIdentityWithPriority pairs a loaded LocalIdentity with a priority
// value. Priorities are used when multiple identities share the same name to
// select which identity should be preferred.
type LocalIdentityWithPriority struct {
	Identity *LocalIdentity
	Priority int
}

// PriorityComparison compares two LocalIdentityWithPriority values. It gives
// precedence to smaller integer values (i.e. lower numeric value == higher
// priority).
var PriorityComparison = func(a, b LocalIdentityWithPriority) int {
	if a.Priority < b.Priority {
		return -1
	} else if a.Priority > b.Priority {
		return 1
	}
	return 0
}

// LocalMembership manages the set of long-term identities that this process
// can act as (or on behalf of). It supports loading identities from
// configuration files and from a persistent identity store, registering new
// identities, and looking up identity information used by the token
// processing stack.
//
// Concurrency: read/write access to the in-memory indices is guarded by
// `localIdentitiesMutex`.
//
// The main responsibilities are:
// - Load identities from configuration and persistent store
// - Register an identity configuration and persist it to the store
// - Provide IdentityInfo wrappers that fetch token.Identity instances on-demand
// - Maintain mappings by identity name and by concrete identity string
// - Register typed signer deserializers from the KeyManager with the global manager
type LocalMembership struct {
	logger                 logging.Logger
	config                 Config
	defaultNetworkIdentity token.Identity
	deserializerManager    SignerDeserializerManager
	identityDB             IdentityStoreService
	KeyManagerProviders    []KeyManagerProvider
	IdentityType           string
	IdentityProvider       IdentityProvider

	localIdentitiesMutex      sync.RWMutex
	localIdentities           []*LocalIdentity
	localIdentitiesByName     map[string][]LocalIdentityWithPriority
	localIdentitiesByIdentity map[string]*LocalIdentity
	targetIdentities          []view.Identity // optional list of identities to prefer
	anonymous                 bool            // when true, only anonymous identities are considered selectable by default
}

// NewLocalMembership creates a new LocalMembership instance.
// Parameters:
// - logger: logger scoped to the identity type
// - config: configuration provider used to translate paths
// - defaultNetworkIdentity: the root network identity to bind other identities to
// - deserializerManager: manager where typed signer deserializers are registered
// - identityDB: persistent store for identity configurations
// - identityType: the identity type string used to wrap loaded identities
// - defaultAnonymous: whether identities should be loaded as anonymous by default
// - identityProvider: provider used to register and bind identities
// - keyManagerProviders: list of key manager providers to try when loading an identity
func NewLocalMembership(
	logger logging.Logger,
	config Config,
	defaultNetworkIdentity token.Identity,
	deserializerManager SignerDeserializerManager,
	identityDB IdentityStoreService,
	identityType string,
	defaultAnonymous bool,
	identityProvider IdentityProvider,
	keyManagerProviders ...KeyManagerProvider,
) *LocalMembership {
	return &LocalMembership{
		logger:                    logger.Named(identityType),
		config:                    config,
		defaultNetworkIdentity:    defaultNetworkIdentity,
		deserializerManager:       deserializerManager,
		identityDB:                identityDB,
		localIdentitiesByName:     map[string][]LocalIdentityWithPriority{},
		localIdentitiesByIdentity: map[string]*LocalIdentity{},
		IdentityType:              identityType,
		KeyManagerProviders:       keyManagerProviders,
		anonymous:                 defaultAnonymous,
		IdentityProvider:          identityProvider,
	}
}

// DefaultNetworkIdentity returns the root network identity used when binding loaded identities.
func (l *LocalMembership) DefaultNetworkIdentity() token.Identity {
	return l.defaultNetworkIdentity
}

// IsMe reports whether the given identity belongs to this local membership set.
// It delegates to the configured IdentityProvider to determine membership.
func (l *LocalMembership) IsMe(ctx context.Context, id token.Identity) bool {
	return l.IdentityProvider.IsMe(ctx, id)
}

// GetIdentifier returns the configured identifier (label) for the provided token.Identity.
// The method tries both the raw bytes and the string representation of the identity
// when looking up the in-memory mapping.
func (l *LocalMembership) GetIdentifier(ctx context.Context, id token.Identity) (string, error) {
	l.localIdentitiesMutex.RLock()
	defer l.localIdentitiesMutex.RUnlock()

	for _, label := range []string{string(id), id.String()} {
		l.logger.DebugfContext(ctx, "get local identity by label [%s]", utils.Hashable(label))
		r := l.getLocalIdentity(ctx, label)
		if r == nil {
			l.logger.DebugfContext(ctx,
				"local identity not found for label [%s] [%v]",
				logging.Keys(l.localIdentitiesByName),
				logging.Printable(label),
			)
			continue
		}
		return r.Name, nil
	}
	return "", errors.Errorf("identifier not found for id [%s]", id)
}

// GetDefaultIdentifier returns the name of the default identity currently loaded.
// It honors the LocalMembership anonymous flag and only returns an identity
// selectable under the current anonymity mode.
func (l *LocalMembership) GetDefaultIdentifier() string {
	l.localIdentitiesMutex.RLock()
	defer l.localIdentitiesMutex.RUnlock()

	return l.getDefaultIdentifier()
}

// GetIdentityInfo looks up identity information for a given label and produces an IdentityInfo
// that can be used to fetch a token.Identity on demand. The auditInfo bytes are passed to the
// underlying key manager when requesting the identity. The returned IdentityInfo will lazily
// fetch or compute the actual token.Identity when needed.
func (l *LocalMembership) GetIdentityInfo(ctx context.Context, label string, auditInfo []byte) (idriver.IdentityInfo, error) {
	l.localIdentitiesMutex.RLock()
	defer l.localIdentitiesMutex.RUnlock()

	l.logger.DebugfContext(ctx, "get identity info by label [%s][%s]", logging.Printable(label), utils.Hashable(label))
	localIdentity := l.getLocalIdentity(ctx, label)
	if localIdentity == nil {
		return nil, errors.Errorf("local identity not found for label [%s][%v]", utils.Hashable(label), l.localIdentitiesByName)
	}
	return NewIdentityInfo(localIdentity, func(ctx context.Context) (token.Identity, []byte, error) {
		return localIdentity.GetIdentity(ctx, auditInfo)
	}), nil
}

// RegisterIdentity registers a new identity configuration into the LocalMembership and
// persists it into the identity store if it is successfully added. The function
// acquires a write lock while modifying internal maps/lists.
func (l *LocalMembership) RegisterIdentity(ctx context.Context, idConfig IdentityConfiguration) error {
	l.localIdentitiesMutex.Lock()
	defer l.localIdentitiesMutex.Unlock()

	return l.registerIdentityConfiguration(ctx, &idConfig, l.getDefaultIdentifier() == "")
}

// IDs returns the list of identity names currently loaded in the LocalMembership.
func (l *LocalMembership) IDs() ([]string, error) {
	l.localIdentitiesMutex.RLock()
	defer l.localIdentitiesMutex.RUnlock()

	set := collections.NewSet[string]()
	for _, li := range l.localIdentities {
		set.Add(li.Name)
	}
	return set.ToSlice(), nil
}

// Load initializes LocalMembership from a list of configured identities and optional target
// identities (to give higher priority to the matching ones). It also loads any identities found
// in the persistent identity store. The function will log errors for identities that fail to
// register but will try to continue loading the remaining entries.
func (l *LocalMembership) Load(ctx context.Context, identities []idriver.ConfiguredIdentity, targets []view.Identity) error {
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
	storedIdentityConfigurations, err := l.storedIdentityConfigurations(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to load stored identity configurations")
	}

	// merge identityConfigurations and storedIdentityConfigurations
	// filter out stored configuration that are already in identityConfigurations
	var filtered []IdentityConfiguration
	if len(storedIdentityConfigurations) != 0 {
		for _, stored := range storedIdentityConfigurations {
			found := false
			// if stored is in identityConfigurations, skip it
			for _, ic := range identityConfigurations {
				if stored.ID == ic.ID && stored.URL == ic.URL {
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
		if err := l.registerIdentityConfiguration(ctx, &identityConfiguration, defaults[i]); err != nil {
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

// getDefaultIdentifier returns the name of the current default identity (may return empty string).
func (l *LocalMembership) getDefaultIdentifier() string {
	for _, li := range l.localIdentities {
		// if we are in anonymous mode skip non-anonymous identities
		if l.anonymous && !li.Anonymous {
			continue
		}

		if li.Default {
			return li.Name
		}
	}
	return ""
}

// firstDefaultIdentifier returns the first identity that can be used as default under the current
// anonymity setting (or nil if none exists).
func (l *LocalMembership) firstDefaultIdentifier() *LocalIdentity {
	for _, li := range l.localIdentities {
		if l.anonymous && !li.Anonymous {
			continue
		}
		return li
	}
	return nil
}

func (l *LocalMembership) toIdentityConfiguration(identities []idriver.ConfiguredIdentity) ([]IdentityConfiguration, []bool, error) {
	ics := make([]IdentityConfiguration, len(identities))
	defaults := make([]bool, len(identities))

	for i, ci := range identities {
		optsRaw, err := marshalOpts(ci.Opts)
		if err != nil {
			return nil, nil, errors.WithMessagef(err, "failed to marshal identity options")
		}

		ics[i] = IdentityConfiguration{
			ID:     ci.ID,
			URL:    ci.Path,
			Type:   l.IdentityType,
			Config: optsRaw,
			Raw:    nil,
		}
		defaults[i] = ci.Default
	}
	return ics, defaults, nil
}

func (l *LocalMembership) registerLocalIdentity(ctx context.Context, identityConfig *IdentityConfiguration, defaultIdentity bool) error {
	var errs []error
	var keyManager KeyManager
	var priority int
	l.logger.DebugfContext(ctx, "try to load identity with [%d] key managers [%v]", len(l.KeyManagerProviders), l.KeyManagerProviders)
	for i, p := range l.KeyManagerProviders {
		var err error
		keyManager, err = p.Get(ctx, identityConfig)
		if err == nil && keyManager != nil && len(keyManager.EnrollmentID()) != 0 {
			priority = i
			break
		}
		keyManager = nil
		errs = append(errs, err)
	}
	if keyManager == nil {
		return errors.Wrapf(
			errors.Join(errs...),
			"failed to get a key manager for the passed identity config for [%s:%s]",
			identityConfig.ID,
			identityConfig.URL,
		)
	}

	l.logger.DebugfContext(ctx, "append local identity for [%s]", identityConfig.ID)
	if err := l.addLocalIdentity(ctx, identityConfig, keyManager, defaultIdentity, priority); err != nil {
		return errors.Wrapf(err, "failed to add local identity for [%s]", identityConfig.ID)
	}

	if exists, _ := l.identityDB.ConfigurationExists(ctx, identityConfig.ID, l.IdentityType, identityConfig.URL); !exists {
		l.logger.DebugfContext(ctx, "does the configuration already exists for [%s]? no, add it", identityConfig.ID)
		// enforce type
		identityConfig.Type = l.IdentityType
		if err := l.identityDB.AddConfiguration(ctx, *identityConfig); err != nil {
			return err
		}
	}
	l.logger.DebugfContext(ctx, "added local identity for id [%s], remote [%v]", identityConfig.ID+"@"+keyManager.EnrollmentID(), keyManager.IsRemote())
	return nil
}

func (l *LocalMembership) registerIdentityConfiguration(ctx context.Context, identity *IdentityConfiguration, defaultIdentity bool) error {
	// Try to register the local identity
	identity.URL = l.config.TranslatePath(identity.URL)
	if err := l.registerLocalIdentity(ctx, identity, defaultIdentity); err != nil {
		l.logger.Warnf("failed to load local identity at [%s]:[%s]", identity.URL, err)
		// Does path correspond to a folder containing multiple identities?
		if err := l.registerLocalIdentities(ctx, identity); err != nil {
			return errors.WithMessagef(err, "failed to register local identity")
		}
	}
	return nil
}

func (l *LocalMembership) registerLocalIdentities(ctx context.Context, configuration *IdentityConfiguration) error {
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
		if err := l.registerLocalIdentity(ctx, &IdentityConfiguration{
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

func (l *LocalMembership) addLocalIdentity(ctx context.Context, config *IdentityConfiguration, keyManager KeyManager, defaultID bool, priority int) error {
	var getIdentity GetIdentityFunc
	var resolvedIdentity token.Identity

	typedIdentityInfo := &TypedIdentityInfo{
		GetIdentity:      keyManager.Identity,
		IdentityType:     keyManager.IdentityType(),
		EnrollmentID:     keyManager.EnrollmentID(),
		RootIdentity:     l.defaultNetworkIdentity,
		IdentityProvider: l.IdentityProvider,
	}
	if keyManager.Anonymous() {
		// For anonymous key managers we keep the provider function so the identity
		// can be obtained later with arbitrary audit info.
		getIdentity = typedIdentityInfo.Get
	} else {
		// For non-anonymous key managers we eagerly fetch the identity and audit
		// info now and cache it to avoid repeated remote calls.
		var auditInfo []byte
		var err error
		resolvedIdentity, auditInfo, err = typedIdentityInfo.Get(ctx, nil)
		if err != nil {
			return errors.WithMessagef(err, "failed to get identity")
		}
		getIdentity = func(context.Context, []byte) (token.Identity, []byte, error) {
			return resolvedIdentity, auditInfo, nil
		}
	}

	// check for duplicates
	name := config.ID
	if keyManager.Anonymous() || len(l.targetIdentities) == 0 {
		l.logger.Debugf("no target identity check needed, skip it")
	} else if found := slices.ContainsFunc(l.targetIdentities, resolvedIdentity.Equal); !found {
		// the identity is not in the target identities, we should give it a lower priority
		l.logger.Debugf("identity [%s:%s] not in target identities", name, config.URL)
	} else {
		// give it high priority
		priority = MaxPriority
		l.logger.Debugf("identity [%s:%s][%s] in target identities", name, config.URL, resolvedIdentity)
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
	l.deserializerManager.AddTypedSignerDeserializer(keyManager.IdentityType(), &TypedSignerDeserializer{KeyManager: keyManager})

	// if the keyManager is not anonymous
	if !keyManager.Anonymous() {
		l.logger.Debugf("adding identity mapping for [%s]", resolvedIdentity)
		l.localIdentitiesByIdentity[resolvedIdentity.String()] = localIdentity
		if err := l.IdentityProvider.Bind(ctx, l.defaultNetworkIdentity, resolvedIdentity); err != nil {
			return errors.WithMessagef(err, "cannot bind identity for [%s,%s]", resolvedIdentity, eID)
		}
	}

	l.localIdentities = append(l.localIdentities, localIdentity)
	return nil
}

func (l *LocalMembership) getLocalIdentity(ctx context.Context, label string) *LocalIdentity {
	l.logger.DebugfContext(ctx, "get local identity by label [%s]", utils.Hashable(label))
	identities, ok := l.localIdentitiesByName[label]
	if ok {
		l.logger.DebugfContext(ctx, "get local identity by name found with label [%s]", utils.Hashable(label))
		return identities[0].Identity
	}
	mapped, ok := l.localIdentitiesByIdentity[label]
	if ok {
		return mapped
	}

	l.logger.DebugfContext(ctx, "local identity not found for label [%s][%v]", utils.Hashable(label), l.localIdentitiesByName)
	return nil
}

func (l *LocalMembership) storedIdentityConfigurations(ctx context.Context) ([]idriver.IdentityConfiguration, error) {
	it, err := l.identityDB.IteratorConfigurations(ctx, l.IdentityType)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get registered identities from kvs")
	}
	return collections.ReadAll[idriver.IdentityConfiguration](it)
}

// TypedIdentityInfo is a helper that knows how to materialize a typed identity
// (optionally wrapping the underlying identity with an identity type) and
// register/bind the identity descriptor with the identity provider.
//
// The Get method returns the token.Identity to use and any audit info bytes.
type TypedIdentityInfo struct {
	// GetIdentity fetches the identity descriptor (identity + audit info) from
	// the KeyManager. It accepts auditInfo bytes that may be used by remote
	// key managers to produce a specific identity variant.
	GetIdentity  func(context.Context, []byte) (*idriver.IdentityDescriptor, error)
	IdentityType identity.Type

	EnrollmentID     string
	RootIdentity     token.Identity
	IdentityProvider idriver.IdentityProvider
}

func (i *TypedIdentityInfo) Get(ctx context.Context, auditInfo []byte) (token.Identity, []byte, error) {
	// get the identity
	logger.DebugfContext(ctx, "fetch identity")

	identityDescriptor, err := i.GetIdentity(ctx, auditInfo)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get root identity for [%s]", i.EnrollmentID)
	}
	id := identityDescriptor.Identity
	ai := identityDescriptor.AuditInfo

	typedIdentity := id
	if len(i.IdentityType) != 0 {
		logger.DebugfContext(ctx, "wrap and bind as [%s]", i.IdentityType)
		typedIdentity, err = identity.WrapWithType(i.IdentityType, id)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to wrap identity [%s]", i.IdentityType)
		}
	}

	// register the audit info
	logger.DebugfContext(ctx, "register identity descriptor")
	if err := i.IdentityProvider.RegisterIdentityDescriptor(ctx, identityDescriptor, typedIdentity); err != nil {
		return nil, nil, errors.Wrapf(err, "failed to register identity descriptor for [%s][%s]", id, typedIdentity)
	}

	logger.DebugfContext(ctx, "bind to root identity")
	if err := i.IdentityProvider.Bind(ctx, i.RootIdentity, id, typedIdentity); err != nil {
		return nil, nil, errors.Wrapf(err, "failed to bind identity [%s] to [%s]", id, i.RootIdentity)
	}
	return typedIdentity, ai, nil
}

// TypedSignerDeserializer adapts a KeyManager so it can be used where the
// driver expects an idriver.TypedSignerDeserializer. It forwards DeserializeSigner
// calls to the underlying KeyManager implementation.
type TypedSignerDeserializer struct {
	KeyManager
}

func (t *TypedSignerDeserializer) DeserializeSigner(ctx context.Context, _ identity.Type, raw []byte) (tdriver.Signer, error) {
	return t.KeyManager.DeserializeSigner(ctx, raw)
}

func marshalOpts(opts interface{}) (optsRaw []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("panic caught while marshalling identity options: %v", r)
		}
	}()
	optsRaw, err = yaml.Marshal(opts)
	return
}
