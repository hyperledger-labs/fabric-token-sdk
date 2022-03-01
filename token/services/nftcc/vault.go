/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nftcc

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"reflect"
)

type CommonIteratorInterface interface {
	// HasNext returns true if the range query iterator contains additional keys
	// and values.
	HasNext() bool

	// Close closes the iterator. This should be called when done
	// reading from the iterator to free up resources.
	Close() error
}

// QueryIteratorInterface models a state iterator
type QueryIteratorInterface interface {
	CommonIteratorInterface

	Next(state interface{}) error
}

// Vault models a container of states
type Vault interface {
	// GetState loads the state identified by the tuple [namespace, id] into the passed state reference.
	GetState(namespace string, id string, state interface{}) error

	GetStateCertification(namespace string, key string) ([]byte, error)

	GetStateByPartialCompositeID(ns string, prefix string, attrs []string) (QueryIteratorInterface, error)
}

// VaultService models a vault instance provider
type VaultService interface {
	// Vault returns the world state for the passed channel.
	Vault(network string, channel string) (Vault, error)
}

func GetVaultService(ctx view2.ServiceProvider) VaultService {
	s, err := ctx.GetService(reflect.TypeOf((*VaultService)(nil)))
	if err != nil {
		panic(err)
	}
	return s.(VaultService)
}

func GetVault(ctx view2.ServiceProvider) Vault {
	ws, err := GetVaultService(ctx).Vault(
		fabric.GetDefaultFNS(ctx).Name(),
		fabric.GetDefaultChannel(ctx).Name(),
	)
	if err != nil {
		panic(err)
	}
	return ws
}

func GetVaultForChannel(ctx view2.ServiceProvider, channel string) Vault {
	ws, err := GetVaultService(ctx).Vault(
		fabric.GetDefaultFNS(ctx).Name(),
		channel,
	)
	if err != nil {
		panic(err)
	}
	return ws
}
