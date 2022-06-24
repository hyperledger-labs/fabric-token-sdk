/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	sig2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/core/sig"

	math3 "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/x509"
	api2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

const (
	DefaultLabel = "default"

	BccspMSP       = "bccsp"
	BccspMSPFolder = "bccsp-folder"
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

type LM struct {
	sp                 view2.ServiceProvider
	configManager      config.Manager
	defaultFSCIdentity view.Identity
	signerService      SignerService
	binderService      BinderService
	mspID              string

	resolversMutex           sync.RWMutex
	resolvers                []*Resolver
	resolversByName          map[string]*Resolver
	resolversByEnrollmentID  map[string]*Resolver
	resolversByTypeAndName   map[string]*Resolver
	bccspResolversByIdentity map[string]*Resolver
}

func NewLocalMembership(
	sp view2.ServiceProvider,
	configManager config.Manager,
	defaultFSCIdentity view.Identity,
	signerService SignerService,
	binderService BinderService,
	mspID string,
) *LM {
	return &LM{
		sp:                       sp,
		configManager:            configManager,
		defaultFSCIdentity:       defaultFSCIdentity,
		signerService:            signerService,
		binderService:            binderService,
		mspID:                    mspID,
		resolversByTypeAndName:   map[string]*Resolver{},
		bccspResolversByIdentity: map[string]*Resolver{},
		resolversByEnrollmentID:  map[string]*Resolver{},
		resolversByName:          map[string]*Resolver{},
	}
}

func (lm *LM) Load(identities []*config.Identity) error {
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

func (lm *LM) DefaultIdentity() view.Identity {
	return lm.defaultFSCIdentity
}

func (lm *LM) IsMe(id view.Identity) bool {
	return view2.GetSigService(lm.sp).IsMe(id)
}

func (lm *LM) GetIdentifier(id view.Identity) (string, error) {
	label := id.String()
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get identity info by label [%s]", label)
	}
	r := lm.getLongTermResolver(label)
	if r == nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("identity info not found for label [%s][%v]", label, lm.resolversByTypeAndName)
		}
		return "", errors.New("not found")
	}
	if r.Default {
		return DefaultLabel, nil
	}
	return r.Name, nil
}

func (lm *LM) GetIdentityInfo(label string, auditInfo []byte) (driver.IdentityInfo, error) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get identity info by label [%s]", label)
	}
	r := lm.getLongTermResolver(label)
	if r == nil {
		return nil, errors.Errorf("identity info not found for label [%s][%v]", label, lm.resolversByTypeAndName)
	}

	return &Info{
		id:  r.Name,
		eid: r.EnrollmentID,
		getIdentity: func() (view.Identity, []byte, error) {
			return r.GetIdentity(nil)
		},
	}, nil
}

func (lm *LM) RegisterIdentity(id string, typ string, path string) error {
	return lm.registerIdentity(id, typ, path, false)
}

func (lm *LM) registerIdentity(id string, typ string, path string, setDefault bool) error {
	dm := lm.deserializerManager()

	// split type in type and msp id
	typeAndMspID := strings.Split(typ, ":")
	if len(typeAndMspID) < 2 {
		return errors.Errorf("invalid identity type '%s'", typ)
	}

	logger.Debugf("registerIdentity: [%s][%v]", typ, typeAndMspID)

	translatedPath := lm.configManager.TranslatePath(path)
	switch typeAndMspID[0] {
	case BccspMSP:
		provider, err := x509.NewProvider(translatedPath, lm.mspID, lm.signerService)
		if err != nil {
			return errors.Wrapf(err, "failed instantiating x509 msp provider from [%s]", translatedPath)
		}
		dm.AddDeserializer(provider)
		lm.addResolver(id, BccspMSP, provider.EnrollmentID(), setDefault, provider.Identity)
	case BccspMSPFolder:
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

			// Try without "msp"
			provider, err := x509.NewProvider(filepath.Join(translatedPath, id), lm.mspID, lm.signerService)
			if err != nil {
				logger.Debugf("failed reading bccsp msp configuration from [%s]: [%s]", filepath.Join(translatedPath, id), err)
				// Try with "msp"
				provider, err = x509.NewProvider(filepath.Join(translatedPath, id, "msp"), lm.mspID, lm.signerService)
				if err != nil {
					logger.Warnf("failed reading bccsp msp configuration from [%s and %s]: [%s]",
						filepath.Join(translatedPath), filepath.Join(translatedPath, id, "msp"), err,
					)
					continue
				}
			}

			logger.Debugf("Adding resolver [%s:%s]", id, provider.EnrollmentID())
			dm.AddDeserializer(provider)
			lm.addResolver(id, BccspMSP, provider.EnrollmentID(), false, provider.Identity)
		}
	default:
		logger.Warnf("msp type [%s] not recognized, skipping", typeAndMspID[0])
	}

	return nil
}

func (lm *LM) addResolver(Name string, Type string, EnrollmentID string, defaultID bool, IdentityGetter GetIdentityFunc) {
	logger.Debugf("Adding resolver [%s:%s:%s]", Name, Type, EnrollmentID)
	lm.resolversMutex.Lock()
	defer lm.resolversMutex.Unlock()

	if Type == BccspMSP && lm.binderService != nil {
		id, _, err := IdentityGetter(nil)
		if err != nil {
			panic(fmt.Sprintf("cannot get identity for [%s,%s,%s][%s]", Name, Type, EnrollmentID, err))
		}
		if err := lm.binderService.Bind(lm.defaultFSCIdentity, id); err != nil {
			panic(fmt.Sprintf("cannot bing identity for [%s,%s,%s][%s]", Name, Type, EnrollmentID, err))
		}
	}

	resolver := &Resolver{
		Name:         Name,
		Default:      defaultID,
		Type:         Type,
		EnrollmentID: EnrollmentID,
		GetIdentity:  IdentityGetter,
	}
	if Type == BccspMSP {
		id, _, err := IdentityGetter(nil)
		if err != nil {
			panic(fmt.Sprintf("cannot get identity for [%s,%s,%s][%s]", Name, Type, EnrollmentID, err))
		}
		lm.bccspResolversByIdentity[id.String()] = resolver
	}
	lm.resolversByTypeAndName[Type+Name] = resolver
	lm.resolversByName[Name] = resolver
	if len(EnrollmentID) != 0 {
		lm.resolversByEnrollmentID[EnrollmentID] = resolver
	}
	lm.resolvers = append(lm.resolvers, resolver)
}

func (lm *LM) getLongTermResolver(label string) *Resolver {
	lm.resolversMutex.RLock()
	defer lm.resolversMutex.RUnlock()

	r, ok := lm.resolversByName[label]
	if ok {
		return r
	}

	r, ok = lm.bccspResolversByIdentity[label]
	if ok {
		return r
	}

	r, ok = lm.resolversByTypeAndName[BccspMSP+label]
	if ok {
		return r
	}

	if label == DefaultLabel {
		for _, resolver := range lm.resolvers {
			if resolver.Type == BccspMSP && resolver.Default {
				return resolver
			}
		}
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("identity info not found for label [%s][%v]", label, lm.bccspResolversByIdentity)
	}
	return nil
}

func (lm *LM) deserializerManager() DeserializerManager {
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
