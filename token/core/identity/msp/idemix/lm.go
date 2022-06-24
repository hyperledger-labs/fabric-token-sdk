/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	math3 "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	idemix2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/idemix"
	api2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	sig2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/core/sig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	"github.com/hyperledger/fabric/msp"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

const (
	DefaultLabel = "idemix"

	MSP       = "idemix"
	MSPFolder = "idemix-folder"
)

type GetIdentityFunc func(opts *driver2.IdentityOptions) (view.Identity, []byte, error)

type Resolver struct {
	Name         string `yaml:"name,omitempty"`
	Type         string `yaml:"type,omitempty"`
	EnrollmentID string
	GetIdentity  GetIdentityFunc
	Default      bool
}

type SignerService interface {
	RegisterSigner(identity view.Identity, signer api2.Signer, verifier api2.Verifier) error
}

type BinderService interface {
	Bind(longTerm view.Identity, ephemeral view.Identity) error
}

type EnrollmentService interface {
	GetEnrollmentID(auditInfo []byte) (string, error)
}

type DeserializerManager interface {
	AddDeserializer(deserializer sig2.Deserializer)
}

type LocalMembership struct {
	sp                 view2.ServiceProvider
	configManager      config.Manager
	defaultFSCIdentity view.Identity
	signerService      SignerService
	binderService      BinderService
	mspID              string

	resolversMutex          sync.RWMutex
	resolvers               []*Resolver
	resolversByName         map[string]*Resolver
	resolversByEnrollmentID map[string]*Resolver
	resolversByTypeAndName  map[string]*Resolver
}

func NewLocalMembership(
	sp view2.ServiceProvider,
	configManager config.Manager,
	defaultFSCIdentity view.Identity,
	signerService SignerService,
	binderService BinderService,
	mspID string,
) *LocalMembership {
	return &LocalMembership{
		sp:                      sp,
		configManager:           configManager,
		defaultFSCIdentity:      defaultFSCIdentity,
		signerService:           signerService,
		binderService:           binderService,
		mspID:                   mspID,
		resolversByTypeAndName:  map[string]*Resolver{},
		resolversByEnrollmentID: map[string]*Resolver{},
		resolversByName:         map[string]*Resolver{},
	}
}

func (lm *LocalMembership) Load(identities []*config.Identity) error {
	logger.Debugf("loadWallets: %+v", identities)

	type Provider interface {
		EnrollmentID() string
		Identity(opts *fabric.IdentityOption) (view.Identity, []byte, error)
		DeserializeVerifier(raw []byte) (driver.Verifier, error)
		DeserializeSigner(raw []byte) (driver.Signer, error)
		Info(raw []byte, auditInfo []byte) (string, error)
	}

	for _, identityConfig := range identities {
		logger.Debugf("loadWallet: %+v", identityConfig)
		if err := lm.registerIdentity(identityConfig.ID, identityConfig.Type, identityConfig.Path, identityConfig.Default); err != nil {
			return errors.WithMessage(err, "failed to load identity")
		}
	}
	return nil
}

func (lm *LocalMembership) DefaultIdentity() view.Identity {
	return lm.defaultFSCIdentity
}

func (lm *LocalMembership) IsMe(id view.Identity) bool {
	return view2.GetSigService(lm.sp).IsMe(id)
}

func (lm *LocalMembership) GetIdentifier(id view.Identity) (string, error) {
	label := string(id)
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get anonymous identity info by label [%s]", label)
	}
	r := lm.getResolver(label)
	if r == nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("anonymous identity info not found for label [%s][%v]", label, lm.resolversByTypeAndName)
		}
		return "", errors.New("not found")
	}
	if r.Default {
		// TODO: do we still need this?
		return DefaultLabel, nil
	}
	return r.Name, nil
}

func (lm *LocalMembership) GetIdentityInfo(label string, auditInfo []byte) (driver.IdentityInfo, error) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get anonymous identity info by label [%s]", label)
	}
	r := lm.getResolver(label)
	if r == nil {
		return nil, errors.Errorf("anonymous identity info not found for label [%s][%v]", label, lm.resolversByTypeAndName)
	}

	return &Info{
		id:  r.Name,
		eid: r.EnrollmentID,
		getIdentity: func() (view.Identity, []byte, error) {
			return r.GetIdentity(&driver2.IdentityOptions{
				EIDExtension: true,
				AuditInfo:    auditInfo,
			})
		},
	}, nil
}

func (lm *LocalMembership) RegisterIdentity(id string, typ string, path string) error {
	return lm.registerIdentity(id, typ, path, false)
}

func (lm *LocalMembership) registerIdentity(id string, typ string, path string, setDefault bool) error {
	dm := lm.deserializerManager()

	// split type in type and msp id
	typeAndMspID := strings.Split(typ, ":")
	if len(typeAndMspID) < 2 {
		return errors.Errorf("invalid identity type '%s'", typ)
	}

	translatedPath := lm.configManager.TranslatePath(path)
	switch typeAndMspID[0] {
	case MSP:
		conf, err := msp.GetLocalMspConfigWithType(translatedPath, nil, lm.mspID, MSP)
		if err != nil {
			return errors.Wrapf(err, "failed reading idemix msp configuration from [%s]", translatedPath)
		}
		curveID, err := StringToCurveID(typeAndMspID[2])
		if err != nil {
			return errors.Errorf("invalid curve ID '%s'", typ)
		}
		provider, err := idemix2.NewAnyProviderWithCurve(conf, lm.sp, curveID)
		if err != nil {
			return errors.Wrapf(err, "failed instantiating idemix msp provider from [%s]", translatedPath)
		}
		dm.AddDeserializer(provider)
		lm.addResolver(
			id,
			MSP,
			provider.EnrollmentID(),
			setDefault,
			NewIdentityCache(provider.Identity, DefaultCacheSize).Identity,
		)
		logger.Debugf("added %s resolver for id %s with cache of size %d", MSP, id+"@"+provider.EnrollmentID(), DefaultCacheSize)
	case MSPFolder:
		entries, err := ioutil.ReadDir(translatedPath)
		if err != nil {
			logger.Warnf("failed reading from [%s]: [%s]", translatedPath, err)
			return nil
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			id := entry.Name()
			conf, err := msp.GetLocalMspConfigWithType(filepath.Join(translatedPath, id), nil, lm.mspID, MSP)
			if err != nil {
				logger.Warnf("failed reading idemix msp configuration from [%s]: [%s]", filepath.Join(translatedPath, id), err)
				continue
			}
			curveID, err := StringToCurveID(typeAndMspID[2])
			if err != nil {
				return errors.Errorf("invalid curve ID '%s'", typ)
			}
			provider, err := idemix2.NewAnyProviderWithCurve(conf, lm.sp, curveID)
			if err != nil {
				logger.Warnf("failed instantiating idemix msp configuration from [%s]: [%s]", filepath.Join(translatedPath, id), err)
				continue
			}
			logger.Debugf("Adding resolver [%s:%s]", id, provider.EnrollmentID())
			dm.AddDeserializer(provider)
			lm.addResolver(
				id,
				MSP,
				provider.EnrollmentID(),
				false,
				NewIdentityCache(provider.Identity, DefaultCacheSize).Identity,
			)
			logger.Debugf("added %s resolver for id %s with cache of size %d", MSP, id+"@"+provider.EnrollmentID(), DefaultCacheSize)
		}
	default:
		logger.Warnf("msp type [%s] not recognized, skipping", typeAndMspID[0])
	}

	return nil
}

func (lm *LocalMembership) addResolver(Name string, Type string, EnrollmentID string, defaultID bool, IdentityGetter GetIdentityFunc) {
	lm.resolversMutex.Lock()
	defer lm.resolversMutex.Unlock()

	resolver := &Resolver{
		Name:         Name,
		Default:      defaultID,
		Type:         Type,
		EnrollmentID: EnrollmentID,
		GetIdentity:  IdentityGetter,
	}
	lm.resolversByTypeAndName[Type+Name] = resolver
	lm.resolversByName[Name] = resolver
	if len(EnrollmentID) != 0 {
		lm.resolversByEnrollmentID[EnrollmentID] = resolver
	}
	lm.resolvers = append(lm.resolvers, resolver)
}

func (lm *LocalMembership) getResolver(label string) *Resolver {
	lm.resolversMutex.RLock()
	defer lm.resolversMutex.RUnlock()

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get anonymous identity info by label [%s]", label)
	}
	r, ok := lm.resolversByTypeAndName[MSP+label]
	if ok {
		return r
	}

	if label == DefaultLabel {
		for _, resolver := range lm.resolvers {
			if resolver.Type == MSP && resolver.Default {
				return resolver
			}
		}
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("anonymous identity info not found for label [%s][%v]", label, lm.resolversByTypeAndName)
	}
	return nil
}

func (lm *LocalMembership) deserializerManager() DeserializerManager {
	dm, err := lm.sp.GetService(reflect.TypeOf((*DeserializerManager)(nil)))
	if err != nil {
		panic(fmt.Sprintf("failed looking up deserializer manager [%s]", err))
	}
	return dm.(DeserializerManager)
}

func StringToCurveID(id string) (math3.CurveID, error) {
	switch id {
	case "BN254":
		return math3.BN254, nil
	case "FP256BN_AMCL":
		return math3.FP256BN_AMCL, nil
	case "FP256BN_AMCL_MIRACL":
		return math3.FP256BN_AMCL_MIRACL, nil
	default:
		return math3.CurveID(0), errors.Errorf("unknown curve [%s]", id)
	}
}

type Info struct {
	id          string
	eid         string
	getIdentity func() (view.Identity, []byte, error)
}

func (i *Info) ID() string {
	return i.id
}

func (i *Info) EnrollmentID() string {
	return i.eid
}

func (i *Info) Get() (view.Identity, []byte, error) {
	id, ai, err := i.getIdentity()
	if err != nil {
		return nil, nil, err
	}
	return id, ai, nil
}
