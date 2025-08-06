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
	"go.uber.org/zap/zapcore"
)

var logger = logging.MustGetLogger()

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
	StoreSignerInfo(ctx context.Context, id, info []byte) error
	GetExistingSignerInfo(ctx context.Context, ids ...driver.Identity) ([]string, error)
	SignerInfoExists(ctx context.Context, id []byte) (bool, error)
	GetSignerInfo(ctx context.Context, identity []byte) ([]byte, error)
}

type cache[T any] interface {
	Get(key string) (T, bool)
	Add(key string, value T)
	Delete(key string)
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
	Logger  logging.Logger
	Binder  idriver.NetworkBinderService
	Storage storage

	enrollmentIDUnmarshaler enrollmentIDUnmarshaler
	isMeCache               cache[bool]

	signers   cache[*SignerEntry]
	verifiers cache[*VerifierEntry]

	storage      storage
	deserializer idriver.Deserializer
}

// NewProvider creates a new identity provider implementing the driver.IdentityProvider interface.
// The Provider handles the long-term identities on top of which wallets are defined.
func NewProvider(
	logger logging.Logger,
	storage storage,
	deserializer idriver.Deserializer,
	binder idriver.NetworkBinderService,
	enrollmentIDUnmarshaler enrollmentIDUnmarshaler,
) *Provider {
	return &Provider{
		Logger:                  logger,
		Storage:                 storage,
		Binder:                  binder,
		enrollmentIDUnmarshaler: enrollmentIDUnmarshaler,
		isMeCache:               secondcache.NewTyped[bool](1000),

		signers:      secondcache.NewTyped[*SignerEntry](1000),
		verifiers:    secondcache.NewTyped[*VerifierEntry](1000),
		deserializer: deserializer,
		storage:      storage,
	}
}

func (p *Provider) RegisterIdentityDescriptor(ctx context.Context, identityDescriptor *idriver.IdentityDescriptor, alias driver.Identity) error {
	id := identityDescriptor.Identity
	if err := p.RegisterSigner(
		ctx,
		id,
		identityDescriptor.Signer,
		identityDescriptor.Verifier,
		identityDescriptor.SignerInfo,
	); err != nil {
		return errors2.Wrapf(err, "failed to register signer")
	}
	if err := p.RegisterAuditInfo(ctx, id, identityDescriptor.AuditInfo); err != nil {
		return errors2.Wrapf(err, "failed to register audit info for identity [%s]", id)
	}
	// typedIdentity has id's signer and verifier
	if err := p.Copy(ctx, id, alias); err != nil {
		return errors2.Wrapf(err, "failed to bind identity [%s] to [%s]", alias, id)
	}
	return nil
}

func (p *Provider) RegisterVerifier(ctx context.Context, identity driver.Identity, v driver.Verifier) error {
	return p.localRegisterVerifier(ctx, identity, v)
}

func (p *Provider) RegisterAuditInfo(ctx context.Context, identity driver.Identity, info []byte) error {
	return p.Storage.StoreIdentityData(ctx, identity, info, nil, nil)
}

func (p *Provider) GetAuditInfo(ctx context.Context, identity driver.Identity) ([]byte, error) {
	return p.Storage.GetAuditInfo(ctx, identity)
}

func (p *Provider) RegisterRecipientData(ctx context.Context, data *driver.RecipientData) error {
	return p.Storage.StoreIdentityData(ctx, data.Identity, data.AuditInfo, data.TokenMetadata, data.TokenMetadataAuditInfo)
}

func (p *Provider) RegisterSigner(ctx context.Context, identity driver.Identity, signer driver.Signer, verifier driver.Verifier, signerInfo []byte) error {
	defer func() {
		p.isMeCache.Add(identity.UniqueID(), true)
	}()
	return p.localRegisterSigner(ctx, identity, signer, verifier, signerInfo)
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

	found := p.localAreMe(ctx, notFound...)
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
	defer func() {
		p.isMeCache.Add(identity.UniqueID(), found)
	}()
	signer, err := p.localGetSigner(ctx, identity)
	if err != nil {
		p.Logger.Warn(err)
		return nil, errors.Errorf("failed to get signer for identity [%s], it is neither register nor deserialazable", identity.String())
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

func (p *Provider) Copy(ctx context.Context, longTerm driver.Identity, ephemeral driver.Identity) error {
	if ephemeral.Equal(longTerm) {
		// no action required
		return nil
	}

	p.Logger.DebugfContext(ctx, "Binding ephemeral identity [%s] longTerm identity [%s]", ephemeral, longTerm)
	setSV := true
	signer, err := p.GetSigner(ctx, longTerm)
	if err != nil {
		if p.Logger.IsEnabledFor(zapcore.DebugLevel) {
			p.Logger.DebugfContext(ctx, "failed getting signer for [%s][%s][%s]", longTerm, err, string(debug.Stack()))
		}
		setSV = false
	}
	verifier, err := p.localGetVerifier(ctx, longTerm)
	if err != nil {
		if p.Logger.IsEnabledFor(zapcore.DebugLevel) {
			p.Logger.DebugfContext(ctx, "failed getting verifier for identity [%s][%s][%s]", longTerm, err, string(debug.Stack()))
		}
		verifier = nil
	}

	setAI := true
	auditInfo, err := p.GetAuditInfo(ctx, longTerm)
	if err != nil {
		p.Logger.DebugfContext(ctx, "failed getting audit info for [%s][%s]", longTerm, err)
		setAI = false
	}

	if setSV {
		signerInfo, err := p.localGetSignerInfo(ctx, longTerm)
		if err != nil {
			return err
		}
		if err := p.localRegisterSigner(ctx, ephemeral, signer, verifier, signerInfo); err != nil {
			return err
		}
	}
	if setAI {
		if err := p.RegisterAuditInfo(ctx, ephemeral, auditInfo); err != nil {
			return err
		}
	}

	return nil
}

func (p *Provider) Bind(ctx context.Context, longTerm driver.Identity, ephemeral driver.Identity) error {
	if ephemeral.Equal(longTerm) {
		// no action required
		return nil
	}

	if err := p.Binder.Bind(ctx, longTerm, ephemeral); err != nil {
		return err
	}
	return nil
}

func (p *Provider) localRegisterSigner(ctx context.Context, identity driver.Identity, signer driver.Signer, verifier driver.Verifier, signerInfo []byte) error {
	idHash := identity.UniqueID()
	logger.DebugfContext(ctx, "register signer and verifier [%s]:[%s][%s]", idHash, logging.Identifier(signer), logging.Identifier(verifier))

	if signer != nil {
		if err := p.registerSigner(ctx, identity, signer, verifier, signerInfo); err != nil {
			return err
		}
	}

	if verifier != nil {
		if err := p.RegisterVerifier(ctx, identity, verifier); err != nil {
			p.deleteSigner(idHash)
			return err
		}
	}

	logger.DebugfContext(ctx, "register signer and verifier [%s]:[%s][%s], done", idHash, logging.Identifier(signer), logging.Identifier(verifier))
	return nil
}

func (p *Provider) localRegisterVerifier(ctx context.Context, identity driver.Identity, verifier driver.Verifier) error {
	if verifier == nil {
		return errors.New("invalid verifier, expected a valid instance")
	}

	idHash := identity.UniqueID()
	entry := &VerifierEntry{Verifier: verifier}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		entry.DebugStack = debug.Stack()
	}
	p.verifiers.Add(idHash, entry)

	logger.DebugfContext(ctx, "register verifier to [%s]:[%s]", idHash, logging.Identifier(verifier))
	return nil
}

func (p *Provider) localAreMe(ctx context.Context, identities ...driver.Identity) []string {
	logger.DebugfContext(ctx, "is me [%s]?", identities)
	idHashes := make([]string, len(identities))
	for i, id := range identities {
		idHashes[i] = id.UniqueID()
	}

	result := collections.NewSet[string]()
	notFound := make([]driver.Identity, 0)

	// check local cache
	for _, id := range identities {
		if _, ok := p.signers.Get(id.UniqueID()); ok {
			logger.DebugfContext(ctx, "is me [%s]? yes, from cache", id)
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
		logger.Errorf("failed checking if a signer exists [%s]", err)
		return result.ToSlice()
	}
	result.Add(found...)
	return result.ToSlice()
}

func (p *Provider) localGetSigner(ctx context.Context, identity driver.Identity) (driver.Signer, error) {
	idHash := identity.UniqueID()
	return p.getSigner(ctx, identity, idHash)
}

func (p *Provider) localGetSignerInfo(ctx context.Context, identity driver.Identity) ([]byte, error) {
	return p.storage.GetSignerInfo(ctx, identity)
}

func (p *Provider) localGetVerifier(ctx context.Context, identity driver.Identity) (driver.Verifier, error) {
	idHash := identity.UniqueID()

	// check cache
	entry, ok := p.verifiers.Get(idHash)
	if ok {
		return entry.Verifier, nil
	}

	// ask the deserializer
	if p.deserializer == nil {
		return nil, errors.Errorf("cannot find verifier for [%s], no deserializer set", identity)
	}
	var err error
	verifier, err := p.deserializer.DeserializeVerifier(ctx, identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed deserializing identity for verifier %v", identity)
	}

	// store entry
	entry = &VerifierEntry{Verifier: verifier}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		entry.DebugStack = debug.Stack()
	}
	logger.DebugfContext(ctx, "add deserialized verifier for [%s]:[%s]", idHash, logging.Identifier(verifier))
	p.verifiers.Add(idHash, entry)
	return verifier, nil
}

func (p *Provider) registerSigner(ctx context.Context, identity driver.Identity, signer driver.Signer, verifier driver.Verifier, signerInfo []byte) error {
	if signer == nil {
		// Nothing to do here
		return nil
	}

	idHash := identity.UniqueID()
	logger.DebugfContext(ctx, "register signer and verifier [%s]:[%s][%s]", idHash, logging.Identifier(signer), logging.Identifier(verifier))
	// First check with read lock
	s, ok := p.signers.Get(idHash)
	if ok {
		logger.Warnf("another signer bound to [%s]:[%s][%s] from [%s]", identity, logging.Identifier(s), logging.Identifier(signer), string(s.DebugStack))
		return nil
	}

	entry := &SignerEntry{Signer: signer}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		entry.DebugStack = debug.Stack()
	}
	p.signers.Add(idHash, entry)

	// store, if a failure happens then remove the entry
	logger.DebugfContext(ctx, "checks done, store signer info")
	if err := p.storage.StoreSignerInfo(ctx, identity, signerInfo); err != nil {
		p.deleteSigner(idHash)
		return errors.Wrap(err, "failed to store entry in storage for the passed signer")
	}

	return nil
}

func (p *Provider) getSigner(ctx context.Context, identity driver.Identity, idHash string) (driver.Signer, error) {
	// check again the cache
	entry, ok := p.signers.Get(idHash)
	if ok {
		logger.DebugfContext(ctx, "signer for [%s] found", idHash)
		return entry.Signer, nil
	}

	logger.DebugfContext(ctx, "signer for [%s] not found, try to deserialize", idHash)
	// ask the deserializer
	signer, err := p.deserializeSigner(ctx, identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed deserializing identity for signer [%s]", identity)
	}
	entry = &SignerEntry{Signer: signer}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
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

func (p *Provider) deleteSigner(id string) {
	p.signers.Delete(id)
}
