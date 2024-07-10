/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorsement

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/pkg/errors"
)

var (
	logger = flogging.MustGetLogger("token-sdk.network.fabric.endorsement")
)

type IdentityProvider interface {
	Identity(string) view.Identity
}

type ViewManager interface {
	InitiateView(view view2.View, ctx context.Context) (interface{}, error)
}

type ViewRegistry = driver2.Registry

type ServiceProvider struct {
	utils.LazyProvider[token2.TMSID, Service]
}

func NewServiceProvider(
	fns *fabric.NetworkService,
	configService common.Configuration,
	viewManager ViewManager,
	viewRegistry ViewRegistry,
	identityProvider IdentityProvider,
	tmsProvider *token2.ManagementServiceProvider,
) *ServiceProvider {
	l := &loader{
		fns:              fns,
		configService:    configService,
		viewManager:      viewManager,
		viewRegistry:     viewRegistry,
		identityProvider: identityProvider,
		tmsProvider:      tmsProvider,
	}
	return &ServiceProvider{LazyProvider: utils.NewLazyProviderWithKeyMapper(key, l.load)}
}

type Service interface {
	Endorse(context view.Context, requestRaw []byte, signer view.Identity, txID driver.TxID) (driver.Envelope, error)
}

type loader struct {
	fns              *fabric.NetworkService
	configService    common.Configuration
	viewManager      ViewManager
	viewRegistry     ViewRegistry
	identityProvider IdentityProvider
	tmsProvider      *token2.ManagementServiceProvider
}

func (l *loader) load(tmsID token2.TMSID) (Service, error) {
	// if I'm an endorser, I need to process all token transactions
	configuration, err := l.configService.ConfigurationFor(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get configuration for [%s]", tmsID)
	}

	if !configuration.IsSet("services.network.fabric.endorsement") {
		return newChaincodeEndorsementService(tmsID), nil
	}

	tms, err := l.tmsProvider.GetManagementService(token2.WithTMSID(tmsID))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get tms for [%s]", tmsID)
	}
	return newFSCService(l.fns, tms, configuration, l.viewRegistry, l.viewManager, l.identityProvider)
}

func key(tmsID token2.TMSID) string {
	return tmsID.Network + tmsID.Channel + tmsID.Namespace
}
