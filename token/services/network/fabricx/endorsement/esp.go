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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/pp"
)

type IdentityProvider interface {
	Identity(string) view.Identity
}

type ViewManager interface {
	InitiateView(view view.View, ctx context.Context) (interface{}, error)
}

type ViewRegistry = endorsement.ViewRegistry

type ServiceProvider struct {
	lazy.Provider[token2.TMSID, endorsement.Service]
}

func NewServiceProvider(
	fnsp *fabric.NetworkServiceProvider,
	configService common.Configuration,
	viewManager ViewManager,
	viewRegistry ViewRegistry,
	identityProvider IdentityProvider,
	keyTranslator translator.KeyTranslator,
	versionKeeperProvider pp.VersionKeeperProvider,
) *ServiceProvider {
	l := &loader{
		fnsp:                  fnsp,
		configService:         configService,
		viewManager:           viewManager,
		viewRegistry:          viewRegistry,
		identityProvider:      identityProvider,
		keyTranslator:         keyTranslator,
		versionKeeperProvider: versionKeeperProvider,
	}
	return &ServiceProvider{Provider: lazy.NewProviderWithKeyMapper(key, l.load)}
}

type loader struct {
	fnsp                  *fabric.NetworkServiceProvider
	configService         common.Configuration
	viewManager           ViewManager
	viewRegistry          ViewRegistry
	identityProvider      IdentityProvider
	keyTranslator         translator.KeyTranslator
	versionKeeperProvider pp.VersionKeeperProvider
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
	return endorsement.NewFSCService(
		l.fnsp,
		tmsID,
		configuration,
		l.viewRegistry,
		l.viewManager,
		l.identityProvider,
		l.keyTranslator,
		func(txID string, namespace string, rws *fabric.RWSet) (endorsement.Translator, error) {
			return translator.New(
				txID,
				NewRWSetWrapper(rws, namespace, txID, vk.GetVersion()),
				l.keyTranslator,
			), nil
		},
	)
}

func key(tmsID token2.TMSID) string {
	return tmsID.Network + tmsID.Channel + tmsID.Namespace
}
