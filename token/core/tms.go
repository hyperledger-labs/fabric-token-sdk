/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"sync"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/pkg/errors"

	api2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

var logger = flogging.MustGetLogger("token-sdk.core")

type CallbackFunc func(network, channel, namespace string) error

type tmsProvider struct {
	sp           view2.ServiceProvider
	callbackFunc CallbackFunc

	lock     sync.Mutex
	services map[string]api2.TokenManagerService
}

func NewTMSProvider(sp view2.ServiceProvider, callbackFunc CallbackFunc) *tmsProvider {
	ms := &tmsProvider{
		sp:           sp,
		callbackFunc: callbackFunc,
		services:     map[string]api2.TokenManagerService{},
	}
	return ms
}

func (m *tmsProvider) GetTokenManagerService(network string, channel string, namespace string, publicParamsFetcher api2.PublicParamsFetcher) (api2.TokenManagerService, error) {
	if len(network) == 0 {
		return nil, errors.Errorf("network not specified")
	}
	if len(channel) == 0 {
		return nil, errors.Errorf("channel not specified")
	}
	if len(namespace) == 0 {
		return nil, errors.Errorf("namespace not specified")
	}
	if publicParamsFetcher == nil {
		return nil, errors.Errorf("public params fetcher not specified")
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	key := network + channel + namespace
	service, ok := m.services[key]
	if !ok {
		logger.Debugf("creating new token manager service for network %s, channel %s, namespace %s", network, channel, namespace)
		var err error
		service, err = m.newTMS(network, channel, namespace, publicParamsFetcher)
		if err != nil {
			return nil, err
		}
		m.services[key] = service
	}
	return service, nil
}

func (m *tmsProvider) newTMS(networkID string, channel string, namespace string, publicParamsFetcher api2.PublicParamsFetcher) (api2.TokenManagerService, error) {
	ppRaw, err := publicParamsFetcher.Fetch()
	if err != nil {
		return nil, errors.WithMessage(err, "failed fetching public parameters")
	}
	pp, err := SerializedPublicParametersFromBytes(ppRaw)
	if err != nil {
		return nil, errors.WithMessage(err, "failed unmarshalling public parameters")
	}
	d, ok := drivers[pp.Identifier]
	if !ok {
		return nil, errors.Errorf("failed instantiate token service, driver [%s] not found", pp.Identifier)
	}
	logger.Debugf("instantiating token service for network [%s], channel [%s], namespace [%s], with driver identifier [%s]", networkID, channel, namespace, pp.Identifier)

	ts, err := d.NewTokenService(m.sp, publicParamsFetcher, networkID, channel, namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "failed instantiating token service")
	}

	if m.callbackFunc != nil {
		if err := m.callbackFunc(networkID, channel, namespace); err != nil {
			return nil, err
		}
	}
	return ts, nil
}
