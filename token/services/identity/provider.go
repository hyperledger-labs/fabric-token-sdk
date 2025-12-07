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

type StorageProvider = idriver.StorageProvider

// enrollmentIDUnmarshaler decodes an enrollment ID form an audit info
type enrollmentIDUnmarshaler interface {
	// GetEnrollmentID returns the enrollment ID from the audit info
	GetEnrollmentID(ctx context.Context, identity driver.Identity, auditInfo []byte) (string, error)
	// GetRevocationHandler returns the revocation handle from the audit info
	GetRevocationHandler(ctx context.Context, identity driver.Identity, auditInfo []byte) (string, error)
	// GetEIDAndRH returns both enrollment ID and revocation handle
	GetEIDAndRH(ctx context.Context, identity driver.Identity, auditInfo []byte) (string, string, error)
}

type storage interface {
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

type deserializer interface {
	DeserializeSigner(ctx context.Context, raw []byte) (driver.Signer, error)
}

type VerifierEntry struct {
	Verifier   driver.Verifier
	DebugStack []byte
}

type SignerEntry struct {
	Signer     driver.Signer
	DebugStack []byte
}

// Provider implements the driver.IdentityProvider interface.
// Provider handles the long-term identities on top of which wallets are defined.
type Provider struct {
	Logger                  logging.Logger
	Binder                  idriver.NetworkBinderService
	enrollmentIDUnmarshaler enrollmentIDUnmarshaler
	storage                 storage
	deserializer            deserializer

	isMeCache cache[bool]
	signers   cache[*SignerEntry]
}

// NewProvider creates a new identity provider implementing the driver.IdentityProvider interface.
// The Provider handles the long-term identities on top of which wallets are defined.
func NewProvider(
	logger logging.Logger,
	storage storage,
	deserializer deserializer,
	binder idriver.NetworkBinderService,
	enrollmentIDUnmarshaler enrollmentIDUnmarshaler,
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

func (p *Provider) RegisterIdentityDescriptor(ctx context.Context, identityDescriptor *idriver.IdentityDescriptor, alias driver.Identity) error {
	// register in the storage
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

func (p *Provider) RegisterVerifier(ctx context.Context, identity driver.Identity, v driver.Verifier) error {
	if v == nil {
		return errors.New("invalid verifier, expected a valid instance")
	}
	return nil
}

func (p *Provider) RegisterAuditInfo(ctx context.Context, identity driver.Identity, info []byte) error {
	return p.storage.StoreIdentityData(ctx, identity, info, nil, nil)
}

func (p *Provider) GetAuditInfo(ctx context.Context, identity driver.Identity) ([]byte, error) {
	return p.storage.GetAuditInfo(ctx, identity)
}

func (p *Provider) RegisterRecipientData(ctx context.Context, data *driver.RecipientData) error {
	return p.storage.StoreIdentityData(ctx, data.Identity, data.AuditInfo, data.TokenMetadata, data.TokenMetadataAuditInfo)
}

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

func (p *Provider) IsMe(ctx context.Context, identity driver.Identity) bool {
	return len(p.AreMe(ctx, identity)) > 0
}

func (p *Provider) RegisterRecipientIdentity(ctx context.Context, id driver.Identity) error {
	p.Logger.DebugfContext(ctx, "Registering identity [%s]", id)
	p.isMeCache.Add(id.UniqueID(), false)
	return nil
}

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

func (p *Provider) GetEIDAndRH(ctx context.Context, identity driver.Identity, auditInfo []byte) (string, string, error) {
	return p.enrollmentIDUnmarshaler.GetEIDAndRH(ctx, identity, auditInfo)
}

func (p *Provider) GetEnrollmentID(ctx context.Context, identity driver.Identity, auditInfo []byte) (string, error) {
	return p.enrollmentIDUnmarshaler.GetEnrollmentID(ctx, identity, auditInfo)
}

func (p *Provider) GetRevocationHandler(ctx context.Context, identity driver.Identity, auditInfo []byte) (string, error) {
	return p.enrollmentIDUnmarshaler.GetRevocationHandler(ctx, identity, auditInfo)
}

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

	// check storage
	found, err := p.storage.GetExistingSignerInfo(ctx, notFound...)
	if err != nil {
		p.Logger.Errorf("failed checking if a signer exists [%s]", err)
		return result.ToSlice()
	}
	result.Add(found...)
	return result.ToSlice()
}

func (p *Provider) getSigner(ctx context.Context, identity driver.Identity, idHash string) (driver.Signer, error) {
	// check again the cache
	entry, ok := p.signers.Get(idHash)
	if ok {
		p.Logger.DebugfContext(ctx, "signer for [%s] found", idHash)
		return entry.Signer, nil
	}

	p.Logger.DebugfContext(ctx, "signer for [%s] not found, try to deserialize", idHash)
	// ask the deserializer
	signer, err := p.deserializeSigner(ctx, identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed deserializing identity for signer [%s]", identity)
	}

	entry = &SignerEntry{Signer: signer}
	if p.Logger.IsEnabledFor(zapcore.DebugLevel) {
		entry.DebugStack = debug.Stack()
	}
	p.signers.Add(idHash, entry)
	if err := p.storage.StoreSignerInfo(ctx, identity, nil); err != nil {
		return nil, errors.Wrap(err, "failed to store entry in storage for the passed signer")
	}

	return entry.Signer, nil
}

func (p *Provider) deserializeSigner(ctx context.Context, identity driver.Identity) (driver.Signer, error) {
	if p.deserializer == nil {
		return nil, errors.Errorf("cannot find signer for [%s], no deserializer set", identity)
	}
	var err error
	signer, err := p.deserializer.DeserializeSigner(ctx, identity)
	if err == nil {
		return signer, nil
	}

	// give it a second chance

	// is the identity wrapped in TypedIdentity?
	ro, err2 := UnmarshalTypedIdentity(identity)
	if err2 != nil {
		// No
		return nil, errors.Wrapf(err2, "failed to unmarshal raw owner for identity [%s] and failed deserialization [%s]", identity.String(), err)
	}

	// yes, check ro.Identity
	signer, err = p.getSigner(ctx, ro.Identity, ro.Identity.UniqueID())
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting signer for identity [%s]", ro.Identity)
	}
	return signer, nil
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
