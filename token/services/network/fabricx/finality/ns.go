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
	finalityx "github.com/hyperledger-labs/fabric-smart-client/platform/fabricx/core/finality"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabricx/core/vault/queryservice"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	ndriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/finality"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/finality/queue"
	"github.com/hyperledger/fabric-x-committer/api/protoblocktx"
)

type ConfigService interface {
	UnmarshalKey(key string, rawVal interface{}) error
}

type QueryService interface {
	GetState(ns cdriver.Namespace, key cdriver.PKey) (*cdriver.VaultValue, error)
	GetStates(map[cdriver.Namespace][]cdriver.PKey) (map[cdriver.Namespace]map[cdriver.PKey]cdriver.VaultValue, error)
	GetTransactionStatus(txID string) (int32, error)
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
	logger.Debugf("[ListenerEvent] get notification for [%s], status [%d]", l.TxID, l.Status)

	if l.Status == fdriver.Unknown || l.Status == fdriver.Busy {
		// perform a query
		txCheck := TxCheck{
			QueryService:  l.QueryService,
			KeyTranslator: l.KeyTranslator,
			Listener:      l.Listener,
			TxID:          l.TxID,
			Namespace:     l.Namespace,
		}
		if err := txCheck.Process(ctx); err == nil {
			// this means that the query has notified the event
			return nil
		}
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
	logger.Debugf("[TxCheck] check for transaction [%s]", t.TxID)

	var err error
	s, err := t.QueryService.GetTransactionStatus(t.TxID)
	if err != nil {
		return errors.Wrapf(err, "can't get status for tx [%s]", t.TxID)
	}
	status := fabricXFSCStatus(s)

	logger.Debugf("check for transaction [%s], status [%d]", t.TxID, status)
	if status == fdriver.Unknown || status == fdriver.Busy {
		return errors.Errorf("transaction [%s] is not in a valid state", t.TxID)
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
	logger.Debugf("check for transaction [%s], notify validity", t.TxID)

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
	logger.Debugf("AddFinalityListener [%s]", txID)
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
	QueryServiceProvider queryservice.Provider
	FinalityProvider     *finalityx.Provider
	queue                Queue
}

func NewNotificationServiceBased(
	queryServiceProvider queryservice.Provider,
	finalityProvider *finalityx.Provider,
) (finality.ListenerManagerProvider, error) {
	q, err := queue.NewEventQueue(queue.Config{
		Workers:   10,
		QueueSize: 1000,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed creating event queue")
	}

	return &NSListenerManagerProvider{
		QueryServiceProvider: queryServiceProvider,
		FinalityProvider:     finalityProvider,
		queue:                q,
	}, nil
}

func (n *NSListenerManagerProvider) NewManager(network, channel string) (finality.ListenerManager, error) {
	finalityManager, err := n.FinalityProvider.NewManager(network, channel)
	if err != nil {
		return nil, errors.Wrapf(err, "failed creating finality manager")
	}

	qs, err := n.QueryServiceProvider.Get(network, channel)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting query service")
	}

	return NewNSListenerManager(finalityManager, n.queue, qs, &keys.Translator{}), nil
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

func fabricXFSCStatus(c int32) fdriver.ValidationCode {
	switch protoblocktx.Status(c) {
	case protoblocktx.Status_NOT_VALIDATED:
		return fdriver.Unknown
	case protoblocktx.Status_COMMITTED:
		return fdriver.Valid
	default:
		return fdriver.Invalid
	}
}
