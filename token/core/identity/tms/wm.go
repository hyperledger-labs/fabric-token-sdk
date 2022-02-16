/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tms

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
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/x509"
	api2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	sig2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/core/sig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger/fabric/msp"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
)

var logger = flogging.MustGetLogger("token-sdk.driver.identity.tms")

const (
	IdemixMSP       = "idemix"
	IdemixMSPFolder = "idemix-folder"
	BccspMSP        = "bccsp"
	BccspMSPFolder  = "bccsp-folder"
)

type DeserializerManager interface {
	AddDeserializer(deserializer sig2.Deserializer)
}

type IdentityInfo struct {
	ID           string
	EnrollmentID string
	GetIdentity  GetIdentityFunc
}

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

type WalletManager struct {
	sp              view2.ServiceProvider
	cm              driver.ConfigManager
	empty           bool
	defaultIdentity view.Identity
	signerService   SignerService
	binderService   BinderService

	owners   *LocalMembership
	issuers  *LocalMembership
	auditors *LocalMembership
}

func NewWalletManager(sp view2.ServiceProvider, cm driver.ConfigManager, defaultIdentity view.Identity, signerService SignerService, binderService BinderService) *WalletManager {
	return &WalletManager{
		sp:              sp,
		cm:              cm,
		empty:           true,
		defaultIdentity: defaultIdentity,
		signerService:   signerService,
		binderService:   binderService,
	}
}

func (l *WalletManager) Load() error {
	logger.Debugf("load wallets...")
	defer logger.Debugf("load wallets...done")
	if l.cm.TMS().Wallets == nil {
		logger.Warnf("No wallets found in config")
		return nil
	}
	empty := true
	owners := NewLocalMembership(l.sp, l.cm, l.defaultIdentity, l.signerService, l.binderService)
	if len(l.cm.TMS().Wallets.Owners) != 0 {
		empty = false
		if err := owners.Load(l.cm.TMS().Wallets.Owners); err != nil {
			return errors.WithMessage(err, "failed to load owners")
		}
	}

	issuers := NewLocalMembership(l.sp, l.cm, l.defaultIdentity, l.signerService, l.binderService)
	if len(l.cm.TMS().Wallets.Issuers) != 0 {
		empty = false
		if err := issuers.Load(l.cm.TMS().Wallets.Issuers); err != nil {
			return errors.WithMessage(err, "failed to load issuers")
		}
	}

	auditors := NewLocalMembership(l.sp, l.cm, l.defaultIdentity, l.signerService, l.binderService)
	if len(l.cm.TMS().Wallets.Auditors) != 0 {
		empty = false
		if err := auditors.Load(l.cm.TMS().Wallets.Auditors); err != nil {
			return errors.WithMessage(err, "failed to load auditors")
		}
	}

	l.owners = owners
	l.issuers = issuers
	l.auditors = auditors
	l.empty = empty
	return nil
}

func (l *WalletManager) IsEmpty() bool {
	return l.empty
}

func (l *WalletManager) Owners() *LocalMembership {
	return l.owners
}

func (l *WalletManager) Issuers() *LocalMembership {
	return l.issuers
}

func (l *WalletManager) Auditors() *LocalMembership {
	return l.auditors
}

type LocalMembership struct {
	sp                 view2.ServiceProvider
	cm                 driver.ConfigManager
	defaultFSCIdentity view.Identity
	signerService      SignerService
	binderService      BinderService

	resolversMutex           sync.RWMutex
	resolvers                []*Resolver
	resolversByName          map[string]*Resolver
	resolversByEnrollmentID  map[string]*Resolver
	resolversByTypeAndName   map[string]*Resolver
	bccspResolversByIdentity map[string]*Resolver
}

func NewLocalMembership(sp view2.ServiceProvider, cm driver.ConfigManager, defaultFSCIdentity view.Identity, signerService SignerService, binderService BinderService) *LocalMembership {
	return &LocalMembership{
		sp:                       sp,
		cm:                       cm,
		defaultFSCIdentity:       defaultFSCIdentity,
		signerService:            signerService,
		binderService:            binderService,
		resolversByTypeAndName:   map[string]*Resolver{},
		bccspResolversByIdentity: map[string]*Resolver{},
		resolversByEnrollmentID:  map[string]*Resolver{},
		resolversByName:          map[string]*Resolver{},
	}
}

func (lm *LocalMembership) Load(identities []*driver.Identity) error {
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

func (lm *LocalMembership) GetAnonymousIdentifier(label string) (string, error) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get anonymous identity info by label [%s]", label)
	}
	r := lm.getAnonymousResolver(label)
	if r == nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("anonymous identity info not found for label [%s][%v]", label, lm.resolversByTypeAndName)
		}
		return "", errors.New("not found")
	}
	if r.Default {
		return "idemix", nil
	}
	return r.Name, nil
}

func (lm *LocalMembership) GetAnonymousIdentity(label string, auditInfo []byte) (string, string, network.GetFunc, error) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get anonymous identity info by label [%s]", label)
	}
	r := lm.getAnonymousResolver(label)
	if r == nil {
		return "", "", nil, errors.Errorf("anonymous identity info not found for label [%s][%v]", label, lm.resolversByTypeAndName)
	}
	return r.Name, r.EnrollmentID,
		func() (view.Identity, []byte, error) {
			return r.GetIdentity(&driver2.IdentityOptions{
				EIDExtension: true,
				AuditInfo:    auditInfo,
			})
		},
		nil
}

func (lm *LocalMembership) GetLongTermIdentifier(id view.Identity) (string, error) {
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
		return "default", nil
	}
	return r.Name, nil
}

func (lm *LocalMembership) GetLongTermIdentity(label string) (string, string, view.Identity, error) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get identity info by label [%s]", label)
	}
	r := lm.getLongTermResolver(label)
	if r == nil {
		return "", "", nil, errors.Errorf("identity info not found for label [%s][%v]", label, lm.resolversByTypeAndName)
	}
	id, _, err := r.GetIdentity(nil)
	if err != nil {
		return "", "", nil, err
	}
	return r.Name, r.EnrollmentID, id, nil
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

	switch typeAndMspID[0] {
	case IdemixMSP:
		conf, err := msp.GetLocalMspConfigWithType(
			lm.cm.TranslatePath(path),
			nil,
			typeAndMspID[1],
			IdemixMSP,
		)
		if err != nil {
			return errors.Wrapf(err, "failed reading idemix msp configuration from [%s]", lm.cm.TranslatePath(path))
		}
		curveID, err := StringToCurveID(typeAndMspID[2])
		if err != nil {
			return errors.Errorf("invalid curve ID '%s'", typ)
		}
		provider, err := idemix2.NewAnyProviderWithCurve(conf, lm.sp, curveID)
		if err != nil {
			return errors.Wrapf(err, "failed instantiating idemix msp provider from [%s]", lm.cm.TranslatePath(path))
		}
		dm.AddDeserializer(provider)
		lm.addResolver(
			id,
			IdemixMSP,
			provider.EnrollmentID(),
			setDefault,
			NewIdentityCache(provider.Identity, 500).Identity,
		)
	case BccspMSP:
		provider, err := x509.NewProvider(lm.cm.TranslatePath(path), typeAndMspID[1], lm.signerService)
		if err != nil {
			return errors.Wrapf(err, "failed instantiating x509 msp provider from [%s]", lm.cm.TranslatePath(path))
		}
		dm.AddDeserializer(provider)
		lm.addResolver(id, BccspMSP, provider.EnrollmentID(), setDefault, provider.Identity)
	case IdemixMSPFolder:
		entries, err := ioutil.ReadDir(lm.cm.TranslatePath(path))
		if err != nil {
			logger.Warnf("failed reading from [%s]: [%s]", lm.cm.TranslatePath(path), err)
			return nil
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			id := entry.Name()
			conf, err := msp.GetLocalMspConfigWithType(
				filepath.Join(lm.cm.TranslatePath(path), id),
				nil,
				typeAndMspID[1],
				IdemixMSP,
			)
			if err != nil {
				logger.Warnf("failed reading idemix msp configuration from [%s]: [%s]", filepath.Join(lm.cm.TranslatePath(path), id), err)
				continue
			}
			curveID, err := StringToCurveID(typeAndMspID[2])
			if err != nil {
				return errors.Errorf("invalid curve ID '%s'", typ)
			}
			provider, err := idemix2.NewAnyProviderWithCurve(conf, lm.sp, curveID)
			if err != nil {
				logger.Warnf("failed instantiating idemix msp configuration from [%s]: [%s]", filepath.Join(lm.cm.TranslatePath(path), id), err)
				continue
			}
			logger.Debugf("Adding resolver [%s:%s]", id, provider.EnrollmentID())
			dm.AddDeserializer(provider)
			lm.addResolver(
				id,
				IdemixMSP,
				provider.EnrollmentID(),
				false,
				NewIdentityCache(provider.Identity, 500).Identity,
			)
		}
	case BccspMSPFolder:
		entries, err := ioutil.ReadDir(lm.cm.TranslatePath(path))
		if err != nil {
			logger.Warnf("failed reading from [%s]: [%s]", lm.cm.TranslatePath(path), err)
			return nil
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			id := entry.Name()

			// Try without "msp"
			provider, err := x509.NewProvider(
				filepath.Join(lm.cm.TranslatePath(path), id),
				typeAndMspID[1],
				lm.signerService,
			)
			if err != nil {
				logger.Debugf("failed reading bccsp msp configuration from [%s]: [%s]", filepath.Join(lm.cm.TranslatePath(path), id), err)
				// Try with "msp"
				provider, err = x509.NewProvider(
					filepath.Join(lm.cm.TranslatePath(path), id, "msp"),
					typeAndMspID[1],
					lm.signerService,
				)
				if err != nil {
					logger.Warnf("failed reading bccsp msp configuration from [%s and %s]: [%s]",
						filepath.Join(lm.cm.TranslatePath(path),
							filepath.Join(lm.cm.TranslatePath(path), id, "msp")), err,
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

func (lm *LocalMembership) addResolver(Name string, Type string, EnrollmentID string, defaultID bool, IdentityGetter GetIdentityFunc) {
	lm.resolversMutex.RLock()
	defer lm.resolversMutex.RUnlock()

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

func (lm *LocalMembership) getAnonymousResolver(label string) *Resolver {
	lm.resolversMutex.RLock()
	defer lm.resolversMutex.RUnlock()

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get anonymous identity info by label [%s]", label)
	}
	r, ok := lm.resolversByTypeAndName[IdemixMSP+label]
	if ok {
		return r
	}

	if label == "idemix" {
		for _, resolver := range lm.resolvers {
			if resolver.Type == IdemixMSP && resolver.Default {
				return resolver
			}
		}
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("anonymous identity info not found for label [%s][%v]", label, lm.resolversByTypeAndName)
	}
	return nil
}

func (lm *LocalMembership) getLongTermResolver(label string) *Resolver {
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

	if label == "default" {
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
