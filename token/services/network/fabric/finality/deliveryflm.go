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
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/fabricutils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/finality"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/rwset"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/vault"
	driver3 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type newTxInfoMapper = func(network, channel string) finality.TxInfoMapper[TxInfo]

type listenerEntry struct {
	namespace driver2.Namespace
	listener  driver.FinalityListener
}

func (e *listenerEntry) OnStatus(ctx context.Context, info TxInfo) {
	if len(e.namespace) == 0 || len(info.Namespace) == 0 || e.namespace == info.Namespace {
		e.listener.OnStatus(ctx, info.TxId, info.Status, info.Message, info.RequestHash)
	}
}

func (e *listenerEntry) Equals(other finality.ListenerEntry[TxInfo]) bool {
	return other != nil && other.(*listenerEntry).listener == e.listener
}

type TxInfo struct {
	TxId        driver2.TxID
	Namespace   driver2.Namespace
	Status      driver.TxStatus
	Message     string
	RequestHash []byte
}

func (i TxInfo) TxID() driver2.TxID {
	return i.TxId
}

type deliveryBasedFLMProvider struct {
	fnsp           *fabric.NetworkServiceProvider
	tracerProvider trace.TracerProvider
	config         finality.DeliveryListenerManagerConfig
	newMapper      newTxInfoMapper
}

func NewDeliveryBasedFLMProvider(fnsp *fabric.NetworkServiceProvider, tracerProvider trace.TracerProvider, config finality.DeliveryListenerManagerConfig, newMapper newTxInfoMapper) *deliveryBasedFLMProvider {
	return &deliveryBasedFLMProvider{
		fnsp:           fnsp,
		tracerProvider: tracerProvider,
		config:         config,
		newMapper:      newMapper,
	}
}

func newEndorserDeliveryBasedFLMProvider(fnsp *fabric.NetworkServiceProvider, tracerProvider trace.TracerProvider, keyTranslator translator.KeyTranslator, config finality.DeliveryListenerManagerConfig) *deliveryBasedFLMProvider {
	return NewDeliveryBasedFLMProvider(fnsp, tracerProvider, config, func(network, _ string) finality.TxInfoMapper[TxInfo] {
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
	flm, err := finality.NewListenerManager[TxInfo](p.config, ch.Delivery(), p.tracerProvider.Tracer("finality_listener_manager", tracing.WithMetricsOpts(tracing.MetricsOpts{
		Namespace: network,
	})), p.newMapper(network, channel))
	if err != nil {
		return nil, err
	}
	return &deliveryBasedFLM{flm}, nil
}

type deliveryBasedFLM struct {
	lm finality.ListenerManager[TxInfo]
}

func (m *deliveryBasedFLM) AddFinalityListener(namespace string, txID string, listener driver.FinalityListener) error {
	return m.lm.AddFinalityListener(txID, &listenerEntry{namespace, listener})
}

func (m *deliveryBasedFLM) RemoveFinalityListener(txID string, listener driver.FinalityListener) error {
	return m.lm.RemoveFinalityListener(txID, &listenerEntry{"", listener})
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
	txInfos := make(map[driver2.Namespace]TxInfo, len(rwSet.WriteSet.Writes))
	logger.Debugf("TX [%s] has %d namespaces", txID, len(rwSet.WriteSet.Writes))
	for ns, write := range rwSet.WriteSet.Writes {
		logger.Debugf("TX [%s:%s] has %d writes", txID, ns, len(write))
		if requestHash, ok := write[key]; ok {
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
	return txInfos, nil
}
