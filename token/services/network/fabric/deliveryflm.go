/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"context"
	"sync"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/committer"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/fabricutils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/rwset"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

// deliveryBasedFLMProvider assumes that a listener for a transaction is added before the transaction (i.e. the corresponding block) arrives in the delivery service listener.
type deliveryBasedFLMProvider struct {
	fnsp           *fabric.NetworkServiceProvider
	tracerProvider trace.TracerProvider
	keyTranslator  translator.KeyTranslator
}

func NewDeliveryBasedFLMProvider(fnsp *fabric.NetworkServiceProvider, tracerProvider trace.TracerProvider, keyTranslator translator.KeyTranslator) *deliveryBasedFLMProvider {
	return &deliveryBasedFLMProvider{
		fnsp:           fnsp,
		tracerProvider: tracerProvider,
		keyTranslator:  keyTranslator,
	}
}

type listenerEntry struct {
	namespace driver2.Namespace
	listener  driver.FinalityListener
}

func (p *deliveryBasedFLMProvider) NewManager(network, channel string) (FinalityListenerManager, error) {
	net, err := p.fnsp.FabricNetworkService(network)
	if err != nil {
		return nil, err
	}
	ch, err := net.Channel(channel)
	if err != nil {
		return nil, err
	}

	flm := &deliveryBasedFLM{
		mapper: NewParallelResponseMapper(10, network, p.keyTranslator),
		tracer: p.tracerProvider.Tracer("finality_listener_manager", tracing.WithMetricsOpts(tracing.MetricsOpts{
			Namespace: network,
		})),
		listeners: make(map[translator.TxID][]listenerEntry),
	}
	logger.Infof("Starting delivery service for [%s:%s]", network, channel)
	go func() {
		err := ch.Delivery().ScanBlock(context.Background(), func(ctx context.Context, block *common.Block) (bool, error) {
			return false, flm.onBlock(ctx, block)
		})
		logger.Errorf("failed running delivery for [%s:%s]: %v", network, channel, err)
	}()

	return flm, nil
}

type deliveryBasedFLM struct {
	tracer trace.Tracer
	mapper *parallelBlockMapper

	mu        sync.RWMutex
	listeners map[translator.TxID][]listenerEntry
}

func (m *deliveryBasedFLM) onBlock(ctx context.Context, block *common.Block) error {
	logger.Infof("New block with %d txs detected [%d]", len(block.Data.Data), block.Header.Number)

	txs, err := m.mapper.Map(ctx, block)
	if err != nil {
		return errors.Wrapf(err, "failed to process block [%d]", block.Header.Number)
	}

	invokedTxIDs := make([]translator.TxID, 0)

	m.mu.RLock()

	for _, txInfos := range txs {
		for ns, info := range txInfos {
			logger.Infof("Look for listeners of [%s:%s]", ns, info.txID)
			// We expect there to be only one namespace.
			// The complexity is better with a listenerEntry slice (because of the write operations)
			// If more namespaces are expected, it is worth switching to a map.
			listeners, ok := m.listeners[info.txID]
			if ok {
				invokedTxIDs = append(invokedTxIDs, info.txID)
			}
			for _, entry := range listeners {
				logger.Infof("Invoking %d listeners for [%s]", len(listeners), info.txID)
				if len(entry.namespace) == 0 || len(ns) == 0 || entry.namespace == ns {
					entry.listener.OnStatus(ctx, info.txID, info.status, info.message, info.requestHash)
				}
			}
		}
	}
	m.mu.RUnlock()

	logger.Infof("Invoked listeners for %d TxIDs: [%v]. Removing listeners...", len(invokedTxIDs), invokedTxIDs)

	m.mu.Lock()
	for _, txID := range invokedTxIDs {
		delete(m.listeners, txID)
	}
	m.mu.Unlock()

	logger.Infof("Removed listeners for %d invoked TxIDs", len(invokedTxIDs))

	return nil

}

func (m *deliveryBasedFLM) AddFinalityListener(namespace string, txID string, listener driver.FinalityListener) error {
	logger.Infof("Add finality listener for [%s]", txID)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners[txID] = append(m.listeners[txID], listenerEntry{namespace, listener})
	return nil
}

func (m *deliveryBasedFLM) RemoveFinalityListener(txID string, listener driver.FinalityListener) error {
	logger.Infof("Manually invoked listener removal for [%s]", txID)
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, entry := range m.listeners[txID] {
		if entry.listener == listener {
			m.listeners[txID] = append(m.listeners[txID][:i], m.listeners[txID][i+1:]...)
			if len(m.listeners[txID]) == 0 {
				delete(m.listeners, txID)
			}
			return nil
		}
	}
	return errors.Errorf("could not find listener [%v] in txid [%s]", listener, txID)
}

type txInfo struct {
	txID        translator.TxID
	status      driver.TxStatus
	message     string
	requestHash []byte
}

type parallelBlockMapper struct {
	keyTranslator translator.KeyTranslator
	network       string
	cap           int
}

func NewParallelResponseMapper(cap int, network string, keyTranslator translator.KeyTranslator) *parallelBlockMapper {
	return &parallelBlockMapper{cap: cap, network: network, keyTranslator: keyTranslator}
}

func (m *parallelBlockMapper) Map(ctx context.Context, block *common.Block) ([]map[driver2.Namespace]txInfo, error) {
	logger.Infof("Mapping block [%d]", block.Header.Number)
	eg := errgroup.Group{}
	eg.SetLimit(m.cap)
	results := make([]map[driver2.Namespace]txInfo, len(block.Data.Data))
	for i, tx := range block.Data.Data {
		eg.Go(func() error {
			event, err := m.mapTxInfo(ctx, tx, block.Metadata, block.Header.Number, driver2.TxNum(i))
			if err != nil {
				return err
			}
			results[i] = event
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

func (m *parallelBlockMapper) mapTxInfo(ctx context.Context, tx []byte, block *common.BlockMetadata, blockNum driver2.BlockNum, txNum driver2.TxNum) (map[driver2.Namespace]txInfo, error) {
	_, payl, chdr, err := fabricutils.UnmarshalTx(tx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unmarshaling tx [%d:%d]", blockNum, txNum)
	}
	if common.HeaderType(chdr.Type) != common.HeaderType_ENDORSER_TRANSACTION {
		logger.Warnf("Type of TX [%d:%d] is [%d]. Skipping...", blockNum, txNum, chdr.Type)
		return nil, nil
	}
	rwSet, err := rwset.NewEndorserTransactionReader(m.network).Read(payl, chdr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed extracting rwset")
	}
	key, err := m.keyTranslator.CreateTokenRequestKey(chdr.TxId)
	if err != nil {
		return nil, errors.Wrapf(err, "can't create for token request [%s]", chdr.TxId)
	}
	_, finalityEvent, err := committer.MapFinalityEvent(ctx, block, txNum, chdr.TxId)
	if err != nil {
		return nil, errors.Wrapf(err, "failed mapping finality event")
	}

	txInfos := make(map[driver2.Namespace]txInfo, len(rwSet.WriteSet.Writes))
	for ns, write := range rwSet.WriteSet.Writes {
		if requestHash, ok := write[key]; ok {
			txInfos[ns] = txInfo{
				txID:        chdr.TxId,
				status:      finalityEvent.ValidationCode,
				message:     finalityEvent.ValidationMessage,
				requestHash: requestHash,
			}
		}
	}
	return txInfos, nil
}
