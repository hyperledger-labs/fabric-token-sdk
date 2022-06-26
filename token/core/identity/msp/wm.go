/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	api2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	sig2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/core/sig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.driver.identity.tms")

const (
	// OwnerMSPID is the default MSP ID for the owner wallet
	OwnerMSPID = "OwnerMSPID"
	// IssuerMSPID is the default MSP ID for the issuer wallet
	IssuerMSPID = "IssuerMSPID"
	// AuditorMSPID is the default MSP ID for the auditor wallet
	AuditorMSPID = "AuditorMSPID"
)

type IdentityType int

const (
	// LongTermIdentity is the type of the long term identity
	LongTermIdentity IdentityType = iota
	AnonymousIdentity
)

type LocalMembership interface {
	Load(owners []*config.Identity) error
	FSCNodeIdentity() view.Identity
	IsMe(id view.Identity) bool
	GetIdentityInfo(label string, auditInfo []byte) (driver.IdentityInfo, error)
	GetIdentifier(id view.Identity) (string, error)
	GetDefaultIdentifier() string
	RegisterIdentity(id string, path string) error
}

type DeserializerManager interface {
	AddDeserializer(deserializer sig2.Deserializer)
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

type WalletManager struct {
	sp                     view2.ServiceProvider
	networkID              string
	configManager          config.Manager
	fscIdentity            view.Identity
	networkDefaultIdentity view.Identity
	signerService          SignerService
	binderService          BinderService

	roles    map[int]IdentityType
	owners   LocalMembership
	issuers  LocalMembership
	auditors LocalMembership
}

func NewWalletManager(
	sp view2.ServiceProvider,
	networkID string,
	configManager config.Manager,
	fscIdentity view.Identity,
	networkDefaultIdentity view.Identity,
	signerService SignerService,
	binderService BinderService,
) *WalletManager {
	return &WalletManager{
		sp:                     sp,
		networkID:              networkID,
		configManager:          configManager,
		fscIdentity:            fscIdentity,
		networkDefaultIdentity: networkDefaultIdentity,
		signerService:          signerService,
		binderService:          binderService,
		roles:                  map[int]IdentityType{},
	}
}

// Load loads the wallets defined in the configuration
func (wm *WalletManager) Load() error {
	logger.Debugf("load wallets...")
	if len(wm.roles) < 3 {
		return errors.New("missing roles")
	}
	defer logger.Debugf("load wallets...done")

	tmsConfig := wm.configManager.TMS()
	if tmsConfig.Wallets == nil {
		logger.Warnf("No wallets found in config")
		tmsConfig.Wallets = &config.Wallets{}
	}

	owners, err := wm.newLocalManager(driver.OwnerRole, OwnerMSPID, wm.configManager.TMS().Wallets.Owners)
	if err != nil {
		return errors.Wrap(err, "failed to load owners")
	}

	issuers, err := wm.newLocalManager(driver.IssuerRole, IssuerMSPID, wm.configManager.TMS().Wallets.Issuers)
	if err != nil {
		return errors.Wrap(err, "failed to load issuers")
	}

	auditors, err := wm.newLocalManager(driver.AuditorRole, AuditorMSPID, wm.configManager.TMS().Wallets.Auditors)
	if err != nil {
		return errors.Wrap(err, "failed to load auditors")
	}

	wm.owners = owners
	wm.issuers = issuers
	wm.auditors = auditors
	return nil
}

// SetRoleIdentityType sets the identity type for the given role
func (wm *WalletManager) SetRoleIdentityType(role int, identityType IdentityType) {
	wm.roles[role] = identityType
}

func (wm *WalletManager) Mappers() (identity.Mappers, error) {
	mappers := identity.NewMappers()
	// issuers
	m, err := wm.newMapper(driver.IssuerRole, wm.issuers)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create issuer mapper")
	}
	mappers.SetIssuerRole(m)
	// auditors
	m, err = wm.newMapper(driver.AuditorRole, wm.auditors)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create auditor mapper")
	}
	mappers.SetAuditorRole(m)
	// owners
	m, err = wm.newMapper(driver.OwnerRole, wm.owners)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create owner mapper")
	}
	mappers.SetOwnerRole(m)

	return mappers, nil
}

// newMapper creates a new mapper for the given role
func (wm *WalletManager) newMapper(role int, lm LocalMembership) (identity.Mapper, error) {
	identityType, ok := wm.roles[role]
	if !ok {
		return nil, errors.Errorf("missing role %d", role)
	}

	var m identity.Mapper
	switch identityType {
	case AnonymousIdentity:
		m = idemix.NewMapper(wm.networkID, wm.fscIdentity, lm)
	case LongTermIdentity:
		m = x509.NewMapper(wm.networkID, wm.fscIdentity, lm)
	default:
		return nil, errors.Errorf("unknown identity type %d", identityType)
	}
	return m, nil
}

func (wm *WalletManager) newLocalManager(role int, mspID string, identities []*config.Identity) (LocalMembership, error) {
	identityType, ok := wm.roles[role]
	if !ok {
		return nil, errors.Errorf("missing role %d", role)
	}

	var lm LocalMembership
	switch identityType {
	case AnonymousIdentity:
		lm = idemix.NewLocalMembership(wm.sp, wm.configManager, wm.networkDefaultIdentity, wm.signerService, wm.binderService, mspID)
	case LongTermIdentity:
		lm = x509.NewLocalMembership(wm.sp, wm.configManager, wm.networkDefaultIdentity, wm.signerService, wm.binderService, mspID)
	}
	if err := lm.Load(identities); err != nil {
		return nil, errors.WithMessage(err, "failed to load owners")
	}
	return lm, nil
}
