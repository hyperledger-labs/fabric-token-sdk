/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.core")

type CallbackFunc func(network, channel, namespace string) error

type Vault interface {
	PublicParams(networkID string, channel string, namespace string) ([]byte, error)
}

// TMSProvider is a token management service provider.
// It is responsible for creating token management services for different networks.
type TMSProvider struct {
	sp           view.ServiceProvider
	vault        Vault
	callbackFunc CallbackFunc

	lock     sync.Mutex
	services map[string]driver.TokenManagerService
}

func NewTMSProvider(sp view.ServiceProvider, vault Vault, callbackFunc CallbackFunc) *TMSProvider {
	ms := &TMSProvider{
		sp:           sp,
		vault:        vault,
		callbackFunc: callbackFunc,
		services:     map[string]driver.TokenManagerService{},
	}
	return ms
}

// GetTokenManagerService returns a driver.TokenManagerService instance for the passed parameters.
// If a TokenManagerService is not available, it creates one by first fetching the public parameters using the passed driver.PublicParamsFetcher.
// If no driver is registered for the public params identifier, it returns an error.
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
	driverName, err := m.driverFor(networkID, channel, namespace, publicParamsFetcher)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get driver for [%s:%s:%s]", networkID, channel, namespace)
	}
	d, ok := drivers[driverName]
	if !ok {
		return nil, errors.Errorf("failed instantiate token service, driver [%s] not found", driverName)
	}
	logger.Debugf("instantiating token service for network [%s], channel [%s], namespace [%s], with driver identifier [%s]", networkID, channel, namespace, driverName)

	ts, err := d.NewTokenService(m.sp, publicParamsFetcher, networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to instantiate token service for [%s:%s:%s]", networkID, channel, namespace)
	}

	if m.callbackFunc != nil {
		if err := m.callbackFunc(networkID, channel, namespace); err != nil {
			return nil, err
		}
	}
	return ts, nil
}

func (m *TMSProvider) driverFor(networkID string, channel string, namespace string, publicParamsFetcher driver.PublicParamsFetcher) (string, error) {
	pp, err := m.loadPublicParams(networkID, channel, namespace, publicParamsFetcher)
	if err != nil {
		// resort to configuration
		tmsConfig, err2 := config.NewTokenSDK(view.GetConfigService(m.sp)).GetTMS(networkID, channel, namespace)
		if err2 != nil {
			return "", errors.WithMessagef(err, "failed to identify driver from the configuration of [%s:%s:%s], loading driver from public parameters failed too [%s]", networkID, channel, namespace, err)
		}

		driverName := tmsConfig.TMS().Driver
		if len(driverName) != 0 {
			return driverName, nil
		}
		return "", errors.WithMessagef(err, "failed to identify driver for [%s:%s:%s]", networkID, channel, namespace)
	}
	return pp.Identifier, nil
}

func (m *TMSProvider) loadPublicParams(networkID string, channel string, namespace string, publicParamsFetcher driver.PublicParamsFetcher) (*driver.SerializedPublicParameters, error) {
	ppRaw, err := m.vault.PublicParams(networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to load public params from the vault")
	}
	if len(ppRaw) == 0 {
		ppRaw, err = publicParamsFetcher.Fetch()
		if err != nil {
			return nil, errors.WithMessage(err, "failed fetching public parameters")
		}
	}
	pp, err := SerializedPublicParametersFromBytes(ppRaw)
	if err != nil {
		return nil, errors.WithMessage(err, "failed unmarshalling public parameters")
	}
	return pp, nil
}
