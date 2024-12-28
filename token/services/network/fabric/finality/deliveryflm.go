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

type listenerEntry struct {
	namespace driver2.Namespace
	listener  driver.FinalityListener
}

func (e *listenerEntry) OnStatus(ctx context.Context, info txInfo) {
	if len(e.namespace) == 0 || len(info.namespace) == 0 || e.namespace == info.namespace {
		e.listener.OnStatus(ctx, info.txID, info.status, info.message, info.requestHash)
	}
}

func (e *listenerEntry) Equals(other finality.ListenerEntry[txInfo]) bool {
	return other != nil && other.(*listenerEntry).listener == e.listener
}

type txInfo struct {
	txID        driver2.TxID
	namespace   driver2.Namespace
	status      driver.TxStatus
	message     string
	requestHash []byte
}

func (i txInfo) TxID() driver2.TxID {
	return i.txID
}

type deliveryBasedFLMProvider struct {
	fnsp           *fabric.NetworkServiceProvider
	tracerProvider trace.TracerProvider
	keyTranslator  translator.KeyTranslator
	config         finality.DeliveryListenerManagerConfig
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
	flm, err := finality.NewListenerManager[txInfo](p.config, ch.Delivery(), p.tracerProvider.Tracer("finality_listener_manager", tracing.WithMetricsOpts(tracing.MetricsOpts{
		Namespace: network,
	})), &txInfoMapper{
		network:       network,
		keyTranslator: p.keyTranslator,
	})
	if err != nil {
		return nil, err
	}
	return &deliveryBasedFLM{flm}, nil
}

type deliveryBasedFLM struct {
	lm finality.ListenerManager[txInfo]
}

func (m *deliveryBasedFLM) AddFinalityListener(namespace string, txID string, listener driver.FinalityListener) error {
	return m.lm.AddFinalityListener(txID, &listenerEntry{namespace, listener})
}

func (m *deliveryBasedFLM) RemoveFinalityListener(txID string, listener driver.FinalityListener) error {
	return m.lm.RemoveFinalityListener(txID, &listenerEntry{"", listener})
}

type txInfoMapper struct {
	network       string
	keyTranslator translator.KeyTranslator
}

func (m *txInfoMapper) MapTxData(ctx context.Context, tx []byte, block *common.BlockMetadata, blockNum driver2.BlockNum, txNum driver2.TxNum) (map[driver2.Namespace]txInfo, error) {
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

	_, finalityEvent, err := committer.MapFinalityEvent(ctx, block, txNum, chdr.TxId)
	if err != nil {
		return nil, errors.Wrapf(err, "failed mapping finality event")
	}

	code := finalityEvent.ValidationCode
	message := finalityEvent.ValidationMessage
	txID := chdr.TxId

	return m.mapTxInfo(rwSet, txID, code, message)
}

func (m *txInfoMapper) MapProcessedTx(tx *fabric.ProcessedTransaction) ([]txInfo, error) {
	logger.Debugf("Map processed tx [%s] with results of status [%v] and length [%d]", tx.TxID(), tx.ValidationCode(), len(tx.Results()))
	status, message := committer.MapValidationCode(tx.ValidationCode())
	if status == driver.Invalid {
		return []txInfo{{txID: tx.TxID(), status: status, message: message}}, nil
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

func (m *txInfoMapper) mapTxInfo(rwSet vault2.ReadWriteSet, txID string, code driver3.ValidationCode, message string) (map[driver2.Namespace]txInfo, error) {
	key, err := m.keyTranslator.CreateTokenRequestKey(txID)
	if err != nil {
		return nil, errors.Wrapf(err, "can't create for token request [%s]", txID)
	}
	txInfos := make(map[driver2.Namespace]txInfo, len(rwSet.WriteSet.Writes))
	logger.Infof("TX [%s] has %d namespaces", txID, len(rwSet.WriteSet.Writes))
	for ns, write := range rwSet.WriteSet.Writes {
		logger.Infof("TX [%s:%s] has %d writes", txID, ns, len(write))
		if requestHash, ok := write[key]; ok {
			txInfos[ns] = txInfo{
				txID:        txID,
				namespace:   ns,
				status:      code,
				message:     message,
				requestHash: requestHash,
			}
		} else {
			logger.Warnf("TX [%s:%s] did not have key [%s]. Found: %v", txID, ns, key, write.Keys())
		}
	}
	return txInfos, nil
}
