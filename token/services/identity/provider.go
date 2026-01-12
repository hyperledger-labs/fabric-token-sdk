/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"context"
	"runtime/debug"
	"slices"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/cache/secondcache"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	cache2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/cache"
	"go.uber.org/zap/zapcore"
)

var (
	// This makes sure that Provider implements driver.IdentityProvider
	_ driver.IdentityProvider = &Provider{}
)

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
type NetworkBinderService = idriver.NetworkBinderService

type SignerEntry struct {
	Signer     driver.Signer
	DebugStack []byte
}

// Provider implements the driver.IdentityProvider interface.
// Provider manages identity-related concepts like signature signers, verifiers, audit information, and so on.
type Provider struct {
	Logger                  logging.Logger
	Binder                  idriver.NetworkBinderService
	enrollmentIDUnmarshaler EnrollmentIDUnmarshaler
	storage                 Storage
	deserializer            Deserializer

	isMeCache cache[bool]
	signers   cache[*SignerEntry]
}

// NewProvider returns a new instance of Provider
func NewProvider(
	logger logging.Logger,
	storage Storage,
	deserializer Deserializer,
	binder NetworkBinderService,
	enrollmentIDUnmarshaler EnrollmentIDUnmarshaler,
) *Provider {
	return &Provider{
		Logger:                  logger,
		Binder:                  binder,
		enrollmentIDUnmarshaler: enrollmentIDUnmarshaler,
		deserializer:            deserializer,
		storage:                 storage,
		isMeCache:               cache2.NewNoCache[bool](),
		signers:                 secondcache.NewTyped[*SignerEntry](50),
	}
}

// RegisterRecipientData stores the passed recipient data in the configured storage.
func (p *Provider) RegisterRecipientData(ctx context.Context, data *driver.RecipientData) error {
	return p.storage.StoreIdentityData(ctx, data.Identity, data.AuditInfo, data.TokenMetadata, data.TokenMetadataAuditInfo)
}

// RegisterSigner registers a Signer and a Verifier for passed identity.
// This is implemented via an invocation of  RegisterIdentityDescriptor using an IdentityDescriptor with empty AuditInfo.
// The audit info might or might not be already stored.
func (p *Provider) RegisterSigner(ctx context.Context, identity driver.Identity, signer driver.Signer, verifier driver.Verifier, signerInfo []byte, ephemeral bool) error {
	identityDescriptor := &idriver.IdentityDescriptor{
		Identity:   identity,
		AuditInfo:  nil,
		Signer:     signer,
		SignerInfo: signerInfo,
		Verifier:   verifier,
		Ephemeral:  ephemeral,
	}
	return p.RegisterIdentityDescriptor(ctx, identityDescriptor, nil)
}

// AreMe returns the hashes of the passed identities that have a signer registered before.
// First a local cache is checked, if not found the configured storag is queried.
func (p *Provider) AreMe(ctx context.Context, identities ...driver.Identity) []string {
	p.Logger.DebugfContext(ctx, "identity [%s] is me?", identities)

	result := make([]string, 0)
	notFound := make([]driver.Identity, 0)

	for _, id := range identities {
		uniqueID := id.UniqueID()
		if isMe, ok := p.isMeCache.Get(uniqueID); !ok {
			notFound = append(notFound, id)
		} else if isMe {
			result = append(result, uniqueID)
		}
	}
	if len(notFound) == 0 {
		return result
	}

	found := p.areMe(ctx, notFound...)
	for _, id := range notFound {
		uniqueID := id.UniqueID()
		p.isMeCache.Add(uniqueID, slices.Contains(found, uniqueID))
	}
	return append(result, found...)
}

// IsMe returns true if a signer was ever registered for the passed identity
func (p *Provider) IsMe(ctx context.Context, identity driver.Identity) bool {
	return len(p.AreMe(ctx, identity)) > 0
}

// GetAuditInfo returns the audit information associated to the passed identity, nil otherwise.
// The audit info is retrieved from the configured storage.
func (p *Provider) GetAuditInfo(ctx context.Context, identity driver.Identity) ([]byte, error) {
	return p.storage.GetAuditInfo(ctx, identity)
}

// GetSigner returns a Signer for passed identity.
// If a signer cannot be retrieved an error is returned.
// Signers that can be reused are cached.
// If a signer is not found in cache,
// this provider tries to construct an instance of driver.Signer that produces valid signatures under that identity.
func (p *Provider) GetSigner(ctx context.Context, identity driver.Identity) (driver.Signer, error) {
	found := false
	idHash := identity.UniqueID()
	defer func() {
		p.isMeCache.Add(idHash, found)
	}()
	signer, err := p.getSigner(ctx, identity, idHash)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get signer for identity [%s], it is neither register nor deserialazable", identity.String())
	}
	found = true
	return signer, nil
}

// GetEIDAndRH returns both enrollment ID and revocation handle
func (p *Provider) GetEIDAndRH(ctx context.Context, identity driver.Identity, auditInfo []byte) (string, string, error) {
	return p.enrollmentIDUnmarshaler.GetEIDAndRH(ctx, identity, auditInfo)
}

// GetEnrollmentID extracts the enrollment ID from the passed audit info
func (p *Provider) GetEnrollmentID(ctx context.Context, identity driver.Identity, auditInfo []byte) (string, error) {
	return p.enrollmentIDUnmarshaler.GetEnrollmentID(ctx, identity, auditInfo)
}

// GetRevocationHandler extracts the revocation handler from the passed audit info
func (p *Provider) GetRevocationHandler(ctx context.Context, identity driver.Identity, auditInfo []byte) (string, error) {
	return p.enrollmentIDUnmarshaler.GetRevocationHandler(ctx, identity, auditInfo)
}

// Bind binds longTerm to the passed ephemeral identities.
func (p *Provider) Bind(ctx context.Context, longTerm driver.Identity, ephemeralIdentities ...driver.Identity) error {
	for _, identity := range ephemeralIdentities {
		if identity.Equal(longTerm) {
			// no action required
			continue
		}
		if err := p.Binder.Bind(ctx, longTerm, identity); err != nil {
			return err
		}
	}
	return nil
}

// RegisterRecipientIdentity register the passed identity as a third-party recipient identity.
func (p *Provider) RegisterRecipientIdentity(ctx context.Context, id driver.Identity) error {
	p.Logger.DebugfContext(ctx, "Registering identity [%s]", id)
	p.isMeCache.Add(id.UniqueID(), false)
	return nil
}

// RegisterIdentityDescriptor stores the given identity descriptor in the configured storage.
// If alias is not nil, the alias can be used as an alternative to `idriver.IdentityDescriptor#Identity`.
func (p *Provider) RegisterIdentityDescriptor(ctx context.Context, identityDescriptor *idriver.IdentityDescriptor, alias driver.Identity) error {
	// register in the Storage
	if !identityDescriptor.Ephemeral {
		if err := p.storage.RegisterIdentityDescriptor(ctx, identityDescriptor, alias); err != nil {
			return errors.Wrapf(err, "failed to register identity descriptor")
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

		if typed.Type == "x509" {
			shouldCache = false
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
		// if the signer is set, this means that id belongs to this node
		p.isMeCache.Add(id, true)

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
