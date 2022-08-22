/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.core")

type CallbackFunc func(network, channel, namespace string) error

// TMSProvider is a token management service provider.
// It is responsible for creating token management services for different networks.
type TMSProvider struct {
	sp           view.ServiceProvider
	callbackFunc CallbackFunc

	lock     sync.Mutex
	services map[string]driver.TokenManagerService
}

func NewTMSProvider(sp view.ServiceProvider, callbackFunc CallbackFunc) *TMSProvider {
	ms := &TMSProvider{
		sp:           sp,
		callbackFunc: callbackFunc,
		services:     map[string]driver.TokenManagerService{},
	}
	return ms
}

// GetTokenManagerService returns a driver.TokenManagerService instance for the passed parameters.
// If a TokenManagerService is not available, it creates one by first fetching the public parameters using the passed driver.PublicParamsFetcher.
// If no driver is registered for the public params' identifier, it returns an error.
func (m *TMSProvider) GetTokenManagerService(network string, channel string, namespace string, publicParamsFetcher driver.PublicParamsFetcher) (driver.TokenManagerService, error) {
	if len(network) == 0 {
		return nil, errors.Errorf("network not specified")
	}
	if len(namespace) == 0 {
		return nil, errors.Errorf("namespace not specified")
	}
	m.lock.Lock()
	defer m.lock.Unlock()

	key := network + channel + namespace
	service, ok := m.services[key]
	if !ok {
		if publicParamsFetcher == nil {
			return nil, errors.Errorf("public params fetcher not specified")
		}
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

func (m *TMSProvider) newTMS(networkID string, channel string, namespace string, publicParamsFetcher driver.PublicParamsFetcher) (driver.TokenManagerService, error) {
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
