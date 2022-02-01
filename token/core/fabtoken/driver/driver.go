/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/tms"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
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

func (d *Driver) NewTokenService(sp view2.ServiceProvider, publicParamsFetcher driver.PublicParamsFetcher, networkID string, channel string, namespace string) (driver.TokenManagerService, error) {
	n := network.GetInstance(sp, networkID, channel)
	if n == nil {
		return nil, errors.Errorf("network [%s] does not exists", networkID)
	}
	v, err := n.Vault(namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "vault [%s:%s] does not exists", networkID, namespace)
	}
	qe := v.TokenVault().QueryEngine()
	lm := n.LocalMembership()

	cm, err := config.NewManager(view2.GetConfigService(sp), networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create config manager")
	}

	// if the tms comes equipped with wallets, then use those wallets.
	// Otherwise, resort to network local membership
	nodeIdentity := view2.GetIdentityProvider(sp).DefaultIdentity()
	mappers := identity.NewMappers()
	tmsWalletManager := tms.NewWalletManager(sp, cm, lm.DefaultIdentity(), tms.NewSigService(view2.GetSigService(sp)), view2.GetEndpointService(sp))
	if err := tmsWalletManager.Load(); err != nil {
		return nil, errors.WithMessage(err, "failed to load wallet")
	}
	eidDeserializer := fabtoken.NewEnrollmentIDDeserializer()

	if tmsWalletManager.IsEmpty() {
		// use network local membership
		logger.Debugf("using network local membership")
		mappers.SetIssuerRole(identity.NewMapper(networkID, identity.LongTermIdentity, nodeIdentity, lm))
		mappers.SetAuditorRole(identity.NewMapper(networkID, identity.LongTermIdentity, nodeIdentity, lm))
		mappers.SetOwnerRole(identity.NewMapper(networkID, identity.LongTermIdentity, nodeIdentity, lm))
	} else {
		// use tms local membership
		logger.Debugf("using tms local membership")
		mappers.SetIssuerRole(identity.NewMapper(networkID, identity.LongTermIdentity, nodeIdentity, tmsWalletManager.Issuers()))
		mappers.SetAuditorRole(identity.NewMapper(networkID, identity.LongTermIdentity, nodeIdentity, tmsWalletManager.Auditors()))
		mappers.SetOwnerRole(identity.NewMapper(networkID, identity.LongTermIdentity, nodeIdentity, tmsWalletManager.Owners()))
	}

	return fabtoken.NewService(
		sp,
		channel,
		namespace,
		fabtoken.NewPublicParamsManager(&fabtoken.VaultPublicParamsLoader{
			TokenVault:          qe,
			PublicParamsFetcher: publicParamsFetcher,
			PPLabel:             fabtoken.PublicParameters,
		}),
		&fabtoken.VaultTokenLoader{TokenVault: qe},
		qe,
		identity.NewProvider(sp, eidDeserializer, mappers),
		fabtoken.NewDeserializer(),
		cm,
	), nil
}

func (d *Driver) NewValidator(params driver.PublicParameters) (driver.Validator, error) {
	pp, ok := params.(*fabtoken.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}
	return fabtoken.NewValidator(pp, fabtoken.NewDeserializer()), nil
}

func (d *Driver) NewPublicParametersManager(params driver.PublicParameters) (driver.PublicParamsManager, error) {
	pp, ok := params.(*fabtoken.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}
	return fabtoken.NewPublicParamsManagerFromParams(pp), nil
}

func init() {
	core.Register(fabtoken.PublicParameters, &Driver{})
}
