/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"context"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/committer"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/fabricutils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/finality"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/rwset"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

func NewDeliveryBasedFLMProvider(fnsp *fabric.NetworkServiceProvider, tracerProvider trace.TracerProvider, keyTranslator translator.KeyTranslator) *deliveryBasedFLMProviderWrapper {
	return &deliveryBasedFLMProviderWrapper{
		lmp:           finality.NewDeliveryBasedFLMProvider[txInfo](fnsp, tracerProvider),
		keyTranslator: keyTranslator,
	}
}

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

type deliveryBasedFLMProviderWrapper struct {
	lmp finality.ListenerManagerProvider[txInfo]

	keyTranslator translator.KeyTranslator
}

type deliveryBasedFLMWrapper struct {
	lm finality.ListenerManager[txInfo]
}

func (p *deliveryBasedFLMProviderWrapper) NewManager(network, channel string) (FinalityListenerManager, error) {
	flm, err := p.lmp.NewManager(network, channel, &txInfoMapper{
		network:       network,
		keyTranslator: p.keyTranslator,
	})
	if err != nil {
		return nil, err
	}
	return &deliveryBasedFLMWrapper{flm}, nil
}

func (m *deliveryBasedFLMWrapper) AddFinalityListener(namespace string, txID string, listener driver.FinalityListener) error {
	return m.lm.AddFinalityListener(txID, &listenerEntry{namespace, listener})
}

func (m *deliveryBasedFLMWrapper) RemoveFinalityListener(txID string, listener driver.FinalityListener) error {
	return m.lm.RemoveFinalityListener(txID, &listenerEntry{"", listener})
}

type txInfoMapper struct {
	network       string
	keyTranslator translator.KeyTranslator
}

func (m *txInfoMapper) Map(ctx context.Context, tx []byte, block *common.BlockMetadata, blockNum driver2.BlockNum, txNum driver2.TxNum) (map[driver2.Namespace]txInfo, error) {
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
	logger.Infof("TX [%s] has %d namespaces", chdr.TxId, len(rwSet.WriteSet.Writes))
	for ns, write := range rwSet.WriteSet.Writes {
		logger.Infof("TX [%s:%s] has %d writes", chdr.TxId, ns, len(write))
		if requestHash, ok := write[key]; ok {
			txInfos[ns] = txInfo{
				txID:        chdr.TxId,
				namespace:   ns,
				status:      finalityEvent.ValidationCode,
				message:     finalityEvent.ValidationMessage,
				requestHash: requestHash,
			}
		} else {
			logger.Warnf("TX [%s:%s] did not have key [%s]. Found: %v", chdr.TxId, ns, key, write.Keys())
		}
	}
	return txInfos, nil
}
