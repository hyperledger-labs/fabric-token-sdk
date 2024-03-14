/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/deserializer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/sig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.driver.fabtoken")

type Driver struct {
}

func (d *Driver) PublicParametersFromBytes(params []byte) (driver.PublicParameters, error) {
	pp, err := fabtoken.NewPublicParamsFromBytes(params, fabtoken.PublicParameters)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal public parameters")
	}
	return pp, nil
}

func (d *Driver) NewTokenService(sp driver.ServiceProvider, networkID string, channel string, namespace string, publicParams []byte) (driver.TokenManagerService, error) {
	if len(publicParams) == 0 {
		return nil, errors.Errorf("empty public parameters")
	}
	n := network.GetInstance(sp, networkID, channel)
	if n == nil {
		return nil, errors.Errorf("network [%s] does not exists", networkID)
	}
	v, err := n.Vault(namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "vault [%s:%s] does not exists", networkID, namespace)
	}
	qe := v.QueryEngine()
	networkLocalMembership := n.LocalMembership()

	cs := view.GetConfigService(sp)
	tmsConfig, err := config.NewTokenSDK(cs).GetTMS(networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create config manager")
	}

	// Prepare roles
	storageProvider, err := identity.GetStorageProvider(sp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	fscIdentity := view.GetIdentityProvider(sp).DefaultIdentity()
	roles := identity.NewRoles()
	deserializerManager := deserializer.NewMultiplexDeserializer()
	tmsID := token.TMSID{
		Network:   networkID,
		Channel:   channel,
		Namespace: namespace,
	}
	identityDB, err := storageProvider.OpenIdentityDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open identity db for tms [%s]", tmsID)
	}
	sigService := sig.NewService(deserializerManager, identityDB)
	ip := identity.NewProvider(identityDB, sigService, view.GetEndpointService(sp), NewEIDRHDeserializer(), deserializerManager)
	roleFactory := msp.NewRoleFactory(
		tmsID,
		config2.NewIdentityConfig(cs, tmsConfig), // config
		fscIdentity,                              // FSC identity
		networkLocalMembership.DefaultIdentity(), // network default identity
		ip,
		sigService,                  // sig service
		view.GetEndpointService(sp), // endpoint service
		storageProvider,
		deserializerManager,
		false,
	)
	role, err := roleFactory.NewWrappedX509(driver.OwnerRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create owner role")
	}
	roles.Register(driver.OwnerRole, role)
	role, err = roleFactory.NewX509(driver.IssuerRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create issuer role")
	}
	roles.Register(driver.IssuerRole, role)
	role, err = roleFactory.NewX509(driver.AuditorRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create auditor role")
	}
	roles.Register(driver.AuditorRole, role)
	role, err = roleFactory.NewX509(driver.CertifierRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create certifier role")
	}
	roles.Register(driver.CertifierRole, role)

	// Instantiate the token service
	walletDB, err := storageProvider.OpenWalletDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	publicParamsManager, err := common.NewPublicParamsManager[*fabtoken.PublicParams](
		&PublicParamsDeserializer{},
		fabtoken.PublicParameters,
		publicParams,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initiliaze public params manager")
	}
	deserializer := NewDeserializer()
	ws := common.NewWalletService(
		logger,
		ip,
		deserializer,
		fabtoken.NewWalletFactory(ip, qe),
		identity.NewWalletRegistry(roles[driver.OwnerRole], walletDB),
		identity.NewWalletRegistry(roles[driver.IssuerRole], walletDB),
		identity.NewWalletRegistry(roles[driver.AuditorRole], walletDB),
		nil,
	)
	service, err := fabtoken.NewService(
		ws,
		publicParamsManager,
		common.NewVaultTokenLoader(qe),
		ip,
		NewDeserializer(),
		tmsConfig,
	)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create token service")
	}
	if err := roles.Reload(publicParamsManager.PublicParameters()); err != nil {
		return nil, errors.WithMessage(err, "failed to update public parameters")
	}
	return service, nil
}

func (d *Driver) NewValidator(params driver.PublicParameters) (driver.Validator, error) {
	pp, ok := params.(*fabtoken.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}
	return fabtoken.NewValidator(pp, NewDeserializer()), nil
}

func (d *Driver) NewPublicParametersManager(params driver.PublicParameters) (driver.PublicParamsManager, error) {
	pp, ok := params.(*fabtoken.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}
	return common.NewPublicParamsManagerFromParams[*fabtoken.PublicParams](pp)
}

func (d *Driver) NewWalletService(sp driver.ServiceProvider, networkID string, channel string, namespace string, params driver.PublicParameters) (driver.WalletService, error) {
	cs := view.GetConfigService(sp)
	tmsConfig, err := config.NewTokenSDK(cs).GetTMS(networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create config manager")
	}

	// Prepare roles
	storageProvider, err := identity.GetStorageProvider(sp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	roles := identity.NewRoles()
	deserializerManager := deserializer.NewMultiplexDeserializer()
	tmsID := token.TMSID{
		Network:   networkID,
		Channel:   channel,
		Namespace: namespace,
	}
	identityDB, err := storageProvider.OpenIdentityDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open identity db for tms [%s]", tmsID)
	}
	sigService := sig.NewService(deserializerManager, identityDB)
	ip := identity.NewProvider(identityDB, sigService, nil, NewEIDRHDeserializer(), deserializerManager)
	roleFactory := msp.NewRoleFactory(
		tmsID,
		config2.NewIdentityConfig(cs, tmsConfig), // config
		nil,                                      // FSC identity
		nil,                                      // network default identity
		ip,
		sigService, // signer service
		nil,        // endpoint service
		storageProvider,
		deserializerManager,
		true,
	)
	role, err := roleFactory.NewX509IgnoreRemote(driver.OwnerRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create owner role")
	}
	roles.Register(driver.OwnerRole, role)
	role, err = roleFactory.NewX509(driver.IssuerRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create issuer role")
	}
	roles.Register(driver.IssuerRole, role)
	role, err = roleFactory.NewX509(driver.AuditorRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create auditor role")
	}
	roles.Register(driver.AuditorRole, role)
	role, err = roleFactory.NewX509(driver.CertifierRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create certifier role")
	}
	roles.Register(driver.CertifierRole, role)

	// Instantiate the token service
	// wallet service
	walletDB, err := storageProvider.OpenWalletDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	deserializer := NewDeserializer()
	ws := common.NewWalletService(
		logger,
		ip,
		deserializer,
		fabtoken.NewWalletFactory(ip, nil),
		identity.NewWalletRegistry(roles[driver.OwnerRole], walletDB),
		identity.NewWalletRegistry(roles[driver.IssuerRole], walletDB),
		identity.NewWalletRegistry(roles[driver.AuditorRole], walletDB),
		nil,
	)

	return ws, nil
}

func init() {
	core.Register(fabtoken.PublicParameters, &Driver{})
}
