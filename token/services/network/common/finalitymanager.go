/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
)

type parallelListenerManagerProvider[V comparable] struct {
	provider driver.ListenerManagerProvider[V]
}

func NewParallelListenerManagerProvider[V comparable](provider driver.ListenerManagerProvider[V]) *parallelListenerManagerProvider[V] {
	return &parallelListenerManagerProvider[V]{provider: provider}
}

func (p *parallelListenerManagerProvider[V]) NewManager() driver.ListenerManager[V] {
	return &parallelListenerManager[V]{ListenerManager: p.provider.NewManager()}
}

type parallelListenerManager[V comparable] struct {
	driver.ListenerManager[V]
}

func (m *parallelListenerManager[V]) InvokeListeners(event driver.FinalityEvent[V]) {
	m.ListenerManager.InvokeListeners(event)
}
