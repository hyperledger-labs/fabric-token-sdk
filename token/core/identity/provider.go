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
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

var logger = flogging.MustGetLogger("token-sdk.driver.identity")

type Mappers map[driver.IdentityUsage]mapper

func NewMappers() Mappers {
	return make(Mappers)
}

func (m Mappers) Put(usage driver.IdentityUsage, mapper mapper) {
	m[usage] = mapper
}

func (m Mappers) SetIssuerRole(mapper mapper) {
	m[driver.IssuerRole] = mapper
}

func (m Mappers) SetAuditorRole(mapper mapper) {
	m[driver.AuditorRole] = mapper
}

func (m Mappers) SetOwnerRole(mapper mapper) {
	m[driver.OwnerRole] = mapper
}

type GetFunc func() (view.Identity, []byte, error)

type Deserializer interface {
	DeserializeSigner(raw []byte) (driver.Signer, error)
}

type EnrollmentIDUnmarshaler interface {
	GetEnrollmentID(auditInfo []byte) (string, error)
}

type mapper interface {
	Info(id string) (string, string, GetFunc)
	Map(v interface{}) (view.Identity, string)
	RegisterIdentity(id string, typ string, path string) error
}

type Provider struct {
	sp view2.ServiceProvider

	mappers                 map[driver.IdentityUsage]mapper
	deserializers           []Deserializer
	enrollmentIDUnmarshaler EnrollmentIDUnmarshaler
	isMeCacheLock           sync.RWMutex
	isMeCache               map[string]bool
}

func NewProvider(sp view2.ServiceProvider, enrollmentIDUnmarshaler EnrollmentIDUnmarshaler, mappers map[driver.IdentityUsage]mapper) *Provider {
	return &Provider{
		sp:                      sp,
		mappers:                 mappers,
		deserializers:           []Deserializer{},
		enrollmentIDUnmarshaler: enrollmentIDUnmarshaler,
		isMeCache:               make(map[string]bool),
	}
}

func (i *Provider) GetIdentityInfo(usage driver.IdentityUsage, id string) *driver.IdentityInfo {
	mapper, ok := i.mappers[usage]
	if !ok {
		panic(fmt.Sprintf("mapper not found for usage [%d]", usage))
	}
	id, eid, getIdentity := mapper.Info(id)
	if getIdentity == nil {
		return nil
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("info for [%v] is [%s,%s]", id, id, eid)
	}
	return &driver.IdentityInfo{
		ID:           id,
		EnrollmentID: eid,
		GetIdentity: func() (view.Identity, error) {
			id, ai, err := getIdentity()
			if err != nil {
				return nil, err
			}
			if err := i.RegisterAuditInfo(id, ai); err != nil {
				return nil, err
			}
			if err := view2.GetEndpointService(i.sp).Bind(view2.GetIdentityProvider(i.sp).DefaultIdentity(), id); err != nil {
				return nil, err
			}
			return id, nil
		},
	}
}

func (i *Provider) LookupIdentifier(usage driver.IdentityUsage, v interface{}) (view.Identity, string) {
	mapper, ok := i.mappers[usage]
	if !ok {
		panic(fmt.Sprintf("mapper not found for usage [%d]", usage))
	}
	id, label := mapper.Map(v)
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("identifier for [%v] is [%s,%s]", v, id, label)
	}
	return id, label
}

func (i *Provider) GetAuditInfo(identity view.Identity) ([]byte, error) {
	auditInfo, err := view2.GetSigService(i.sp).GetAuditInfo(identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for recipient identity [%s]", identity.String())
	}
	return auditInfo, nil
}

func (i *Provider) GetIdentityMetadata(identity view.Identity) ([]byte, error) {
	panic("implement me")
}

func (i *Provider) RegisterSigner(identity view.Identity, signer driver.Signer, verifier driver.Verifier) error {
	return view2.GetSigService(i.sp).RegisterSigner(identity, signer, verifier)
}

func (i *Provider) IsMe(identity view.Identity) bool {
	isMe := view2.GetSigService(i.sp).IsMe(identity)
	if !isMe {
		// check cache
		i.isMeCacheLock.RLock()
		isMe, ok := i.isMeCache[identity.String()]
		i.isMeCacheLock.RUnlock()
		if ok {
			return isMe
		}

		// try to get the signer
		signer, err := i.GetSigner(identity)
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

func (i *Provider) RegisterRecipientIdentity(id view.Identity) error {
	i.isMeCacheLock.Lock()
	i.isMeCache[id.String()] = false
	i.isMeCacheLock.Unlock()
	return nil
}

func (i *Provider) GetSigner(identity view.Identity) (driver.Signer, error) {
	found := false
	defer func() {
		i.isMeCacheLock.Lock()
		i.isMeCache[identity.String()] = found
		i.isMeCacheLock.Unlock()
	}()
	signer, err := view2.GetSigService(i.sp).GetSigner(identity)
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

	signer, err = view2.GetSigService(i.sp).GetSigner(ro.Identity)
	if err == nil {
		found = true
		return signer, nil
	}

	// give it a third chance
	// deserializer using the provider's deserializers
	for _, d := range i.deserializers {
		signer, err = d.DeserializeSigner(ro.Identity)
		if err == nil {
			found = true
			return signer, nil
		}
	}

	return nil, errors.Errorf("failed to get signer for identity [%s], it is neither register nor deserialazable", identity.String())
}

func (i *Provider) RegisterAuditInfo(id view.Identity, auditInfo []byte) error {
	if err := view2.GetSigService(i.sp).RegisterAuditInfo(id, auditInfo); err != nil {
		return err
	}
	return nil
}

func (i *Provider) GetEnrollmentID(auditInfo []byte) (string, error) {
	return i.enrollmentIDUnmarshaler.GetEnrollmentID(auditInfo)
}

func (i *Provider) RegisterOwnerWallet(id string, typ string, path string) error {
	return i.mappers[driver.OwnerRole].RegisterIdentity(id, typ, path)
}

func (i *Provider) AddDeserializer(d Deserializer) {
	i.deserializers = append(i.deserializers, d)
}

// Bind binds id to the passed identity long term identity. The same signer, verifier, and audit of the long term
// identity is associated to id.
func (i *Provider) Bind(id view.Identity, to view.Identity) error {
	sigService := view2.GetSigService(i.sp)

	setSV := true
	signer, err := i.GetSigner(to)
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

	if err := view2.GetEndpointService(i.sp).Bind(to, id); err != nil {
		return err
	}
	return nil
}
