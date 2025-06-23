/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package finality

import (
	"context"

	vault2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/core/generic/vault"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/committer"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/fabricutils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/rwset"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/vault"
	driver3 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type newTxInfoMapper = func(network, channel string) events.EventInfoMapper[TxInfo]

type EventsListenerManager interface {
	AddEventListener(txID string, e events.ListenerEntry[TxInfo]) error
	RemoveEventListener(txID string, e events.ListenerEntry[TxInfo]) error
}

type listenerEntry struct {
	namespace driver2.Namespace
	listener  driver.FinalityListener
}

func (e *listenerEntry) Namespace() driver2.Namespace {
	return e.namespace
}

func (e *listenerEntry) OnStatus(ctx context.Context, info TxInfo) {
	logger.DebugfContext(ctx, "notify listener for tx [%s] in namespace [%s]", info.TxId, info.Namespace)
	if len(e.namespace) == 0 || len(info.Namespace) == 0 || e.namespace == info.Namespace {
		logger.DebugfContext(ctx, "notify listener for tx [%s] in namespace [%s], selected", info.TxId, info.Namespace)
		e.listener.OnStatus(ctx, info.TxId, info.Status, info.Message, info.RequestHash)
	} else {
		logger.DebugfContext(ctx, "notify listener for tx [%s] in namespace [%s], discarded", info.TxId, info.Namespace)
	}
}

func (e *listenerEntry) Equals(other events.ListenerEntry[TxInfo]) bool {
	return other != nil && other.(*listenerEntry).listener == e.listener
}

type TxInfo struct {
	TxId        driver2.TxID
	Namespace   driver2.Namespace
	Status      driver.TxStatus
	Message     string
	RequestHash []byte
}

func (i TxInfo) ID() driver2.TxID {
	return i.TxId
}

type deliveryBasedFLMProvider struct {
	fnsp           *fabric.NetworkServiceProvider
	tracerProvider trace.TracerProvider
	config         events.DeliveryListenerManagerConfig
	newMapper      newTxInfoMapper
}

func NewDeliveryBasedFLMProvider(fnsp *fabric.NetworkServiceProvider, tracerProvider trace.TracerProvider, config events.DeliveryListenerManagerConfig, newMapper newTxInfoMapper) *deliveryBasedFLMProvider {
	return &deliveryBasedFLMProvider{
		fnsp:           fnsp,
		tracerProvider: tracerProvider,
		config:         config,
		newMapper:      newMapper,
	}
}

func newEndorserDeliveryBasedFLMProvider(fnsp *fabric.NetworkServiceProvider, tracerProvider trace.TracerProvider, keyTranslator translator.KeyTranslator, config events.DeliveryListenerManagerConfig) *deliveryBasedFLMProvider {
	return NewDeliveryBasedFLMProvider(fnsp, tracerProvider, config, func(network, _ string) events.EventInfoMapper[TxInfo] {
		return &endorserTxInfoMapper{
			network:       network,
			keyTranslator: keyTranslator,
		}
	})
}

func (p *deliveryBasedFLMProvider) NewManager(network, channel string) (ListenerManager, error) {
	net, err := p.fnsp.FabricNetworkService(network)
	if err != nil {
		return nil, err
	}
	ch, err := net.Channel(channel)
	if err != nil {
		return nil, err
	}
	mapper := p.newMapper(network, channel)
	logger := logging.MustGetLogger()
	flm, err := events.NewListenerManager[TxInfo](
		logger,
		p.config,
		&Delivery{
			Delivery: ch.Delivery(),
			Ledger:   ch.Ledger(),
			Logger:   logger,
		},
		&DeliveryScanQueryByID{
			Delivery: ch.Delivery(),
			Ledger:   ch.Ledger(),
			Mapper:   mapper,
		},
		p.tracerProvider.Tracer("finality_listener_manager", tracing.WithMetricsOpts(tracing.MetricsOpts{
			Namespace: network,
		})),
		mapper,
	)
	if err != nil {
		return nil, err
	}
	return &deliveryBasedFLM{flm}, nil
}

type deliveryBasedFLM struct {
	lm EventsListenerManager
}

func (m *deliveryBasedFLM) AddFinalityListener(namespace string, txID string, listener driver.FinalityListener) error {
	return m.lm.AddEventListener(txID, &listenerEntry{namespace, listener})
}

func (m *deliveryBasedFLM) RemoveFinalityListener(txID string, listener driver.FinalityListener) error {
	return m.lm.RemoveEventListener(txID, &listenerEntry{"", listener})
}

type endorserTxInfoMapper struct {
	network       string
	keyTranslator translator.KeyTranslator
}

func (m *endorserTxInfoMapper) MapTxData(ctx context.Context, tx []byte, block *common.BlockMetadata, blockNum driver2.BlockNum, txNum driver2.TxNum) (map[driver2.Namespace]TxInfo, error) {
	_, payl, chdr, err := fabricutils.UnmarshalTx(tx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unmarshaling tx [%d:%d]", blockNum, txNum)
	}
	if common.HeaderType(chdr.Type) != common.HeaderType_ENDORSER_TRANSACTION {
		logger.DebugfContext(ctx, "Type of TX [%d:%d] is [%d]. Skipping...", blockNum, txNum, chdr.Type)
		return nil, nil
	}
	rwSet, err := rwset.NewEndorserTransactionReader(m.network).Read(payl, chdr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed extracting rwset")
	}

	if len(block.Metadata) < int(common.BlockMetadataIndex_TRANSACTIONS_FILTER) {
		return nil, errors.Errorf("block metadata lacks transaction filter")
	}
	code, message := committer.MapValidationCode(int32(committer.ValidationFlags(block.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])[txNum]))

	return m.mapTxInfo(rwSet, chdr.TxId, code, message)
}

func (m *endorserTxInfoMapper) MapProcessedTx(tx *fabric.ProcessedTransaction) ([]TxInfo, error) {
	logger.Debugf("Map processed tx [%s] with results of status [%v] and length [%d]", tx.TxID(), tx.ValidationCode(), len(tx.Results()))
	status, message := committer.MapValidationCode(tx.ValidationCode())
	if status == driver.Invalid {
		return []TxInfo{{TxId: tx.TxID(), Status: status, Message: message}}, nil
	}
	rwSet, err := vault.NewPopulator().Populate(tx.Results())
	if err != nil {
		return nil, err
	}
	infos, err := m.mapTxInfo(rwSet, tx.TxID(), status, message)
	if err != nil {
		return nil, err
	}
	return collections.Values(infos), nil
}

func (m *endorserTxInfoMapper) mapTxInfo(rwSet vault2.ReadWriteSet, txID string, code driver3.ValidationCode, message string) (map[driver2.Namespace]TxInfo, error) {
	key, err := m.keyTranslator.CreateTokenRequestKey(txID)
	if err != nil {
		return nil, errors.Wrapf(err, "can't create for token request [%s]", txID)
	}
	txInfos := make(map[driver2.Namespace]TxInfo, len(rwSet.Writes))
	logger.Debugf("TX [%s] has %d namespaces", txID, len(rwSet.Writes))
	for ns, write := range rwSet.Writes {
		logger.Debugf("TX [%s:%s] has %d writes", txID, ns, len(write))
		if requestHash, ok := write[key]; ok {
			logger.Debugf("TX [%s:%s] did have key [%s]. Found: %v", txID, ns, key, write.Keys())
			txInfos[ns] = TxInfo{
				TxId:        txID,
				Namespace:   ns,
				Status:      code,
				Message:     message,
				RequestHash: requestHash,
			}
		} else {
			logger.Debugf("TX [%s:%s] did not have key [%s]. Found: %v", txID, ns, key, write.Keys())
		}
	}
	logger.Debugf("TX [%s] has [%d] infos", txID, len(txInfos))
	return txInfos, nil
}
