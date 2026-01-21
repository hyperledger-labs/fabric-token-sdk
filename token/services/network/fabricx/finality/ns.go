/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package finality

import (
	"context"
	"fmt"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	cdriver "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	fdriver "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabricx"
	finalityx "github.com/hyperledger-labs/fabric-smart-client/platform/fabricx/core/finality"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	ndriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/finality"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/finality/queue"
)

type ConfigService interface {
	UnmarshalKey(key string, rawVal interface{}) error
}

type QueryService interface {
	GetState(ns fabricx.Namespace, key fabricx.PKey) (*fabricx.VaultValue, error)
	GetStates(map[fabricx.Namespace][]fabricx.PKey) (map[fabricx.Namespace]map[fabricx.PKey]fabricx.VaultValue, error)
	GetTxStatus(txID string) (fabricx.ValidationCode, error)
}

type ListenerEvent struct {
	QueryService  QueryService
	KeyTranslator translator.KeyTranslator

	Listener      ndriver.FinalityListener
	TxID          string
	Status        fdriver.ValidationCode
	StatusMessage string
	Namespace     string
}

func (l *ListenerEvent) Process(ctx context.Context) error {
	// TODO: what do we do when in these cases??
	if l.Status == fdriver.Unknown || l.Status == fdriver.Busy {
		return nil
	}

	var tokenRequestHash []byte
	if l.Status == fdriver.Valid {
		// fetch token request hash key
		key, err := l.KeyTranslator.CreateTokenRequestKey(l.TxID)
		if err != nil {
			return errors.Wrapf(err, "can't create for token request [%s]", l.TxID)
		}
		v, err := l.QueryService.GetState(l.Namespace, key)
		if err != nil {
			return errors.Wrapf(err, "can't get state for token request [%s]", l.TxID)
		}
		tokenRequestHash = v.Raw
	}
	l.Listener.OnStatus(ctx, l.TxID, l.Status, l.StatusMessage, tokenRequestHash)
	return nil
}

func (l *ListenerEvent) String() string {
	return fmt.Sprintf("ListenerEvent[%s]", l.TxID)
}

type TxCheck struct {
	QueryService  QueryService
	KeyTranslator translator.KeyTranslator

	Listener  ndriver.FinalityListener
	TxID      string
	Namespace string
}

func (t *TxCheck) Process(ctx context.Context) error {
	status, err := t.QueryService.GetTxStatus(t.TxID)
	if err != nil {
		return errors.Wrapf(err, "can't get status for tx [%s]", t.TxID)
	}

	if status == fdriver.Unknown || status == fdriver.Busy {
		return nil
	}

	var tokenRequestHash []byte
	if status == fdriver.Valid {
		// fetch token request hash key
		key, err := t.KeyTranslator.CreateTokenRequestKey(t.TxID)
		if err != nil {
			return errors.Wrapf(err, "can't create for token request [%s]", t.TxID)
		}
		v, err := t.QueryService.GetState(t.Namespace, key)
		if err != nil {
			return errors.Wrapf(err, "can't get state for token request [%s]", t.TxID)
		}
		tokenRequestHash = v.Raw
	}
	t.Listener.OnStatus(ctx, t.TxID, status, "", tokenRequestHash)
	return nil
}

func (l *TxCheck) String() string {
	return fmt.Sprintf("TxCheck[%s]", l.TxID)
}

type Queue interface {
	EnqueueBlocking(ctx context.Context, event queue.Event) error
	Enqueue(event queue.Event) (err error)
}

type NSFinalityListener struct {
	namespace     string
	listener      ndriver.FinalityListener
	queue         Queue
	queryService  QueryService
	keyTranslator translator.KeyTranslator
}

func NewNSFinalityListener(
	namespace string,
	listener ndriver.FinalityListener,
	queue Queue,
	qs QueryService,
	kt translator.KeyTranslator,
) *NSFinalityListener {
	return &NSFinalityListener{
		namespace:     namespace,
		listener:      listener,
		queue:         queue,
		queryService:  qs,
		keyTranslator: kt,
	}
}

func (l *NSFinalityListener) OnStatus(ctx context.Context, txID cdriver.TxID, status fdriver.ValidationCode, statusMessage string) {
	// processing the event must be fast
	// we enqueue an event to be processed asynchronously
	if err := l.queue.EnqueueBlocking(ctx, &ListenerEvent{
		QueryService:  l.queryService,
		KeyTranslator: l.keyTranslator,
		Namespace:     l.namespace,
		Listener:      l.listener,
		TxID:          txID,
		Status:        status,
		StatusMessage: statusMessage,
	}); err != nil {
		logger.Errorf("failed processing event: %s", err)
	}
}

type NSListenerManager struct {
	lm            finalityx.ListenerManager
	queue         Queue
	queryService  QueryService
	keyTranslator translator.KeyTranslator
}

func NewNSListenerManager(
	lm finalityx.ListenerManager,
	queue Queue,
	qs QueryService,
	keyTranslator translator.KeyTranslator,
) *NSListenerManager {
	return &NSListenerManager{lm: lm, queue: queue, queryService: qs, keyTranslator: keyTranslator}
}

func (n *NSListenerManager) AddFinalityListener(namespace string, txID string, listener ndriver.FinalityListener) error {
	l := &onlyOnceListener{listener: listener}

	if err := n.queue.Enqueue(&TxCheck{
		QueryService:  n.queryService,
		KeyTranslator: n.keyTranslator,
		Listener:      l,
		TxID:          txID,
		Namespace:     namespace,
	}); err != nil {
		return err
	}
	return n.lm.AddFinalityListener(txID, NewNSFinalityListener(namespace, l, n.queue, n.queryService, n.keyTranslator))
}

func (n *NSListenerManager) RemoveFinalityListener(id string, listener ndriver.FinalityListener) error {
	// TODO
	return errors.Errorf("not supported")
}

type NSListenerManagerProvider struct {
	FNSProvider *fabricx.NetworkServiceProvider
	queue       Queue
}

func NewNotificationServiceBased(
	fnsProvider *fabricx.NetworkServiceProvider,
) (finality.ListenerManagerProvider, error) {
	q, err := queue.NewEventQueue(queue.Config{
		Workers:   10,
		QueueSize: 1000,
	})
	if err != nil {
		return nil, err
	}

	return &NSListenerManagerProvider{
		FNSProvider: fnsProvider,
		queue:       q,
	}, nil
}

func (n *NSListenerManagerProvider) NewManager(network, channel string) (finality.ListenerManager, error) {
	fn, err := n.FNSProvider.FabricNetworkService(network)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting fabric network service for [%s:%s]", network, channel)
	}
	finality, err := fn.FinalityService()
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting finality for [%s:%s]", network, channel)
	}
	return NewNSListenerManager(finality, n.queue, fn.QueryService(), &keys.Translator{}), nil
}

type onlyOnceListener struct {
	listener ndriver.FinalityListener
	once     sync.Once
}

func (o *onlyOnceListener) OnStatus(ctx context.Context, txID string, status int, message string, tokenRequestHash []byte) {
	o.once.Do(func() {
		o.listener.OnStatus(ctx, txID, status, message, tokenRequestHash)
	})
}
