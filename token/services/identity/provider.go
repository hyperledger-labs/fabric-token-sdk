/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"runtime/debug"
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var logger = logging.MustGetLogger("token-sdk.services.identity")

// Deserializer is an interface for deserializing identities
type Deserializer interface {
	// DeserializeSigner deserializes a signer from its bytes representation
	DeserializeSigner(raw []byte) (driver.Signer, error)
}

// EnrollmentIDUnmarshaler decodes an enrollment ID form an audit info
type EnrollmentIDUnmarshaler interface {
	// GetEnrollmentID returns the enrollment ID from the audit info
	GetEnrollmentID(identity driver.Identity, auditInfo []byte) (string, error)
	// GetRevocationHandler returns the revocation handle from the audit info
	GetRevocationHandler(identity driver.Identity, auditInfo []byte) (string, error)
	// GetEIDAndRH returns both enrollment ID and revocation handle
	GetEIDAndRH(identity driver.Identity, auditInfo []byte) (string, string, error)
}

type sigService interface {
	IsMe(identity driver.Identity) bool
	RegisterSigner(identity driver.Identity, signer driver.Signer, verifier driver.Verifier, signerInfo []byte) error
	RegisterVerifier(identity driver.Identity, v driver.Verifier) error
	GetSigner(identity driver.Identity) (driver.Signer, error)
	GetSignerInfo(identity driver.Identity) ([]byte, error)
	GetVerifier(identity driver.Identity) (driver.Verifier, error)
}

type Storage interface {
	GetAuditInfo(id []byte) ([]byte, error)
	StoreIdentityData(id []byte, identityAudit []byte, tokenMetadata []byte, tokenMetadataAudit []byte) error
}

type Binder interface {
	Bind(longTerm driver.Identity, ephemeral driver.Identity) error
}

// Provider implements the driver.IdentityProvider interface.
// Provider handles the long-term identities on top of which wallets are defined.
type Provider struct {
	SigService sigService
	Binder     Binder
	Storage    Storage

	enrollmentIDUnmarshaler EnrollmentIDUnmarshaler
	isMeCacheLock           sync.RWMutex
	isMeCache               map[string]bool
}

// NewProvider creates a new identity provider implementing the driver.IdentityProvider interface.
// The Provider handles the long-term identities on top of which wallets are defined.
func NewProvider(Storage Storage, sigService sigService, binder Binder, enrollmentIDUnmarshaler EnrollmentIDUnmarshaler) *Provider {
	return &Provider{
		Storage:                 Storage,
		SigService:              sigService,
		Binder:                  binder,
		enrollmentIDUnmarshaler: enrollmentIDUnmarshaler,
		isMeCache:               make(map[string]bool),
	}
}

func (p *Provider) RegisterVerifier(identity driver.Identity, v driver.Verifier) error {
	return p.SigService.RegisterVerifier(identity, v)
}

func (p *Provider) RegisterAuditInfo(identity driver.Identity, info []byte) error {
	return p.Storage.StoreIdentityData(identity, info, nil, nil)
}

func (p *Provider) GetAuditInfo(identity driver.Identity) ([]byte, error) {
	return p.Storage.GetAuditInfo(identity)
}

func (p *Provider) RegisterRecipientData(data *driver.RecipientData) error {
	return p.Storage.StoreIdentityData(data.Identity, data.AuditInfo, data.TokenMetadata, data.TokenMetadataAuditInfo)
}

func (p *Provider) RegisterSigner(identity driver.Identity, signer driver.Signer, verifier driver.Verifier, signerInfo []byte) error {
	defer func() {
		p.isMeCacheLock.Lock()
		p.isMeCache[identity.String()] = true
		p.isMeCacheLock.Unlock()
	}()
	return p.SigService.RegisterSigner(identity, signer, verifier, signerInfo)
}

func (p *Provider) IsMe(identity driver.Identity) bool {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("identity [%s] is me?", identity)
	}
	p.isMeCacheLock.RLock()
	isMe, ok := p.isMeCache[identity.String()]
	p.isMeCacheLock.RUnlock()
	if ok {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("identity [%s] is me? [%v] from cache", identity, isMe)
		}
		return isMe
	}

	found := false
	defer func() {
		p.isMeCacheLock.Lock()
		p.isMeCache[identity.String()] = found
		p.isMeCacheLock.Unlock()
	}()
	found = p.SigService.IsMe(identity)
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("identity [%s] is me? [%v] from sig service", identity, isMe)
	}
	return found
}

func (p *Provider) RegisterRecipientIdentity(id driver.Identity) error {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("Registering identity [%s]", id)
	}
	p.isMeCacheLock.Lock()
	p.isMeCache[id.String()] = false
	p.isMeCacheLock.Unlock()
	return nil
}

func (p *Provider) GetSigner(identity driver.Identity) (driver.Signer, error) {
	found := false
	defer func() {
		p.isMeCacheLock.Lock()
		p.isMeCache[identity.String()] = found
		p.isMeCacheLock.Unlock()
	}()
	signer, err := p.SigService.GetSigner(identity)
	if err != nil {
		logger.Warn(err)
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

func (p *Provider) Bind(longTerm driver.Identity, ephemeral driver.Identity, copyAll bool) error {
	if copyAll {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("Binding ephemeral identity [%s] longTerm identity [%s]", ephemeral, longTerm)
		}
		setSV := true
		signer, err := p.GetSigner(longTerm)
		if err != nil {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("failed getting signer for [%s][%s][%s]", longTerm, err, debug.Stack())
			}
			setSV = false
		}
		verifier, err := p.SigService.GetVerifier(longTerm)
		if err != nil {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("failed getting verifier for [%s][%s][%s]", longTerm, err, debug.Stack())
			}
			verifier = nil
		}

		setAI := true
		auditInfo, err := p.GetAuditInfo(longTerm)
		if err != nil {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("failed getting audit info for [%s][%s]", longTerm, err)
			}
			setAI = false
		}

		if setSV {
			signerInfo, err := p.SigService.GetSignerInfo(longTerm)
			if err != nil {
				return err
			}
			if err := p.SigService.RegisterSigner(ephemeral, signer, verifier, signerInfo); err != nil {
				return err
			}
		}
		if setAI {
			if err := p.RegisterAuditInfo(ephemeral, auditInfo); err != nil {
				return err
			}
		}
	}

	if p.Binder != nil {
		if err := p.Binder.Bind(longTerm, ephemeral); err != nil {
			return err
		}
	}
	return nil
}
