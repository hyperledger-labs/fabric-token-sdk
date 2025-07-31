/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"context"
	"runtime/debug"
	"slices"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/cache/secondcache"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type StorageProvider = idriver.StorageProvider

// enrollmentIDUnmarshaler decodes an enrollment ID form an audit info
type enrollmentIDUnmarshaler interface {
	// GetEnrollmentID returns the enrollment ID from the audit info
	GetEnrollmentID(identity driver.Identity, auditInfo []byte) (string, error)
	// GetRevocationHandler returns the revocation handle from the audit info
	GetRevocationHandler(identity driver.Identity, auditInfo []byte) (string, error)
	// GetEIDAndRH returns both enrollment ID and revocation handle
	GetEIDAndRH(identity driver.Identity, auditInfo []byte) (string, string, error)
}

type sigService interface {
	IsMe(ctx context.Context, identity driver.Identity) bool
	AreMe(ctx context.Context, identities ...driver.Identity) []string
	RegisterSigner(ctx context.Context, identity driver.Identity, signer driver.Signer, verifier driver.Verifier, signerInfo []byte) error
	RegisterVerifier(ctx context.Context, identity driver.Identity, v driver.Verifier) error
	GetSigner(ctx context.Context, identity driver.Identity) (driver.Signer, error)
	GetSignerInfo(ctx context.Context, identity driver.Identity) ([]byte, error)
	GetVerifier(identity driver.Identity) (driver.Verifier, error)
}

type storage interface {
	GetAuditInfo(ctx context.Context, id []byte) ([]byte, error)
	StoreIdentityData(ctx context.Context, id []byte, identityAudit []byte, tokenMetadata []byte, tokenMetadataAudit []byte) error
}

type cache[T any] interface {
	Get(key string) (T, bool)
	Add(key string, value T)
}

// Provider implements the driver.IdentityProvider interface.
// Provider handles the long-term identities on top of which wallets are defined.
type Provider struct {
	Logger     logging.Logger
	SigService sigService
	Binder     idriver.NetworkBinderService
	Storage    storage

	enrollmentIDUnmarshaler enrollmentIDUnmarshaler
	isMeCache               cache[bool]
}

// NewProvider creates a new identity provider implementing the driver.IdentityProvider interface.
// The Provider handles the long-term identities on top of which wallets are defined.
func NewProvider(
	logger logging.Logger,
	storage storage,
	sigService sigService,
	binder idriver.NetworkBinderService,
	enrollmentIDUnmarshaler enrollmentIDUnmarshaler,
) *Provider {
	return &Provider{
		Logger:                  logger,
		Storage:                 storage,
		SigService:              sigService,
		Binder:                  binder,
		enrollmentIDUnmarshaler: enrollmentIDUnmarshaler,
		isMeCache:               secondcache.NewTyped[bool](1000),
	}
}

func (p *Provider) RegisterVerifier(ctx context.Context, identity driver.Identity, v driver.Verifier) error {
	return p.SigService.RegisterVerifier(ctx, identity, v)
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
	return p.SigService.RegisterSigner(ctx, identity, signer, verifier, signerInfo)
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

	found := p.SigService.AreMe(ctx, notFound...)
	for _, id := range notFound {
		uniqueID := id.UniqueID()
		p.isMeCache.Add(uniqueID, slices.Contains(found, uniqueID))
	}
	return append(result, found...)
}

func (p *Provider) IsMe(ctx context.Context, identity driver.Identity) bool {
	return len(p.AreMe(ctx, identity)) > 0
}

func (p *Provider) RegisterRecipientIdentity(id driver.Identity) error {
	p.Logger.Debugf("Registering identity [%s]", id)
	p.isMeCache.Add(id.UniqueID(), false)
	return nil
}

func (p *Provider) GetSigner(ctx context.Context, identity driver.Identity) (driver.Signer, error) {
	found := false
	defer func() {
		p.isMeCache.Add(identity.UniqueID(), found)
	}()
	signer, err := p.SigService.GetSigner(ctx, identity)
	if err != nil {
		p.Logger.Warn(err)
		return nil, errors.Errorf("failed to get signer for identity [%s], it is neither register nor deserialazable", identity.String())
	}
	found = true
	return signer, nil
}

func (p *Provider) GetEIDAndRH(identity driver.Identity, auditInfo []byte) (string, string, error) {
	return p.enrollmentIDUnmarshaler.GetEIDAndRH(identity, auditInfo)
}

func (p *Provider) GetEnrollmentID(identity driver.Identity, auditInfo []byte) (string, error) {
	return p.enrollmentIDUnmarshaler.GetEnrollmentID(identity, auditInfo)
}

func (p *Provider) GetRevocationHandler(identity driver.Identity, auditInfo []byte) (string, error) {
	return p.enrollmentIDUnmarshaler.GetRevocationHandler(identity, auditInfo)
}

func (p *Provider) Bind(ctx context.Context, longTerm driver.Identity, ephemeral driver.Identity, copyAll bool) error {
	if copyAll {
		p.Logger.DebugfContext(ctx, "Binding ephemeral identity [%s] longTerm identity [%s]", ephemeral, longTerm)
		setSV := true
		signer, err := p.GetSigner(ctx, longTerm)
		if err != nil {
			if p.Logger.IsEnabledFor(zapcore.DebugLevel) {
				p.Logger.DebugfContext(ctx, "failed getting signer for [%s][%s][%s]", longTerm, err, string(debug.Stack()))
			}
			setSV = false
		}
		verifier, err := p.SigService.GetVerifier(longTerm)
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
			signerInfo, err := p.SigService.GetSignerInfo(ctx, longTerm)
			if err != nil {
				return err
			}
			if err := p.SigService.RegisterSigner(ctx, ephemeral, signer, verifier, signerInfo); err != nil {
				return err
			}
		}
		if setAI {
			if err := p.RegisterAuditInfo(ctx, ephemeral, auditInfo); err != nil {
				return err
			}
		}
	}

	if p.Binder != nil {
		if err := p.Binder.Bind(ctx, longTerm, ephemeral); err != nil {
			return err
		}
	}
	return nil
}
