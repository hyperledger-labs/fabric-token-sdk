/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package lookup

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/fabricutils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/lookup"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-x-committer/api/protoblocktx"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/proto"
)

var _ events.EventInfoMapper[lookup.KeyInfo] = &endorserTxInfoMapper{}

var logger = logging.MustGetLogger()

// NewListenerManagerProvider returns a new lookup.ListenerManagerProvider instance.
func NewListenerManagerProvider(fnsp *fabric.NetworkServiceProvider, tracerProvider trace.TracerProvider, keyTranslator translator.KeyTranslator, lmConfig config.ListenerManagerConfig) lookup.ListenerManagerProvider {
	logger.Debugf("Create Lookup Listener Manager provider with config: %s", lmConfig)
	return newEndorserDeliveryBasedLLMProvider(fnsp, tracerProvider, keyTranslator, events.DeliveryListenerManagerConfig{
		MapperParallelism:       lmConfig.DeliveryMapperParallelism(),
		BlockProcessParallelism: lmConfig.DeliveryBlockProcessParallelism(),
		ListenerTimeout:         lmConfig.DeliveryListenerTimeout(),
		LRUSize:                 lmConfig.DeliveryLRUSize(),
		LRUBuffer:               lmConfig.DeliveryLRUBuffer(),
	})
}

func newEndorserDeliveryBasedLLMProvider(fnsp *fabric.NetworkServiceProvider, tracerProvider trace.TracerProvider, keyTranslator translator.KeyTranslator, config events.DeliveryListenerManagerConfig) lookup.ListenerManagerProvider {
	prefix, err := keyTranslator.TransferActionMetadataKeyPrefix()
	if err != nil {
		panic(err)
	}
	setupKey, err := keyTranslator.CreateSetupKey()
	if err != nil {
		panic(err)
	}

	return lookup.NewDeliveryBasedLLMProvider(fnsp, tracerProvider, config, func(network, channel string) events.EventInfoMapper[lookup.KeyInfo] {
		return &endorserTxInfoMapper{
			keyTranslator: keyTranslator,
			network:       network,
			prefixes:      []string{prefix, setupKey},
		}
	})
}

// endorserTxInfoMapper models a transaction info mapper for the endorser.
type endorserTxInfoMapper struct {
	keyTranslator translator.KeyTranslator

	network  string
	prefixes []string
}

// MapTxData unmarshals the given transaction data and returns a map of namespace to lookup.KeyInfo.
func (m *endorserTxInfoMapper) MapTxData(ctx context.Context, data []byte, block *common.BlockMetadata, blockNum driver.BlockNum, txNum driver.TxNum) (map[driver.Namespace]lookup.KeyInfo, error) {
	_, payload, chdr, err := fabricutils.UnmarshalTx(data)
	if err != nil {
		logger.Debugf("failed to unmarshal tx [%d:%d]: %v", blockNum, txNum, err)

		return nil, errors.Wrapf(err, "failed to unmarshal tx [%d:%d]", blockNum, txNum)
	}
	if chdr.Type != int32(common.HeaderType_MESSAGE) {
		logger.Warnf("Tx with type [%d] found in [%d:%d]. Skipping...", chdr.Type, blockNum, txNum)

		return nil, nil
	}

	tx := &protoblocktx.Tx{}
	if err := proto.Unmarshal(payload.Data, tx); err != nil {
		logger.Debugf("failed to unmarshal tx [%d:%d]: %v", blockNum, txNum, err)

		return nil, errors.Wrapf(err, "failed to unmarshal tx payload [%d:%d]", blockNum, txNum)
	}

	return m.mapTx(chdr.TxId, tx)
}

func (m *endorserTxInfoMapper) mapTx(txID string, tx *protoblocktx.Tx) (map[driver.Namespace]lookup.KeyInfo, error) {
	key, err := m.keyTranslator.CreateSetupKey()
	if err != nil {
		return nil, errors.Wrapf(err, "can't create for token request [%s]", txID)
	}

	txInfos := make(map[driver.Namespace]lookup.KeyInfo, len(tx.GetNamespaces()))
	for _, ns := range tx.GetNamespaces() {
		for _, write := range ns.GetBlindWrites() {
			if string(write.GetKey()) == key {
				txInfos[ns.GetNsId()] = lookup.KeyInfo{
					Key:       string(write.GetKey()),
					Namespace: ns.GetNsId(),
					Value:     write.GetValue(),
				}

				break
			}
		}

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			if _, ok := txInfos[ns.GetNsId()]; !ok {
				logger.Debugf("TX [%s:%s] did not have key [%s]. Found:\n\nread-writes: %v\n\nblind writes: %v\n\nreads: %v", txID, ns.GetNsId(), key, ns.GetReadWrites(), ns.GetBlindWrites(), ns.GetReadsOnly())
			} else {
				logger.Debugf("TX [%s:%s] had key [%s]. Found:\n\nread-writes: %v\n\nblind writes: %v\n\nreads: %v", txID, ns.GetNsId(), key, ns.GetReadWrites(), ns.GetBlindWrites(), ns.GetReadsOnly())
			}
		}
	}

	return txInfos, nil
}

// MapProcessedTx is not implemented.
func (m *endorserTxInfoMapper) MapProcessedTx(*fabric.ProcessedTransaction) ([]lookup.KeyInfo, error) {
	panic("unimplemented")
}
