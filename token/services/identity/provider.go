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
	RegisterSigner(identity view.Identity, signer driver.Signer, verifier driver.Verifier) error
	RegisterAuditInfo(identity view.Identity, info []byte) error
	GetAuditInfo(identity view.Identity) ([]byte, error)
	GetSigner(identity view.Identity) (driver.Signer, error)
	GetVerifier(identity view.Identity) (driver.Verifier, error)
}

type Binder interface {
	Bind(longTerm view.Identity, ephemeral view.Identity) error
}

// Provider implements the driver.IdentityProvider interface.
// Provider handles the long-term identities on top of which wallets are defined.
type Provider struct {
	SigService         sigService
	Binder             Binder
	DefaultFSCIdentity view.Identity

	roles                   Roles
	deserializerManager     deserializer.Manager
	enrollmentIDUnmarshaler EnrollmentIDUnmarshaler
	isMeCacheLock           sync.RWMutex
	isMeCache               map[string]bool
}

// NewProvider creates a new identity provider implementing the driver.IdentityProvider interface.
// The Provider handles the long-term identities on top of which wallets are defined.
func NewProvider(
	sigService sigService,
	binder Binder,
	defaultFSCIdentity view.Identity,
	enrollmentIDUnmarshaler EnrollmentIDUnmarshaler,
	roles Roles,
	deserializerManager deserializer.Manager,
) *Provider {
	return &Provider{
		SigService:              sigService,
		Binder:                  binder,
		DefaultFSCIdentity:      defaultFSCIdentity,
		roles:                   roles,
		deserializerManager:     deserializerManager,
		enrollmentIDUnmarshaler: enrollmentIDUnmarshaler,
		isMeCache:               make(map[string]bool),
	}
}

func (p *Provider) MapToID(roleID driver.IdentityRole, v interface{}) (view.Identity, string, error) {
	role, ok := p.roles[roleID]
	if !ok {
		return nil, "", errors.Errorf("role not found [%d]", roleID)
	}
	id, label, err := role.MapToID(v)
	if err != nil {
		return nil, "", err
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("identifier for [%v] is [%s,%s]", v, id, toString(label))
	}
	return id, label, nil
}

// GetIdentityInfo returns the long-term identity info associated to the passed id.
// When IdentityInfo#Get function is invoked, two things happen:
// 1. The relative audit-info are stored.
// 2. The identity is bound to the long-term identity of the network node this stack is running on.
func (p *Provider) GetIdentityInfo(roleID driver.IdentityRole, id string) (driver.IdentityInfo, error) {
	role, ok := p.roles[roleID]
	if !ok {
		return nil, errors.Errorf("role not found [%d]", roleID)
	}
	info := role.GetIdentityInfo(id)
	if info == nil {
		return nil, errors.Errorf("identity info not found for id [%s]", id)
	}
	return &Info{IdentityInfo: info, Provider: p}, nil
}

func (p *Provider) RegisterIdentity(roleID driver.IdentityRole, id string, path string) error {
	role, ok := p.roles[roleID]
	if ok {
		logger.Debugf("register identity [role:%d][%s:%s]", roleID, id, path)
		return role.RegisterIdentity(id, path)
	}
	return errors.Errorf("cannot find role [%d]", roleID)
}

func (p *Provider) IDs(roleID driver.IdentityRole) ([]string, error) {
	role, ok := p.roles[roleID]
	if !ok {
		return nil, errors.Errorf("role not found [%d]", roleID)
	}
	return role.IDs()
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
	ro, err2 := UnmarshallTypedIdentity(identity)
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

func (p *Provider) tryDeserialization(id view.Identity) (driver.Signer, error) {
	return p.deserializerManager.DeserializeSigner(id)
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
