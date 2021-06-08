/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package core

import (
	"sync"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	api2 "github.com/hyperledger-labs/fabric-token-sdk/token/api"
)

type CallbackFunc func(network, channel, namespace string) error

type Network interface {
	Channel(name string) (*fabric.Channel, error)
}

type tmsProvider struct {
	network      Network
	sp           view2.ServiceProvider
	CallbackFunc CallbackFunc

	lock     sync.Mutex
	services map[string]api2.TokenManagerService
}

func NewTMSProvider(network Network, sp view2.ServiceProvider, CallbackFunc CallbackFunc) *tmsProvider {
	ms := &tmsProvider{
		network:      network,
		sp:           sp,
		CallbackFunc: CallbackFunc,
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
		var err error
		service, err = m.newTMS(network, channel, namespace, publicParamsFetcher)
		if err != nil {
			return nil, err
		}
		m.services[key] = service
	}
	return service, nil
}

func (m *tmsProvider) newTMS(network string, channel string, namespace string, publicParamsFetcher api2.PublicParamsFetcher) (api2.TokenManagerService, error) {
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
	ch, err := m.network.Channel(channel)
	if err != nil {
		return nil, errors.Wrapf(err, "faile getting channel [%s], not found", channel)
	}
	ts, err := d.NewTokenService(m.sp, publicParamsFetcher, network, ch, namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "failed instantiating token service")
	}

	if m.CallbackFunc != nil {
		if err := m.CallbackFunc(network, channel, namespace); err != nil {
			return nil, err
		}
	}
	return ts, nil
}
