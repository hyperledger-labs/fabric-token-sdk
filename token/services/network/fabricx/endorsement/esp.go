/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorsement

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/endorsement"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/endorsement/fsc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/pp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
)

type IdentityProvider interface {
	Identity(string) view.Identity
}

type ViewManager interface {
	InitiateView(view view.View, ctx context.Context) (interface{}, error)
}

type ViewRegistry = fsc.ViewRegistry

type ServiceProvider struct {
	lazy.Provider[token2.TMSID, endorsement.Service]
}

func NewServiceProvider(
	configService common.Configuration,
	viewManager ViewManager,
	viewRegistry ViewRegistry,
	identityProvider IdentityProvider,
	keyTranslator translator.KeyTranslator,
	versionKeeperProvider pp.VersionKeeperProvider,
	tokenManagementSystemProvider *token2.ManagementServiceProvider,
	storeServiceManager ttxdb.StoreServiceManager,
	fabricProvider *fabric.NetworkServiceProvider,
) *ServiceProvider {
	l := &loader{
		configService:                 configService,
		viewManager:                   viewManager,
		viewRegistry:                  viewRegistry,
		identityProvider:              identityProvider,
		keyTranslator:                 keyTranslator,
		versionKeeperProvider:         versionKeeperProvider,
		tokenManagementSystemProvider: tokenManagementSystemProvider,
		storeServiceManager:           storeServiceManager,
		fabricProvider:                fabricProvider,
	}
	return &ServiceProvider{Provider: lazy.NewProviderWithKeyMapper(key, l.load)}
}

type loader struct {
	configService                 common.Configuration
	viewManager                   ViewManager
	viewRegistry                  ViewRegistry
	identityProvider              IdentityProvider
	keyTranslator                 translator.KeyTranslator
	versionKeeperProvider         pp.VersionKeeperProvider
	storeServiceManager           ttxdb.StoreServiceManager
	tokenManagementSystemProvider *token2.ManagementServiceProvider
	fabricProvider                *fabric.NetworkServiceProvider
}

func (l *loader) load(tmsID token2.TMSID) (endorsement.Service, error) {
	configuration, err := l.configService.ConfigurationFor(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get configuration for [%s]", tmsID)
	}
	vk, err := l.versionKeeperProvider.Get(tmsID)
	if err != nil {
		return nil, err
	}
	return fsc.NewEndorsementService(
		&NamespaceTxProcessor{},
		tmsID,
		configuration,
		l.viewRegistry,
		l.viewManager,
		l.identityProvider,
		l.keyTranslator,
		func(txID string, namespace string, rws *fabric.RWSet) (fsc.Translator, error) {
			return translator.New(
				txID,
				NewRWSetWrapper(rws, namespace, txID, vk.GetVersion()),
				l.keyTranslator,
			), nil
		},
		endorsement.NewEndorserService(l.tokenManagementSystemProvider, l.fabricProvider),
		l.tokenManagementSystemProvider,
		endorsement.NewStorageProvider(l.storeServiceManager),
	)
}

func key(tmsID token2.TMSID) string {
	return tmsID.Network + tmsID.Channel + tmsID.Namespace
}

// NamespaceTxProcessor models a namespace transaction processor for fabric X
type NamespaceTxProcessor struct {
}

// EnableTxProcessing does nothing because for FabricX the endorser is stateless
func (n *NamespaceTxProcessor) EnableTxProcessing(tmsID token2.TMSID) error {
	return nil
}
