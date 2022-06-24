/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"fmt"
	"runtime/debug"
	"sync"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var logger = flogging.MustGetLogger("token-sdk.driver.identity")

type Deserializer interface {
	DeserializeSigner(raw []byte) (driver.Signer, error)
}

type EnrollmentIDUnmarshaler interface {
	GetEnrollmentID(auditInfo []byte) (string, error)
}

type Provider struct {
	sp view2.ServiceProvider

	mappers                 Mappers
	deserializers           []Deserializer
	enrollmentIDUnmarshaler EnrollmentIDUnmarshaler
	isMeCacheLock           sync.RWMutex
	isMeCache               map[string]bool
}

func NewProvider(sp view2.ServiceProvider, enrollmentIDUnmarshaler EnrollmentIDUnmarshaler, mappers Mappers) *Provider {
	return &Provider{
		sp:                      sp,
		mappers:                 mappers,
		deserializers:           []Deserializer{},
		enrollmentIDUnmarshaler: enrollmentIDUnmarshaler,
		isMeCache:               make(map[string]bool),
	}
}

func (p *Provider) GetIdentityInfo(usage driver.IdentityUsage, id string) driver.IdentityInfo {
	mapper, ok := p.mappers[usage]
	if !ok {
		panic(fmt.Sprintf("mapper not found for usage [%d]", usage))
	}
	info := mapper.GetIdentityInfo(id)
	if info == nil {
		return nil
	}
	return &Info{IdentityInfo: info, Provider: p}
}

func (p *Provider) LookupIdentifier(usage driver.IdentityUsage, v interface{}) (view.Identity, string) {
	mapper, ok := p.mappers[usage]
	if !ok {
		panic(fmt.Sprintf("mapper not found for usage [%d]", usage))
	}
	id, label := mapper.MapToID(v)
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("identifier for [%v] is [%s,%s]", v, id, label)
	}
	return id, label
}

func (p *Provider) GetAuditInfo(identity view.Identity) ([]byte, error) {
	auditInfo, err := view2.GetSigService(p.sp).GetAuditInfo(identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for recipient identity [%s]", identity.String())
	}
	return auditInfo, nil
}

func (p *Provider) GetIdentityMetadata(identity view.Identity) ([]byte, error) {
	panic("implement me")
}

func (p *Provider) RegisterSigner(identity view.Identity, signer driver.Signer, verifier driver.Verifier) error {
	return view2.GetSigService(p.sp).RegisterSigner(identity, signer, verifier)
}

func (p *Provider) IsMe(identity view.Identity) bool {
	isMe := view2.GetSigService(p.sp).IsMe(identity)
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
	signer, err := view2.GetSigService(p.sp).GetSigner(identity)
	if err == nil {
		found = true
		return signer, nil
	}

	// give it a second chance
	ro, err2 := UnmarshallRawOwner(identity)
	if err2 != nil {
		found = false
		return nil, errors.Wrapf(err, "failed to unmarshal raw owner for identity [%s]", identity.String())
	}

	signer, err = view2.GetSigService(p.sp).GetSigner(ro.Identity)
	if err == nil {
		found = true
		return signer, nil
	}

	// give it a third chance
	// deserializer using the provider's deserializers
	for _, d := range p.deserializers {
		signer, err = d.DeserializeSigner(ro.Identity)
		if err == nil {
			found = true
			return signer, nil
		}
	}

	return nil, errors.Errorf("failed to get signer for identity [%s], it is neither register nor deserialazable", identity.String())
}

func (p *Provider) RegisterAuditInfo(id view.Identity, auditInfo []byte) error {
	if err := view2.GetSigService(p.sp).RegisterAuditInfo(id, auditInfo); err != nil {
		return err
	}
	return nil
}

func (p *Provider) GetEnrollmentID(auditInfo []byte) (string, error) {
	return p.enrollmentIDUnmarshaler.GetEnrollmentID(auditInfo)
}

func (p *Provider) RegisterOwnerWallet(id string, typ string, path string) error {
	return p.mappers[driver.OwnerRole].RegisterIdentity(id, typ, path)
}

func (p *Provider) RegisterIssuerWallet(id string, typ string, path string) error {
	return p.mappers[driver.IssuerRole].RegisterIdentity(id, typ, path)
}

func (p *Provider) AddDeserializer(d Deserializer) {
	p.deserializers = append(p.deserializers, d)
}

// Bind binds id to the passed identity long term identity. The same signer, verifier, and audit of the long term
// identity is associated to id.
func (p *Provider) Bind(id view.Identity, to view.Identity) error {
	sigService := view2.GetSigService(p.sp)

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

	if err := view2.GetEndpointService(p.sp).Bind(to, id); err != nil {
		return err
	}
	return nil
}

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
	id, ai, err := i.IdentityInfo.Get()
	if err != nil {
		return nil, nil, err
	}
	if err := i.Provider.RegisterAuditInfo(id, ai); err != nil {
		return nil, nil, err
	}
	if err := view2.GetEndpointService(i.Provider.sp).Bind(view2.GetIdentityProvider(i.Provider.sp).DefaultIdentity(), id); err != nil {
		return nil, nil, err
	}
	return id, ai, nil
}
