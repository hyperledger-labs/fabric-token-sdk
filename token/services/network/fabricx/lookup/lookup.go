/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package lookup

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabricx/core/vault/queryservice"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/lookup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/finality/queue"
)

// QueryService models the FabricX query service needed by the NSListenerManager
//
//go:generate counterfeiter -o mock/qs.go -fake-name QueryService . QueryService
type QueryService = queryservice.QueryService

// QueryServiceProvider is an alias for queryservice.Provider
//
//go:generate counterfeiter -o mock/qps.go -fake-name QueryServiceProvider . QueryServiceProvider
type QueryServiceProvider = queryservice.Provider

// Queue models an event processor
//
//go:generate counterfeiter -o mock/queue.go -fake-name Queue . Queue
type Queue interface {
	// Enqueue adds an event to the queue and returns immediately
	Enqueue(event queue.Event) (err error)
}

// Listener is an alias for lookup.Listener
//
//go:generate counterfeiter -o mock/ll.go -fake-name Listener . Listener
type Listener = lookup.Listener

// NSListenerManager is a lookup listener manager that uses a query service and a queue
type NSListenerManager struct {
	queue        Queue
	queryService QueryService

	mu        sync.RWMutex
	listeners map[string][]Listener
}

// NewNSListenerManager creates a new NSListenerManager
func NewNSListenerManager(
	queue Queue,
	qs QueryService,
) *NSListenerManager {
	return &NSListenerManager{
		queue:        queue,
		queryService: qs,
		listeners:    make(map[string][]Listener),
	}
}

// PermanentLookupListenerSupported returns true if permanent lookup listeners are supported
func (n *NSListenerManager) PermanentLookupListenerSupported() bool {
	return true
}

// AddPermanentLookupListener adds a permanent lookup listener for the given key.
func (n *NSListenerManager) AddPermanentLookupListener(namespace string, key string, listener Listener) error {
	logger.Debugf("AddPermanentLookupListener [%s:%s]", namespace, key)
	n.mu.Lock()
	n.listeners[key] = append(n.listeners[key], listener)
	n.mu.Unlock()

	return n.queue.Enqueue(&PermanentKeyCheck{
		Manager:      n,
		QueryService: n.queryService,
		Queue:        n.queue,
		Namespace:    namespace,
		Key:          key,
		Listener:     listener,
		Interval:     1 * time.Minute,
	})
}

// AddLookupListener adds a lookup listener for the given key.
func (n *NSListenerManager) AddLookupListener(namespace string, key string, listener lookup.Listener) error {
	logger.Debugf("AddLookupListener [%s:%s]", namespace, key)
	n.mu.Lock()
	n.listeners[key] = append(n.listeners[key], listener)
	n.mu.Unlock()

	l := &OnlyOnceListener{listener: listener}

	return n.queue.Enqueue(&KeyCheck{
		Manager:      n,
		QueryService: n.queryService,
		Queue:        n.queue,
		Namespace:    namespace,
		Key:          key,
		Listener:     l,
		Original:     listener,
		Deadline:     time.Now().Add(5 * time.Minute),
		Interval:     2 * time.Second,
	})
}

// RemoveLookupListener removes a lookup listener for the given key.
func (n *NSListenerManager) RemoveLookupListener(id string, listener Listener) error {
	logger.Debugf("RemoveLookupListener [%s]", id)
	n.mu.Lock()
	defer n.mu.Unlock()

	listeners, ok := n.listeners[id]
	if !ok {
		return nil
	}

	for i, l := range listeners {
		if l == listener {
			n.listeners[id] = append(listeners[:i], listeners[i+1:]...)

			break
		}
	}

	if len(n.listeners[id]) == 0 {
		delete(n.listeners, id)
	}

	return nil
}

func (n *NSListenerManager) isRegistered(key string, listener Listener) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()

	listeners, ok := n.listeners[key]
	if !ok {
		return false
	}

	for _, l := range listeners {
		if l == listener {
			return true
		}
	}

	return false
}

// NSListenerManagerProvider is a provider for NSListenerManager
type NSListenerManagerProvider struct {
	QueryServiceProvider QueryServiceProvider
	queue                Queue
}

// NewQueryServiceBased creates a new NSListenerManagerProvider
func NewQueryServiceBased(queryServiceProvider QueryServiceProvider, queue Queue, ) lookup.ListenerManagerProvider {
	return &NSListenerManagerProvider{
		QueryServiceProvider: queryServiceProvider,
		queue:                queue,
	}
}

func (n *NSListenerManagerProvider) NewManager(network, channel string) (lookup.ListenerManager, error) {
	qs, err := n.QueryServiceProvider.Get(network, channel)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting query service")
	}

	return NewNSListenerManager(n.queue, qs), nil
}

// KeyCheck represents a key check event
type KeyCheck struct {
	Manager      *NSListenerManager
	QueryService QueryService
	Queue        Queue
	Namespace    string
	Key          string
	Listener     Listener
	Original     Listener
	Deadline     time.Time
	Interval     time.Duration
}

// Process processes the key check event
func (k *KeyCheck) Process(ctx context.Context) error {
	logger.Debugf("[KeyCheck] check for key [%s:%s]", k.Namespace, k.Key)

	if !k.Manager.isRegistered(k.Key, k.Original) {
		logger.Debugf("[KeyCheck] listener for key [%s:%s] no longer registered, stop", k.Namespace, k.Key)

		return nil
	}

	v, err := k.QueryService.GetState(k.Namespace, k.Key)
	if err == nil && v != nil && len(v.Raw) != 0 {
		logger.Debugf("[KeyCheck] key [%s:%s] found, notify listener", k.Namespace, k.Key)
		k.Listener.OnStatus(ctx, k.Key, v.Raw)

		return nil
	}

	if time.Now().After(k.Deadline) {
		logger.Debugf("[KeyCheck] key [%s:%s] not found, deadline reached", k.Namespace, k.Key)
		k.Listener.OnError(ctx, k.Key, errors.Errorf("key [%s:%s] not found", k.Namespace, k.Key))

		return nil
	}

	logger.Debugf("[KeyCheck] key [%s:%s] not found, reschedule in [%v]", k.Namespace, k.Key, k.Interval)
	time.AfterFunc(k.Interval, func() {
		if err := k.Queue.Enqueue(k); err != nil {
			logger.Errorf("failed re-enqueuing KeyCheck for [%s:%s]: %s", k.Namespace, k.Key, err)
		}
	})

	return nil
}

func (k *KeyCheck) String() string {
	return fmt.Sprintf("KeyCheck[%s:%s]", k.Namespace, k.Key)
}

// PermanentKeyCheck represents a permanent key check event
type PermanentKeyCheck struct {
	Manager      *NSListenerManager
	QueryService QueryService
	Queue        Queue
	Namespace    string
	Key          string
	Listener     Listener
	Interval     time.Duration
	LastValue    []byte
}

// Process processes the permanent key check event
func (k *PermanentKeyCheck) Process(ctx context.Context) error {
	logger.Debugf("[PermanentKeyCheck] check for key [%s:%s]", k.Namespace, k.Key)

	if !k.Manager.isRegistered(k.Key, k.Listener) {
		logger.Debugf("[PermanentKeyCheck] listener for key [%s:%s] no longer registered, stop", k.Namespace, k.Key)

		return nil
	}

	v, err := k.QueryService.GetState(k.Namespace, k.Key)
	if err == nil && v != nil && len(v.Raw) != 0 {
		if !bytes.Equal(k.LastValue, v.Raw) {
			logger.Debugf("[PermanentKeyCheck] key [%s:%s] found with new value, notify listener", k.Namespace, k.Key)
			k.Listener.OnStatus(ctx, k.Key, v.Raw)
			k.LastValue = v.Raw
		}
	}

	logger.Debugf("[PermanentKeyCheck] reschedule in [%v]", k.Interval)
	time.AfterFunc(k.Interval, func() {
		if err := k.Queue.Enqueue(k); err != nil {
			logger.Errorf("failed re-enqueuing PermanentKeyCheck for [%s:%s]: %s", k.Namespace, k.Key, err)
		}
	})

	return nil
}

func (k *PermanentKeyCheck) String() string {
	return fmt.Sprintf("PermanentKeyCheck[%s:%s]", k.Namespace, k.Key)
}

// OnlyOnceListener ensures that the listener is notified only once
type OnlyOnceListener struct {
	listener Listener
	once     sync.Once
}

// NewOnlyOnceListener creates a new OnlyOnceListener
func NewOnlyOnceListener(listener Listener) *OnlyOnceListener {
	return &OnlyOnceListener{listener: listener}
}

func (o *OnlyOnceListener) OnStatus(ctx context.Context, key string, value []byte) {
	o.once.Do(func() {
		o.listener.OnStatus(ctx, key, value)
	})
}

func (o *OnlyOnceListener) OnError(ctx context.Context, key string, err error) {
	o.once.Do(func() {
		o.listener.OnError(ctx, key, err)
	})
}
