/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/ppm"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/deserializer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/sig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/pkg/errors"
)

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

	// Prepare wallets
	storageProvider, err := identity.GetStorageProvider(sp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	fscIdentity := view.GetIdentityProvider(sp).DefaultIdentity()
	wallets := identity.NewWallets()
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
	mspWalletFactory := msp.NewWalletFactory(
		tmsID,
		config2.NewIdentityConfig(cs, tmsConfig), // config
		fscIdentity,                              // FSC identity
		networkLocalMembership.DefaultIdentity(), // network default identity
		sigService,                               // sig service
		view.GetEndpointService(sp),              // endpoint service
		storageProvider,
		deserializerManager,
		false,
	)
	wallet, err := mspWalletFactory.NewX509Wallet(driver.OwnerRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create owner wallet")
	}
	wallets.Put(driver.OwnerRole, wallet)
	wallet, err = mspWalletFactory.NewX509Wallet(driver.IssuerRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create issuer wallet")
	}
	wallets.Put(driver.IssuerRole, wallet)
	wallet, err = mspWalletFactory.NewX509Wallet(driver.AuditorRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create auditor wallet")
	}
	wallets.Put(driver.AuditorRole, wallet)
	wallet, err = mspWalletFactory.NewX509Wallet(driver.CertifierRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create certifier wallet")
	}
	wallets.Put(driver.CertifierRole, wallet)

	// Instantiate the token service
	walletDB, err := storageProvider.OpenWalletDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	ip := identity.NewProvider(
		sigService,
		view.GetEndpointService(sp),
		fscIdentity,
		NewEnrollmentIDDeserializer(),
		wallets,
		deserializerManager,
	)
	ws := fabtoken.NewWalletService(
		sigService,
		ip,
		qe,
		NewDeserializer(),
		identity.NewWalletsRegistry(ip, driver.OwnerRole, walletDB),
		identity.NewWalletsRegistry(ip, driver.IssuerRole, walletDB),
		identity.NewWalletsRegistry(ip, driver.AuditorRole, walletDB),
	)

	service := fabtoken.NewService(
		ws,
		ppm.NewPublicParamsManager(
			fabtoken.PublicParameters,
			qe,
		),
		&fabtoken.VaultTokenLoader{TokenVault: qe},
		qe,
		ip,
		NewDeserializer(),
		tmsConfig,
	)
	if err := service.PPM.SetPublicParameters(publicParams); err != nil {
		return nil, errors.WithMessage(err, "failed to update public parameters")
	}
	return service, nil
}

func (d *Driver) NewValidator(params driver.PublicParameters) (driver.Validator, error) {
	pp, ok := params.(*fabtoken.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}
	return fabtoken.NewValidator(pp, NewDeserializer())
}

func (d *Driver) NewPublicParametersManager(params driver.PublicParameters) (driver.PublicParamsManager, error) {
	pp, ok := params.(*fabtoken.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}
	return ppm.NewPublicParamsManagerFromParams(pp)
}

func (d *Driver) NewWalletService(sp driver.ServiceProvider, networkID string, channel string, namespace string, params driver.PublicParameters) (driver.WalletService, error) {
	cs := view.GetConfigService(sp)
	tmsConfig, err := config.NewTokenSDK(cs).GetTMS(networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create config manager")
	}

	// Prepare wallets
	storageProvider, err := identity.GetStorageProvider(sp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	wallets := identity.NewWallets()
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
	mspWalletFactory := msp.NewWalletFactory(
		tmsID,
		config2.NewIdentityConfig(cs, tmsConfig), // config
		nil,                                      // FSC identity
		nil,                                      // network default identity
		sigService,                               // signer service
		nil,                                      // endpoint service
		storageProvider,
		deserializerManager,
		true,
	)
	wallet, err := mspWalletFactory.NewX509WalletIgnoreRemote(driver.OwnerRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create owner wallet")
	}
	wallets.Put(driver.OwnerRole, wallet)
	wallet, err = mspWalletFactory.NewX509Wallet(driver.IssuerRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create issuer wallet")
	}
	wallets.Put(driver.IssuerRole, wallet)
	wallet, err = mspWalletFactory.NewX509Wallet(driver.AuditorRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create auditor wallet")
	}
	wallets.Put(driver.AuditorRole, wallet)
	wallet, err = mspWalletFactory.NewX509Wallet(driver.CertifierRole)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create certifier wallet")
	}
	wallets.Put(driver.CertifierRole, wallet)

	// Instantiate the token service
	// wallet service
	walletDB, err := storageProvider.OpenWalletDB(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get identity storage provider")
	}
	ip := identity.NewProvider(
		sigService,
		nil,
		nil,
		NewEnrollmentIDDeserializer(),
		wallets,
		deserializerManager,
	)
	ws := fabtoken.NewWalletService(
		sigService,
		ip,
		nil,
		NewDeserializer(),
		identity.NewWalletsRegistry(ip, driver.OwnerRole, walletDB),
		identity.NewWalletsRegistry(ip, driver.IssuerRole, walletDB),
		identity.NewWalletsRegistry(ip, driver.AuditorRole, walletDB),
	)

	return ws, nil
}

func init() {
	core.Register(fabtoken.PublicParameters, &Driver{})
}
