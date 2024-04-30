/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package network

import (
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	selector "github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/simple"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/simple/inmemory"
	"github.com/pkg/errors"
)

type FabricVault struct {
	*fabric.Vault
}

func (v *FabricVault) Status(id string) (int, error) {
	r, _, err := v.Vault.Status(id)
	return int(r), err
}

type OrionVault struct {
	orion.Vault
}

func (v *OrionVault) Status(id string) (int, error) {
	r, _, err := v.Vault.Status(id)
	return r, err
}

type LockerProvider struct {
	fabricNSP              *fabric.NetworkServiceProvider
	orionNSP               *orion.NetworkServiceProvider
	sleepTimeout           time.Duration
	validTxEvictionTimeout time.Duration
}

func NewLockerProvider(fabricNSP *fabric.NetworkServiceProvider, orionNSP *orion.NetworkServiceProvider, sleepTimeout time.Duration, validTxEvictionTimeout time.Duration) *LockerProvider {
	return &LockerProvider{
		fabricNSP:              fabricNSP,
		orionNSP:               orionNSP,
		sleepTimeout:           sleepTimeout,
		validTxEvictionTimeout: validTxEvictionTimeout,
	}
}

func (s *LockerProvider) fabricNetworkService(id string) (*fabric.NetworkService, error) {
	if s.fabricNSP == nil {
		return nil, errors.New("fabric nsp not found")
	}
	return s.fabricNSP.FabricNetworkService(id)
}

func (s *LockerProvider) orionNetworkService(id string) (*orion.NetworkService, error) {
	if s.orionNSP == nil {
		return nil, errors.New("orion nsp not found")
	}
	return s.orionNSP.NetworkService(id)
}

func (s *LockerProvider) New(network string, channel string, namespace string) selector.Locker {

	if fns, err := s.fabricNetworkService(network); err == nil {
		if ch, err := fns.Channel(channel); err == nil {
			return inmemory.NewLocker(&FabricVault{Vault: ch.Vault()}, s.sleepTimeout, s.validTxEvictionTimeout)
		}
	}

	if ons, err := s.orionNetworkService(network); err == nil {
		return inmemory.NewLocker(&OrionVault{Vault: ons.Vault()}, s.sleepTimeout, s.validTxEvictionTimeout)
	}

	panic(fmt.Sprintf("network %s not found", network))

}
