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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	ndriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/finality"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/finality/queue"
	"github.com/hyperledger/fabric-x-committer/api/protoblocktx"
)

var logger = logging.MustGetLogger()

// ConfigService models the configuration service needed by the NSListenerManager
//
//go:generate counterfeiter -o mock/cs.go -fake-name ConfigService . ConfigService
type ConfigService interface {
	// UnmarshalKey unmarshals the configuration value for the given key into rawVal
	UnmarshalKey(key string, rawVal interface{}) error
}

// QueryService models the FabricX query service needed by the NSListenerManager
//
//go:generate counterfeiter -o mock/qs.go -fake-name QueryService . QueryService
type QueryService interface {
	// GetState returns the value for the given namespace and key
	GetState(ns cdriver.Namespace, key cdriver.PKey) (*cdriver.VaultValue, error)
	// GetStates returns the values for the given namespaces and keys
	GetStates(map[cdriver.Namespace][]cdriver.PKey) (map[cdriver.Namespace]map[cdriver.PKey]cdriver.VaultValue, error)
	// GetTransactionStatus returns the status of the given transaction
	GetTransactionStatus(txID string) (int32, error)
}

// Listener is an alias for ndriver.FinalityListener
//
//go:generate counterfeiter -o mock/fl.go -fake-name Listener . Listener
type Listener = ndriver.FinalityListener

// Queue models an event processor
//
//go:generate counterfeiter -o mock/queue.go -fake-name Queue . Queue
type Queue interface {
	// EnqueueBlocking adds an event to the queue and blocks until it is accepted or the context is canceled
	EnqueueBlocking(ctx context.Context, event queue.Event) error
	// Enqueue adds an event to the queue and returns immediately
	Enqueue(event queue.Event) (err error)
}

// KeyTranslator is an alias for translator.KeyTranslator
//
//go:generate counterfeiter -o mock/kt.go -fake-name KeyTranslator . KeyTranslator
type KeyTranslator = translator.KeyTranslator

// QueryServiceProvider is an alias for queryservice.Provider
//
//go:generate counterfeiter -o mock/qps.go -fake-name QueryServiceProvider . QueryServiceProvider
type QueryServiceProvider = queryservice.Provider

// ListenerManager is an alias for finalityx.ListenerManager
//
//go:generate counterfeiter -o mock/lm.go -fake-name ListenerManager . ListenerManager
type ListenerManager = finalityx.ListenerManager

// ListenerManagerProvider gives access to instances of ListenerManager
//
//go:generate counterfeiter -o mock/fp.go -fake-name ListenerManagerProvider . ListenerManagerProvider
type ListenerManagerProvider interface {
	NewManager(network, channel string) (ListenerManager, error)
}

// ListenerEvent represents a finality event notification
type ListenerEvent struct {
	// QueryService is the service used to query the state of the network
	QueryService QueryService
	// KeyTranslator is the service used to translate keys
	KeyTranslator KeyTranslator

	// Listener is the listener to be notified
	Listener Listener
	// TxID is the transaction ID
	TxID string
	// Status is the status of the transaction
	Status fdriver.ValidationCode
	// StatusMessage is the status message
	StatusMessage string
	// Namespace is the namespace of the transaction
	Namespace string
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

// TxCheck represents a transaction check event
type TxCheck struct {
	// QueryService is the service used to query the state of the network
	QueryService QueryService
	// KeyTranslator is the service used to translate keys
	KeyTranslator KeyTranslator

	// Listener is the listener to be notified
	Listener Listener
	// TxID is the transaction ID
	TxID string
	// Namespace is the namespace of the transaction
	Namespace string
}

// Process processes the transaction check event
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

func (t *TxCheck) String() string {
	return fmt.Sprintf("TxCheck[%s]", t.TxID)
}

// NSFinalityListener is a finality listener that uses a queue to process events
type NSFinalityListener struct {
	namespace     string
	listener      Listener
	queue         Queue
	queryService  QueryService
	keyTranslator KeyTranslator
}

// NewNSFinalityListener creates a new NSFinalityListener
func NewNSFinalityListener(
	namespace string,
	listener Listener,
	queue Queue,
	qs QueryService,
	kt KeyTranslator,
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

// NSListenerManager is a finality listener manager that uses a notification service
type NSListenerManager struct {
	lm            finalityx.ListenerManager
	queue         Queue
	queryService  QueryService
	keyTranslator KeyTranslator
}

// NewNSListenerManager creates a new NSListenerManager
func NewNSListenerManager(
	lm finalityx.ListenerManager,
	queue Queue,
	qs QueryService,
	keyTranslator KeyTranslator,
) *NSListenerManager {
	return &NSListenerManager{lm: lm, queue: queue, queryService: qs, keyTranslator: keyTranslator}
}

func (n *NSListenerManager) AddFinalityListener(namespace string, txID string, listener Listener) error {
	logger.Debugf("AddFinalityListener [%s]", txID)
	l := &OnlyOnceListener{listener: listener}

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

// NSListenerManagerProvider is a provider for NSListenerManager
type NSListenerManagerProvider struct {
	QueryServiceProvider    QueryServiceProvider
	ListenerManagerProvider ListenerManagerProvider
	queue                   Queue
}

// NewNotificationServiceBased creates a new NSListenerManagerProvider
func NewNotificationServiceBased(
	queryServiceProvider QueryServiceProvider,
	listenerManagerProvider ListenerManagerProvider,
	queue Queue,
) (finality.ListenerManagerProvider, error) {
	return &NSListenerManagerProvider{
		QueryServiceProvider:    queryServiceProvider,
		ListenerManagerProvider: listenerManagerProvider,
		queue:                   queue,
	}, nil
}

func (n *NSListenerManagerProvider) NewManager(network, channel string) (finality.ListenerManager, error) {
	finalityManager, err := n.ListenerManagerProvider.NewManager(network, channel)
	if err != nil {
		return nil, errors.Wrapf(err, "failed creating finality manager")
	}

	qs, err := n.QueryServiceProvider.Get(network, channel)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting query service")
	}

	return NewNSListenerManager(finalityManager, n.queue, qs, &keys.Translator{}), nil
}

// OnlyOnceListener ensures that the listener is notified only once
type OnlyOnceListener struct {
	listener Listener
	once     sync.Once
}

func (o *OnlyOnceListener) OnStatus(ctx context.Context, txID string, status int, message string, tokenRequestHash []byte) {
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
