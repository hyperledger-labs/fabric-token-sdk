/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package lookup

import (
	"context"
	"strings"

	vault2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/core/generic/vault"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/committer"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/fabricutils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/rwset"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/vault"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type newTxInfoMapper = func(network, channel string) events.EventInfoMapper[KeyInfo]

type EventsListenerManager interface {
	AddPermanentEventListener(key string, e events.ListenerEntry[KeyInfo]) error
	AddEventListener(key string, e events.ListenerEntry[KeyInfo]) error
	RemoveEventListener(key string, e events.ListenerEntry[KeyInfo]) error
}

type Listener interface {
	// OnStatus is called when the key has been found
	OnStatus(ctx context.Context, key string, value []byte)
	// OnError is called when an error occurs during the search of the key
	OnError(ctx context.Context, key string, err error)
}

type listenerEntry struct {
	namespace driver2.Namespace
	listener  Listener
}

func (e *listenerEntry) Namespace() driver2.Namespace {
	return e.namespace
}

func (e *listenerEntry) OnStatus(ctx context.Context, info KeyInfo) {
	logger.Debugf("notify info [%v] to namespace [%s]", info, e.namespace)
	if len(e.namespace) == 0 || len(info.Namespace) == 0 || e.namespace == info.Namespace {
		e.listener.OnStatus(ctx, info.Key, info.Value)
	}
}

func (e *listenerEntry) Equals(other events.ListenerEntry[KeyInfo]) bool {
	return other != nil && other.(*listenerEntry).listener == e.listener
}

type KeyInfo struct {
	Namespace driver2.Namespace
	Key       driver2.PKey
	Value     []byte
}

func (i KeyInfo) ID() driver2.PKey {
	return i.Key
}

type deliveryBasedLLMProvider struct {
	fnsp           *fabric.NetworkServiceProvider
	tracerProvider trace.TracerProvider
	config         events.DeliveryListenerManagerConfig
	newMapper      newTxInfoMapper
}

func NewDeliveryBasedLLMProvider(fnsp *fabric.NetworkServiceProvider, tracerProvider trace.TracerProvider, config events.DeliveryListenerManagerConfig, newMapper newTxInfoMapper) *deliveryBasedLLMProvider {
	return &deliveryBasedLLMProvider{
		fnsp:           fnsp,
		tracerProvider: tracerProvider,
		config:         config,
		newMapper:      newMapper,
	}
}

func newEndorserDeliveryBasedLLMProvider(fnsp *fabric.NetworkServiceProvider, tracerProvider trace.TracerProvider, keyTranslator translator.KeyTranslator, config events.DeliveryListenerManagerConfig) *deliveryBasedLLMProvider {
	prefix, err := keyTranslator.TransferActionMetadataKeyPrefix()
	if err != nil {
		panic(err)
	}
	return NewDeliveryBasedLLMProvider(fnsp, tracerProvider, config, func(network, _ string) events.EventInfoMapper[KeyInfo] {
		return &endorserTxInfoMapper{
			network: network,
			prefix:  prefix,
		}
	})
}

func (p *deliveryBasedLLMProvider) NewManager(network, channel string) (ListenerManager, error) {
	net, err := p.fnsp.FabricNetworkService(network)
	if err != nil {
		return nil, err
	}
	ch, err := net.Channel(channel)
	if err != nil {
		return nil, err
	}
	flm, err := events.NewListenerManager[KeyInfo](
		p.config,
		ch.Delivery(),
		&DeliveryScanQueryByID{
			Channel: ch,
		},
		p.tracerProvider.Tracer("finality_listener_manager", tracing.WithMetricsOpts(tracing.MetricsOpts{
			Namespace: network,
		})),
		p.newMapper(network, channel),
	)
	if err != nil {
		return nil, err
	}
	return &deliveryBasedLLM{flm}, nil
}

type deliveryBasedLLM struct {
	lm EventsListenerManager
}

func (c *deliveryBasedLLM) PermanentLookupListenerSupported() bool {
	return true
}

func (m *deliveryBasedLLM) AddPermanentLookupListener(namespace string, key string, listener Listener) error {
	return m.lm.AddPermanentEventListener(key, &listenerEntry{namespace, listener})
}

func (m *deliveryBasedLLM) AddLookupListener(namespace string, key string, startingTxID string, stopOnLastTx bool, listener Listener) error {
	return m.lm.AddEventListener(key, &listenerEntry{namespace, listener})
}

func (m *deliveryBasedLLM) RemoveLookupListener(key string, listener Listener) error {
	return m.lm.RemoveEventListener(key, &listenerEntry{"", listener})
}

type endorserTxInfoMapper struct {
	network string
	prefix  string
}

func (m *endorserTxInfoMapper) MapTxData(ctx context.Context, tx []byte, block *common.BlockMetadata, blockNum driver2.BlockNum, txNum driver2.TxNum) (map[driver2.Namespace]KeyInfo, error) {
	_, payl, chdr, err := fabricutils.UnmarshalTx(tx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unmarshaling tx [%d:%d]", blockNum, txNum)
	}
	if common.HeaderType(chdr.Type) != common.HeaderType_ENDORSER_TRANSACTION {
		logger.Debugf("Type of TX [%d:%d] is [%d]. Skipping...", blockNum, txNum, chdr.Type)
		return nil, nil
	}
	rwSet, err := rwset.NewEndorserTransactionReader(m.network).Read(payl, chdr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed extracting rwset")
	}

	if len(block.Metadata) < int(common.BlockMetadataIndex_TRANSACTIONS_FILTER) {
		return nil, errors.Errorf("block metadata lacks transaction filter")
	}
	return m.mapTxInfo(rwSet, chdr.TxId)
}

func (m *endorserTxInfoMapper) MapProcessedTx(tx *fabric.ProcessedTransaction) ([]KeyInfo, error) {
	logger.Debugf("Map processed tx [%s] with results of status [%v] and length [%d]", tx.TxID(), tx.ValidationCode(), len(tx.Results()))
	status, _ := committer.MapValidationCode(tx.ValidationCode())
	if status == driver.Invalid {
		return []KeyInfo{}, nil
	}
	rwSet, err := vault.NewPopulator().Populate(tx.Results())
	if err != nil {
		return nil, err
	}
	infos, err := m.mapTxInfo(rwSet, tx.TxID())
	if err != nil {
		return nil, err
	}
	return collections.Values(infos), nil
}

func (m *endorserTxInfoMapper) mapTxInfo(rwSet vault2.ReadWriteSet, txID string) (map[driver2.Namespace]KeyInfo, error) {
	txInfos := make(map[driver2.Namespace]KeyInfo, len(rwSet.WriteSet.Writes))
	logger.Debugf("TX [%s] has %d namespaces", txID, len(rwSet.WriteSet.Writes))
	for ns, writes := range rwSet.WriteSet.Writes {
		logger.Debugf("TX [%s:%s] has %d writes", txID, ns, len(writes))
		for key, value := range writes {
			if strings.HasPrefix(key, m.prefix) {
				logger.Debugf("TX [%s:%s] does have key [%s].", txID, ns, key)
				txInfos[ns] = KeyInfo{
					Namespace: ns,
					Key:       key,
					Value:     value,
				}
				// TODO: we assume here there is only one such a key
				break
			}
		}
	}
	return txInfos, nil
}
