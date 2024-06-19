/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorsement

import (
	"reflect"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/pkg/errors"
)

const InvokeFunction = "invoke"

var (
	logger      = flogging.MustGetLogger("token-sdk.endorsement")
	managerType = reflect.TypeOf((*ServiceProvider)(nil))
)

type IdentityProvider interface {
	Identity(string) view.Identity
}

type ViewManager interface {
	InitiateView(view view2.View) (interface{}, error)
}

type ViewRegistry interface {
	RegisterResponder(responder view2.View, initiatedBy interface{}) error
}

type ServiceProvider struct {
	utils.LazyProvider[token2.TMSID, Service]
}

func NewServiceProvider(fnsProvider driver2.FabricNetworkServiceProvider, configService common.Configuration, viewManager ViewManager, viewRegistry ViewRegistry, identityProvider IdentityProvider, tmsProvider *token2.ManagementServiceProvider) *ServiceProvider {
	l := &loader{
		fnsProvider:      fnsProvider,
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
	fnsProvider      driver2.FabricNetworkServiceProvider
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

	nw, err := l.fnsProvider.FabricNetworkService(tmsID.Network)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get FabricNetworkService for [%s]", tmsID)
	}
	tms, err := l.tmsProvider.GetManagementService(token2.WithTMSID(tmsID))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get tms for [%s]", tmsID)
	}
	return newFSCService(nw, tms, configuration, l.viewRegistry, l.viewManager, l.identityProvider)
}

func key(tmsID token2.TMSID) string {
	return tmsID.Network + tmsID.Channel + tmsID.Namespace
}

// GetProvider returns the registered instance of Provider from the passed service provider
func GetProvider(sp token2.ServiceProvider) *ServiceProvider {
	s, err := sp.GetService(managerType)
	if err != nil {
		panic(errors.Wrapf(err, "failed to get token vault provider"))
	}
	return s.(*ServiceProvider)
}
