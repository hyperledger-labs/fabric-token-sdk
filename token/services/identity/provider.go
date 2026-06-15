/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/cache/secondcache"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"go.uber.org/zap/zapcore"
)

var (
	// This makes sure that Provider implements driver.IdentityProvider
	_ driver.IdentityProvider = &Provider{}
	// This makes sure that Provider can roll back partial recipient registration
	_ RecipientRegistrationRollback = &Provider{}
)

// StorageProvider returns storage services scoped to a specific token
// management system (TMS) identified by token.TMSID.
// Callers request the concrete store service for the given TMS and use the returned service to
// access persisted wallet, identity, or keystore data.
type StorageProvider = idriver.StorageProvider

// EnrollmentIDUnmarshaler decodes an enrollment ID form an audit info
//
//go:generate counterfeiter -o mock/eidu.go -fake-name EnrollmentIDUnmarshaler . EnrollmentIDUnmarshaler
type EnrollmentIDUnmarshaler interface {
	// GetEnrollmentID returns the enrollment ID from the audit info
	GetEnrollmentID(ctx context.Context, identity driver.Identity, auditInfo []byte) (string, error)
	// GetRevocationHandler returns the revocation handle from the audit info
	GetRevocationHandler(ctx context.Context, identity driver.Identity, auditInfo []byte) (string, error)
	// GetEIDAndRH returns both enrollment ID and revocation handle
	GetEIDAndRH(ctx context.Context, identity driver.Identity, auditInfo []byte) (string, string, error)
}

//go:generate counterfeiter -o mock/storage.go -fake-name Storage . Storage
type Storage interface {
	GetAuditInfo(ctx context.Context, id []byte) ([]byte, error)
	StoreIdentityData(ctx context.Context, id []byte, identityAudit []byte, tokenMetadata []byte, tokenMetadataAudit []byte) error
	StoreSignerInfo(ctx context.Context, id driver.Identity, info []byte) error
	GetExistingSignerInfo(ctx context.Context, ids ...driver.Identity) ([]string, error)
	SignerInfoExists(ctx context.Context, id []byte) (bool, error)
	GetSignerInfo(ctx context.Context, identity []byte) ([]byte, error)
	RegisterIdentityDescriptor(ctx context.Context, descriptor *idriver.IdentityDescriptor, alias driver.Identity) error
}

type cache[T any] interface {
	Get(key string) (T, bool)
	Add(key string, value T)
	Delete(key string)
}

//go:generate counterfeiter -o mock/deserializer.go -fake-name Deserializer . Deserializer
type Deserializer interface {
	DeserializeSigner(ctx context.Context, raw []byte) (driver.Signer, error)
}

//go:generate counterfeiter -o mock/nbs.go -fake-name NetworkBinderService . NetworkBinderService
type NetworkBinderService interface {
	Bind(ctx context.Context, longTerm driver.Identity, ephemeral ...driver.Identity) error
}

type SignerEntry struct {
	Signer     driver.Signer
	DebugStack []byte
}

// Provider implements the driver.IdentityProvider interface.
// Provider manages identity-related concepts like signature signers, verifiers, audit information, and so on.
type Provider struct {
	Logger                  logging.Logger
	Binder                  NetworkBinderService
	enrollmentIDUnmarshaler EnrollmentIDUnmarshaler
	storage                 Storage
	deserializer            Deserializer

	signers cache[*SignerEntry]

	metrics        *IdentityMetrics
	circuitBreaker *CircuitBreaker
}

// Option is a functional option for the identity provider.
type Option func(*Provider)

// WithMetrics returns an option to configure the identity provider with the provided metrics provider.
func WithMetrics(p metrics.Provider) Option {
	return func(pr *Provider) {
		pr.metrics = NewIdentityMetrics(p)
	}
}

// WithCircuitBreaker returns an option to configure the identity provider with the provided circuit breaker configuration.
func WithCircuitBreaker(config CircuitBreakerConfig) Option {
	return func(pr *Provider) {
		pr.circuitBreaker = NewCircuitBreaker(config)
	}
}

// NewProvider returns a new instance of Provider
func NewProvider(
	logger logging.Logger,
	storage Storage,
	deserializer Deserializer,
	binder NetworkBinderService,
	enrollmentIDUnmarshaler EnrollmentIDUnmarshaler,
	opts ...Option,
) *Provider {
	p := &Provider{
		Logger:                  logger,
		Binder:                  binder,
		enrollmentIDUnmarshaler: enrollmentIDUnmarshaler,
		deserializer:            deserializer,
		storage:                 storage,
		signers:                 secondcache.NewTyped[*SignerEntry](50),
	}
	for _, opt := range opts {
		opt(p)
	}

	return p
}

// RegisterRecipientData stores the passed recipient data in the configured storage.
func (p *Provider) RegisterRecipientData(ctx context.Context, data *driver.RecipientData) (err error) {
	if p.metrics != nil {
		p.metrics.Requests.Add(1)
		p.metrics.InFlight.Add(1)
		defer p.metrics.InFlight.Add(-1)
	}
	start := time.Now()
	defer func() { p.recordMetrics(start, err) }()

	if p.circuitBreaker != nil && !p.circuitBreaker.Allow() {
		return errors.New("back-pressure: service temporarily overloaded, retry later")
	}

	err = p.storage.StoreIdentityData(ctx, data.Identity, data.AuditInfo, data.TokenMetadata, data.TokenMetadataAuditInfo)

	return err
}

// RegisterSigner registers a Signer and a Verifier for passed identity.
// This is implemented via an invocation of  RegisterIdentityDescriptor using an IdentityDescriptor with empty AuditInfo.
// The audit info might or might not be already stored.
func (p *Provider) RegisterSigner(ctx context.Context, identity driver.Identity, signer driver.Signer, verifier driver.Verifier, signerInfo []byte, ephemeral bool) (err error) {
	if p.metrics != nil {
		p.metrics.Requests.Add(1)
		p.metrics.InFlight.Add(1)
		defer p.metrics.InFlight.Add(-1)
	}
	start := time.Now()
	defer func() { p.recordMetrics(start, err) }()

	if p.circuitBreaker != nil && !p.circuitBreaker.Allow() {
		return errors.New("back-pressure: service temporarily overloaded, retry later")
	}

	identityDescriptor := &idriver.IdentityDescriptor{
		Identity:   identity,
		AuditInfo:  nil,
		Signer:     signer,
		SignerInfo: signerInfo,
		Verifier:   verifier,
		Ephemeral:  ephemeral,
	}

	err = p.RegisterIdentityDescriptor(ctx, identityDescriptor, nil)

	return err
}

// AreMe returns the hashes of the passed identities that have a signer registered before.
// Each identity is resolved via the signer cache and configured storage.
// There is no secondary "is me" cache: a real cache would need careful handling for
// single-use identities (for example Idemix nyms) and is intentionally omitted here.
func (p *Provider) AreMe(ctx context.Context, identities ...driver.Identity) []string {
	if p.metrics != nil {
		p.metrics.Requests.Add(1)
		p.metrics.InFlight.Add(1)
		defer p.metrics.InFlight.Add(-1)
	}
	start := time.Now()
	defer func() { p.recordMetrics(start, nil) }()

	p.Logger.DebugfContext(ctx, "identity [%s] is me?", identities)

	return p.areMe(ctx, identities...)
}

// IsMe returns true if a signer was ever registered for the passed identity
func (p *Provider) IsMe(ctx context.Context, identity driver.Identity) bool {
	if p.metrics != nil {
		p.metrics.Requests.Add(1)
		p.metrics.InFlight.Add(1)
		defer p.metrics.InFlight.Add(-1)
	}
	start := time.Now()
	defer func() { p.recordMetrics(start, nil) }()

	return len(p.AreMe(ctx, identity)) > 0
}

// GetAuditInfo returns the audit information associated to the passed identity, nil otherwise.
// The audit info is retrieved from the configured storage.
func (p *Provider) GetAuditInfo(ctx context.Context, identity driver.Identity) (res []byte, err error) {
	if p.metrics != nil {
		p.metrics.Requests.Add(1)
		p.metrics.InFlight.Add(1)
		defer p.metrics.InFlight.Add(-1)
	}
	start := time.Now()
	defer func() { p.recordMetrics(start, err) }()

	if p.circuitBreaker != nil && !p.circuitBreaker.Allow() {
		return nil, errors.New("back-pressure: service temporarily overloaded, retry later")
	}

	res, err = p.storage.GetAuditInfo(ctx, identity)

	return res, err
}

// GetSigner returns a Signer for passed identity.
// If a signer cannot be retrieved an error is returned.
// Signers that can be reused are cached.
// If a signer is not found in cache,
// this provider tries to construct an instance of driver.Signer that produces valid signatures under that identity.
func (p *Provider) GetSigner(ctx context.Context, identity driver.Identity) (signer driver.Signer, err error) {
	if p.metrics != nil {
		p.metrics.Requests.Add(1)
		p.metrics.InFlight.Add(1)
		defer p.metrics.InFlight.Add(-1)
	}
	start := time.Now()
	defer func() { p.recordMetrics(start, err) }()

	if p.circuitBreaker != nil && !p.circuitBreaker.Allow() {
		return nil, errors.New("back-pressure: service temporarily overloaded, retry later")
	}

	idHash := identity.UniqueID()
	signer, err = p.getSigner(ctx, identity, idHash)
	if err != nil {
		err = errors.Wrapf(err, "failed to get signer for identity [%s], it is neither register nor deserialazable", identity.String())
	}

	return signer, err
}

// GetEIDAndRH returns both enrollment ID and revocation handle
func (p *Provider) GetEIDAndRH(ctx context.Context, identity driver.Identity, auditInfo []byte) (eid string, rh string, err error) {
	if p.metrics != nil {
		p.metrics.Requests.Add(1)
		p.metrics.InFlight.Add(1)
		defer p.metrics.InFlight.Add(-1)
	}
	start := time.Now()
	defer func() { p.recordMetrics(start, err) }()

	if p.circuitBreaker != nil && !p.circuitBreaker.Allow() {
		return "", "", errors.New("back-pressure: service temporarily overloaded, retry later")
	}

	eid, rh, err = p.enrollmentIDUnmarshaler.GetEIDAndRH(ctx, identity, auditInfo)

	return eid, rh, err
}

// GetEnrollmentID extracts the enrollment ID from the passed audit info
func (p *Provider) GetEnrollmentID(ctx context.Context, identity driver.Identity, auditInfo []byte) (eid string, err error) {
	if p.metrics != nil {
		p.metrics.Requests.Add(1)
		p.metrics.InFlight.Add(1)
		defer p.metrics.InFlight.Add(-1)
	}
	start := time.Now()
	defer func() { p.recordMetrics(start, err) }()

	if p.circuitBreaker != nil && !p.circuitBreaker.Allow() {
		return "", errors.New("back-pressure: service temporarily overloaded, retry later")
	}

	eid, err = p.enrollmentIDUnmarshaler.GetEnrollmentID(ctx, identity, auditInfo)

	return eid, err
}

// GetRevocationHandler extracts the revocation handler from the passed audit info
func (p *Provider) GetRevocationHandler(ctx context.Context, identity driver.Identity, auditInfo []byte) (rh string, err error) {
	if p.metrics != nil {
		p.metrics.Requests.Add(1)
		p.metrics.InFlight.Add(1)
		defer p.metrics.InFlight.Add(-1)
	}
	start := time.Now()
	defer func() { p.recordMetrics(start, err) }()

	if p.circuitBreaker != nil && !p.circuitBreaker.Allow() {
		return "", errors.New("back-pressure: service temporarily overloaded, retry later")
	}

	rh, err = p.enrollmentIDUnmarshaler.GetRevocationHandler(ctx, identity, auditInfo)

	return rh, err
}

// Bind binds longTerm to the passed ephemeral identities.
func (p *Provider) Bind(ctx context.Context, longTerm driver.Identity, ephemeralIdentities ...driver.Identity) (err error) {
	if p.metrics != nil {
		p.metrics.Requests.Add(1)
		p.metrics.InFlight.Add(1)
		defer p.metrics.InFlight.Add(-1)
	}
	start := time.Now()
	defer func() { p.recordMetrics(start, err) }()

	if p.circuitBreaker != nil && !p.circuitBreaker.Allow() {
		return errors.New("back-pressure: service temporarily overloaded, retry later")
	}

	for _, identity := range ephemeralIdentities {
		if identity.Equal(longTerm) {
			// no action required
			continue
		}
		err = p.Binder.Bind(ctx, longTerm, identity)
		if err != nil {
			return err
		}
	}

	return nil
}

// RegisterRecipientIdentity registers the passed identity as a third-party recipient identity.
// The wallet layer performs matching and persistence; this provider records nothing here.
func (p *Provider) RegisterRecipientIdentity(ctx context.Context, id driver.Identity) (err error) {
	if p.metrics != nil {
		p.metrics.Requests.Add(1)
		p.metrics.InFlight.Add(1)
		defer p.metrics.InFlight.Add(-1)
	}
	start := time.Now()
	defer func() { p.recordMetrics(start, err) }()

	if p.circuitBreaker != nil && !p.circuitBreaker.Allow() {
		return errors.New("back-pressure: service temporarily overloaded, retry later")
	}

	p.Logger.DebugfContext(ctx, "Registering identity [%s]", id)

	return nil
}

// RollbackPartialRecipientRegistration implements RecipientRegistrationRollback.
// This provider does not keep partial recipient-registration marks in memory;
// the hook remains for other IdentityProvider implementations.
func (p *Provider) RollbackPartialRecipientRegistration(ctx context.Context, id driver.Identity) {
	if p.metrics != nil {
		p.metrics.Requests.Add(1)
		p.metrics.InFlight.Add(1)
		defer p.metrics.InFlight.Add(-1)
	}
	start := time.Now()
	defer func() { p.recordMetrics(start, nil) }()

	p.Logger.DebugfContext(ctx, "rollback partial recipient registration for identity [%s] (no-op for this provider)", id)
}

// RegisterIdentityDescriptor stores the given identity descriptor in the configured storage.
// If alias is not nil, the alias can be used as an alternative to `idriver.IdentityDescriptor#Identity`.
func (p *Provider) RegisterIdentityDescriptor(ctx context.Context, identityDescriptor *idriver.IdentityDescriptor, alias driver.Identity) (err error) {
	if p.metrics != nil {
		p.metrics.Requests.Add(1)
		p.metrics.InFlight.Add(1)
		defer p.metrics.InFlight.Add(-1)
	}
	start := time.Now()
	defer func() { p.recordMetrics(start, err) }()

	if p.circuitBreaker != nil && !p.circuitBreaker.Allow() {
		return errors.New("back-pressure: service temporarily overloaded, retry later")
	}

	// register in the Storage
	if !identityDescriptor.Ephemeral {
		err = p.storage.RegisterIdentityDescriptor(ctx, identityDescriptor, alias)
		if err != nil {
			err = errors.Wrapf(err, "failed to register identity descriptor")

			return err
		}
	}

	// update caches
	p.Logger.DebugfContext(ctx, "update identity provider caches...")
	p.updateCaches(identityDescriptor, alias)
	p.Logger.DebugfContext(ctx, "update identity provider caches...done")

	return nil
}

func (p *Provider) areMe(ctx context.Context, identities ...driver.Identity) []string {
	p.Logger.DebugfContext(ctx, "is me [%s]?", identities)
	idHashes := make([]string, len(identities))
	for i, id := range identities {
		idHashes[i] = id.UniqueID()
	}

	result := collections.NewSet[string]()
	notFound := make([]driver.Identity, 0)

	// check local cache
	for _, id := range identities {
		if _, ok := p.signers.Get(id.UniqueID()); ok {
			p.Logger.DebugfContext(ctx, "is me [%s]? yes, from cache", id)
			result.Add(id.UniqueID())
		} else {
			notFound = append(notFound, id)
		}
	}

	if len(notFound) == 0 {
		return result.ToSlice()
	}

	// check Storage
	found, err := p.storage.GetExistingSignerInfo(ctx, notFound...)
	if err != nil {
		p.Logger.Errorf("failed checking if a signer exists [%s]", err)

		return result.ToSlice()
	}
	result.Add(found...)

	return result.ToSlice()
}

func (p *Provider) getSigner(ctx context.Context, identity driver.Identity, idHash string) (driver.Signer, error) {
	signer, _, err := p.getSignerAndCache(ctx, identity, idHash, true)

	return signer, err
}

func (p *Provider) getSignerAndCache(ctx context.Context, identity driver.Identity, idHash string, shouldCache bool) (driver.Signer, bool, error) {
	// check cache
	if entry, ok := p.signers.Get(idHash); ok {
		p.Logger.DebugfContext(ctx, "signer for [%s] found", idHash)

		return entry.Signer, false, nil
	}

	p.Logger.DebugfContext(ctx, "signer for [%s] not found, attempting to deserialize", idHash)

	// check that we have a deserializer
	if p.deserializer == nil {
		return nil, false, errors.Errorf("cannot find signer for [%s], no deserializer set", identity)
	}

	// try direct deserialization
	signer, err := p.deserializer.DeserializeSigner(ctx, identity)
	if err != nil {
		// second chance: try a TypedIdentity
		typed, err2 := UnmarshalTypedIdentity(identity)
		if err2 != nil {
			// neither deserializable nor a typed wrapper
			return nil, false, errors.Wrapf(
				err2,
				"failed to unmarshal typed identity for [%s] and failed deserialization [%s]",
				identity.String(), err,
			)
		}

		if typed.Type != driver.X509IdentityType {
			shouldCache = true
		}

		// recursively resolve the inner identity
		signer, shouldCache, err = p.getSignerAndCache(ctx, typed.Identity, typed.Identity.UniqueID(), shouldCache)
		if err != nil {
			return nil, false, errors.Wrapf(err, "failed getting signer for identity [%s]", typed.Identity)
		}
	}

	// Cache the signer for the current idHash
	if shouldCache {
		entry := &SignerEntry{Signer: signer}
		if p.Logger.IsEnabledFor(zapcore.DebugLevel) {
			entry.DebugStack = debug.Stack()
		}
		p.signers.Add(idHash, entry)
	}

	// Persist signer info for the current identity
	if err := p.storage.StoreSignerInfo(ctx, identity, nil); err != nil {
		return nil, false, errors.Wrap(err, "failed to store entry in Storage for the passed signer")
	}

	return signer, shouldCache, nil
}

func (p *Provider) updateCaches(descriptor *idriver.IdentityDescriptor, alias driver.Identity) {
	id := descriptor.Identity.UniqueID()
	setAlias := !alias.IsNone()
	aliasID := alias.UniqueID()

	// signers
	if descriptor.Signer != nil {
		entry := &SignerEntry{Signer: descriptor.Signer}
		if p.Logger.IsEnabledFor(zapcore.DebugLevel) {
			entry.DebugStack = debug.Stack()
		}
		p.signers.Add(id, entry)
		if setAlias {
			p.signers.Add(aliasID, entry)
		}
	}
}

// recordMetrics records latency and error metrics if metrics are configured.
func (p *Provider) recordMetrics(start time.Time, err error) {
	if p.metrics != nil {
		p.metrics.Latency.Observe(float64(time.Since(start).Milliseconds()))
		if err != nil {
			p.metrics.Errors.Add(1)
			if p.circuitBreaker != nil {
				p.circuitBreaker.RecordFailure()
			}
		} else if p.circuitBreaker != nil {
			p.circuitBreaker.RecordSuccess()
		}
	}
}
