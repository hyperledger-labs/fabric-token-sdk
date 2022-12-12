/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"runtime/debug"
	"sync"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/owner"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var logger = flogging.MustGetLogger("token-sdk.driver.identity")

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

type SigService interface {
	GetAuditInfo(identity view.Identity) ([]byte, error)
	RegisterSigner(identity view.Identity, signer view2.Signer, verifier view2.Verifier) error
	IsMe(identity view.Identity) bool
	GetSigner(identity view.Identity) (view2.Signer, error)
	RegisterAuditInfo(identity view.Identity, info []byte) error
	GetVerifier(identity view.Identity) (view2.Verifier, error)
}

type Binder interface {
	Bind(longTerm view.Identity, ephemeral view.Identity) error
}

// Provider implements the driver.IdentityProvider interface
type Provider struct {
	SigService         SigService
	Binder             Binder
	DefaultFSCIdentity view.Identity

	wallets                 Wallets
	deserializers           []Deserializer
	enrollmentIDUnmarshaler EnrollmentIDUnmarshaler
	isMeCacheLock           sync.RWMutex
	isMeCache               map[string]bool
}

// NewProvider creates a new identity provider
func NewProvider(sigService SigService, binder Binder, defaultFSCIdentity view.Identity, enrollmentIDUnmarshaler EnrollmentIDUnmarshaler, wallets Wallets) *Provider {
	return &Provider{
		SigService:              sigService,
		Binder:                  binder,
		DefaultFSCIdentity:      defaultFSCIdentity,
		wallets:                 wallets,
		deserializers:           []Deserializer{},
		enrollmentIDUnmarshaler: enrollmentIDUnmarshaler,
		isMeCache:               make(map[string]bool),
	}
}

func (p *Provider) GetIdentityInfo(role driver.IdentityRole, id string) (driver.IdentityInfo, error) {
	wallet, ok := p.wallets[role]
	if !ok {
		return nil, errors.Errorf("wallet not found for role [%d]", role)
	}
	info := wallet.GetIdentityInfo(id)
	if info == nil {
		return nil, errors.Errorf("identity info not found for id [%s]", id)
	}
	return &Info{IdentityInfo: info, Provider: p}, nil
}

func (p *Provider) LookupIdentifier(role driver.IdentityRole, v interface{}) (view.Identity, string, error) {
	wallet, ok := p.wallets[role]
	if !ok {
		return nil, "", errors.Errorf("wallet not found for role [%d]", role)
	}
	id, label, err := wallet.MapToID(v)
	if err != nil {
		return nil, "", err
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("identifier for [%v] is [%s,%s]", v, id, walletIDToString(label))
	}
	return id, label, nil
}

func (p *Provider) GetAuditInfo(identity view.Identity) ([]byte, error) {
	auditInfo, err := p.SigService.GetAuditInfo(identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for recipient identity [%s]", identity.String())
	}
	return auditInfo, nil
}

func (p *Provider) RegisterSigner(identity view.Identity, signer driver.Signer, verifier driver.Verifier) error {
	return p.SigService.RegisterSigner(identity, signer, verifier)
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
	ro, err2 := owner.UnmarshallTypedIdentity(identity)
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

func (p *Provider) RegisterAuditInfo(id view.Identity, auditInfo []byte) error {
	if err := p.SigService.RegisterAuditInfo(id, auditInfo); err != nil {
		return err
	}
	return nil
}

func (p *Provider) GetEnrollmentID(auditInfo []byte) (string, error) {
	return p.enrollmentIDUnmarshaler.GetEnrollmentID(auditInfo)
}

func (p *Provider) GetRevocationHandler(auditInfo []byte) (string, error) {
	return p.enrollmentIDUnmarshaler.GetRevocationHandler(auditInfo)
}

func (p *Provider) RegisterOwnerWallet(id string, path string) error {
	logger.Debugf("register owner wallet [%s:%s]", id, path)
	return p.wallets[driver.OwnerRole].RegisterIdentity(id, path)
}

func (p *Provider) RegisterIssuerWallet(id string, path string) error {
	logger.Debugf("register issuer wallet [%s:%s]", id, path)
	return p.wallets[driver.IssuerRole].RegisterIdentity(id, path)
}

func (p *Provider) Bind(id view.Identity, to view.Identity) error {
	sigService := p.SigService

	setSV := true
	signer, err := p.GetSigner(to)
	if err != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("failed getting signer for [%s][%s][%s]", to, err, debug.Stack())
		}
		setSV = false
	}
	verifier, err := sigService.GetVerifier(to)
	if err != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("failed getting verifier for [%s][%s][%s]", to, err, debug.Stack())
		}
		verifier = nil
	}

	setAI := true
	auditInfo, err := sigService.GetAuditInfo(to)
	if err != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("failed getting audit info for [%s][%s]", to, err)
		}
		setAI = false
	}

	if setSV {
		if err := sigService.RegisterSigner(id, signer, verifier); err != nil {
			return err
		}
	}
	if setAI {
		if err := sigService.RegisterAuditInfo(id, auditInfo); err != nil {
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

func (p *Provider) WalletIDs(role driver.IdentityRole) ([]string, error) {
	wallet, ok := p.wallets[role]
	if !ok {
		return nil, errors.Errorf("wallet not found for role [%d]", role)
	}
	return wallet.IDs()
}

func (p *Provider) AddDeserializer(d Deserializer) {
	p.deserializers = append(p.deserializers, d)
}

func (p *Provider) tryDeserialization(id view.Identity) (driver.Signer, error) {
	for _, d := range p.deserializers {
		signer, err := d.DeserializeSigner(id)
		if err == nil {
			return signer, nil
		}
	}
	return nil, errors.Errorf("deserialization failed")
}

// Info wraps a driver.IdentityInfo to further register the audit info,
// and binds the new identity to the default FSC node identity
type Info struct {
	driver.IdentityInfo
	Provider *Provider
}

func (i *Info) ID() string {
	return i.IdentityInfo.ID()
}

func (i *Info) EnrollmentID() string {
	return i.IdentityInfo.EnrollmentID()
}

func (i *Info) Get() (view.Identity, []byte, error) {
	// get the identity
	id, ai, err := i.IdentityInfo.Get()
	if err != nil {
		return nil, nil, err
	}
	// register the audit info
	if err := i.Provider.RegisterAuditInfo(id, ai); err != nil {
		return nil, nil, err
	}
	// bind the identity to the default FSC node identity
	if i.Provider.Binder != nil {
		if err := i.Provider.Binder.Bind(i.Provider.DefaultFSCIdentity, id); err != nil {
			return nil, nil, err
		}
	}
	return id, ai, nil
}
