/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"runtime/debug"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/deserializer"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var logger = flogging.MustGetLogger("token-sdk.services.identity")

// Deserializer is an interface for deserializing identities
type Deserializer interface {
	// DeserializeSigner deserializes a signer from its bytes representation
	DeserializeSigner(raw []byte) (driver.Signer, error)
}

// EnrollmentIDUnmarshaler decodes an enrollment ID form an audit info
type EnrollmentIDUnmarshaler interface {
	// GetEnrollmentID returns the enrollment ID from the audit info
	GetEnrollmentID(auditInfo []byte) (string, error)
	// GetRevocationHandler returns the revocation handle from the audit info
	GetRevocationHandler(auditInfo []byte) (string, error)
}

type sigService interface {
	IsMe(identity view.Identity) bool
	RegisterSigner(identity view.Identity, signer driver.Signer, verifier driver.Verifier, signerInfo []byte) error
	RegisterVerifier(identity view.Identity, v driver.Verifier) error
	GetSigner(identity view.Identity) (driver.Signer, error)
	GetSignerInfo(identity view.Identity) ([]byte, error)
	GetVerifier(identity view.Identity) (driver.Verifier, error)
}

type Storage interface {
	GetAuditInfo(id []byte) ([]byte, error)
	StoreIdentityData(id []byte, identityAudit []byte, tokenMetadata []byte, tokenMetadataAudit []byte) error
}

type Binder interface {
	Bind(longTerm view.Identity, ephemeral view.Identity) error
}

// Provider implements the driver.IdentityProvider interface.
// Provider handles the long-term identities on top of which wallets are defined.
type Provider struct {
	SigService sigService
	Binder     Binder
	Storage    Storage

	deserializerManager     deserializer.Manager
	enrollmentIDUnmarshaler EnrollmentIDUnmarshaler
	isMeCacheLock           sync.RWMutex
	isMeCache               map[string]bool
}

// NewProvider creates a new identity provider implementing the driver.IdentityProvider interface.
// The Provider handles the long-term identities on top of which wallets are defined.
func NewProvider(Storage Storage, sigService sigService, binder Binder, enrollmentIDUnmarshaler EnrollmentIDUnmarshaler, deserializerManager deserializer.Manager) *Provider {
	return &Provider{
		Storage:                 Storage,
		SigService:              sigService,
		Binder:                  binder,
		deserializerManager:     deserializerManager,
		enrollmentIDUnmarshaler: enrollmentIDUnmarshaler,
		isMeCache:               make(map[string]bool),
	}
}

func (p *Provider) RegisterVerifier(identity view.Identity, v driver.Verifier) error {
	return p.SigService.RegisterVerifier(identity, v)
}

func (p *Provider) RegisterAuditInfo(identity view.Identity, info []byte) error {
	return p.Storage.StoreIdentityData(identity, info, nil, nil)
}

func (p *Provider) GetAuditInfo(identity view.Identity) ([]byte, error) {
	return p.Storage.GetAuditInfo(identity)
}

func (p *Provider) RegisterRecipientData(data *driver.RecipientData) error {
	return p.Storage.StoreIdentityData(data.Identity, data.AuditInfo, data.TokenMetadata, data.TokenMetadataAuditInfo)
}

func (p *Provider) RegisterSigner(identity view.Identity, signer driver.Signer, verifier driver.Verifier, signerInfo []byte) error {
	return p.SigService.RegisterSigner(identity, signer, verifier, signerInfo)
}

func (p *Provider) IsMe(identity view.Identity) bool {
	isMe := p.SigService.IsMe(identity)
	if !isMe {
		// check cache
		p.isMeCacheLock.RLock()
		isMe, ok := p.isMeCache[identity.String()]
		p.isMeCacheLock.RUnlock()
		if ok {
			return isMe
		}

		// try to get the signer
		signer, err := p.GetSigner(identity)
		if err != nil {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("failed to get signer for identity [%s]", identity.String())
			}
			return false
		}
		return signer != nil
	}
	return true
}

func (p *Provider) RegisterRecipientIdentity(id view.Identity) error {
	p.isMeCacheLock.Lock()
	p.isMeCache[id.String()] = false
	p.isMeCacheLock.Unlock()
	return nil
}

func (p *Provider) GetSigner(identity view.Identity) (driver.Signer, error) {
	found := false
	defer func() {
		p.isMeCacheLock.Lock()
		p.isMeCache[identity.String()] = found
		p.isMeCacheLock.Unlock()
	}()
	signer, err := p.SigService.GetSigner(identity)
	if err == nil {
		found = true
		return signer, nil
	}

	// give it a second chance

	// is the identity wrapped in TypedIdentity?
	ro, err2 := UnmarshalTypedIdentity(identity)
	if err2 != nil {
		// No
		signer, err := p.tryDeserialization(identity)
		if err == nil {
			found = true
			return signer, nil
		}

		found = false
		return nil, errors.Wrapf(err2, "failed to unmarshal raw owner for identity [%s] and failed deserialization [%s]", identity.String(), err)
	}

	// yes, check ro.Identity
	signer, err = p.SigService.GetSigner(ro.Identity)
	if err == nil {
		found = true
		return signer, nil
	}

	// give it a third chance
	// deserializer using the provider's deserializers
	signer, err = p.tryDeserialization(ro.Identity)
	if err == nil {
		found = true
		return signer, nil
	}

	return nil, errors.Errorf("failed to get signer for identity [%s], it is neither register nor deserialazable", identity.String())
}

func (p *Provider) GetEnrollmentID(auditInfo []byte) (string, error) {
	return p.enrollmentIDUnmarshaler.GetEnrollmentID(auditInfo)
}

func (p *Provider) GetRevocationHandler(auditInfo []byte) (string, error) {
	return p.enrollmentIDUnmarshaler.GetRevocationHandler(auditInfo)
}

func (p *Provider) Bind(id view.Identity, to view.Identity) error {
	setSV := true
	signer, err := p.GetSigner(to)
	if err != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("failed getting signer for [%s][%s][%s]", to, err, debug.Stack())
		}
		setSV = false
	}
	verifier, err := p.SigService.GetVerifier(to)
	if err != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("failed getting verifier for [%s][%s][%s]", to, err, debug.Stack())
		}
		verifier = nil
	}

	setAI := true
	auditInfo, err := p.GetAuditInfo(to)
	if err != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("failed getting audit info for [%s][%s]", to, err)
		}
		setAI = false
	}

	if setSV {
		signerInfo, err := p.SigService.GetSignerInfo(id)
		if err != nil {
			return err
		}
		if err := p.SigService.RegisterSigner(id, signer, verifier, signerInfo); err != nil {
			return err
		}
	}
	if setAI {
		if err := p.RegisterAuditInfo(id, auditInfo); err != nil {
			return err
		}
	}

	if p.Binder != nil {
		if err := p.Binder.Bind(to, id); err != nil {
			return err
		}
	}
	return nil
}

func (p *Provider) tryDeserialization(id view.Identity) (driver.Signer, error) {
	return p.deserializerManager.DeserializeSigner(id)
}
