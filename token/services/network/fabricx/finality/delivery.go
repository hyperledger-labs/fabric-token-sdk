/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package finality

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/core/generic/vault"
	driver3 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/fabricutils"
	fdriver "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	fsxvault "github.com/hyperledger-labs/fabric-smart-client/platform/fabricx/core/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	fconfig "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/finality"
	cb "github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-x-committer/api/protoblocktx"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
)

const statusIdx = int(cb.BlockMetadataIndex_TRANSACTIONS_FILTER)

func NewDeliveryBasedFLMProvider(
	fnsProvider *fabric.NetworkServiceProvider,
	tracerProvider trace.TracerProvider,
	config fconfig.ListenerManagerConfig,
) finality.ListenerManagerProvider {
	return finality.NewDeliveryBasedFLMProvider(
		fnsProvider,
		tracerProvider,
		events.DeliveryListenerManagerConfig{
			MapperParallelism:       config.DeliveryMapperParallelism(),
			BlockProcessParallelism: config.DeliveryBlockProcessParallelism(),
			ListenerTimeout:         config.DeliveryListenerTimeout(),
			LRUSize:                 config.DeliveryLRUSize(),
			LRUBuffer:               config.DeliveryLRUBuffer(),
		},
		func(network, channel string) events.EventInfoMapper[finality.TxInfo] {
			return &messageTxInfoMapper{
				keyTranslator: &keys.Translator{},
				marshaller:    fsxvault.NewMarshaller(),
			}
		},
	)
}

type messageTxInfoMapper struct {
	keyTranslator translator.KeyTranslator
	marshaller    *fsxvault.Marshaller
}

func (m *messageTxInfoMapper) MapTxData(_ context.Context, data []byte, blkMetadata *cb.BlockMetadata, blockNum driver3.BlockNum, txNum driver3.TxNum) (map[driver3.Namespace]finality.TxInfo, error) {
	_, payload, chdr, err := fabricutils.UnmarshalTx(data)
	if err != nil {
		logger.Debugf("failed to unmarshal tx [%d:%d]: %v", blockNum, txNum, err)
		return nil, errors.Wrapf(err, "failed to unmarshal tx [%d:%d]", blockNum, txNum)
	}
	if chdr.Type != int32(cb.HeaderType_MESSAGE) {
		logger.Warnf("Tx with type [%d] found in [%d:%d]. Skipping...", chdr.Type, blockNum, txNum)
		return nil, nil
	}
	tx := &protoblocktx.Tx{}
	if err := proto.Unmarshal(payload.Data, tx); err != nil {
		logger.Debugf("failed to unmarshal tx [%d:%d]: %v", blockNum, txNum, err)
		return nil, errors.Wrapf(err, "failed to unmarshal tx payload [%d:%d]", blockNum, txNum)
	}

	if len(blkMetadata.Metadata) < statusIdx {
		return nil, errors.Errorf("block metadata lacks transaction filter")
	}
	statusCode := protoblocktx.Status(blkMetadata.Metadata[statusIdx][txNum])
	return m.mapTx(chdr.TxId, tx, statusCode)
}

func (m *messageTxInfoMapper) MapProcessedTx(tx *fabric.ProcessedTransaction) ([]finality.TxInfo, error) {
	logger.Debugf("Map processed tx [%s] with results of status [%v] and length [%d]", tx.TxID(), tx.ValidationCode(), len(tx.Results()))
	status := convertValidationCode(protoblocktx.Status(tx.ValidationCode()))
	message := protoblocktx.Status(tx.ValidationCode()).String()
	if status == driver.Invalid {
		return []finality.TxInfo{{TxId: tx.TxID(), Status: status, Message: message}}, nil
	}
	rwSet, err := m.marshaller.RWSetFromBytes(tx.Results())
	if err != nil {
		return nil, err
	}
	infos, err := m.mapRWSet(rwSet, tx.TxID(), status, message)
	if err != nil {
		return nil, err
	}
	return collections.Values(infos), nil
}

func (m *messageTxInfoMapper) mapTx(txID string, tx *protoblocktx.Tx, vc protoblocktx.Status) (map[driver3.Namespace]finality.TxInfo, error) {
	key, err := m.keyTranslator.CreateTokenRequestKey(txID)
	if err != nil {
		return nil, errors.Wrapf(err, "can't create for token request [%s]", txID)
	}

	txInfos := make(map[driver3.Namespace]finality.TxInfo, len(tx.GetNamespaces()))
	logger.Debugf("TX [%s] has %d namespaces", txID, len(tx.GetNamespaces()))
	for _, ns := range tx.GetNamespaces() {
		logger.Debugf("TX [%s:%s] has %d writes", txID, ns.GetNsId(), len(ns.GetBlindWrites()))
		for _, write := range ns.GetBlindWrites() {
			if string(write.GetKey()) == key {
				if err != nil {
					return nil, errors.Wrapf(err, "ns [%v] in tx [%s] not found", ns.GetNsId(), txID)
				}
				txInfos[ns.GetNsId()] = finality.TxInfo{
					TxId:        txID,
					Namespace:   ns.GetNsId(),
					Status:      convertValidationCode(vc),
					Message:     vc.String(),
					RequestHash: write.GetValue(),
				}
				break
			}
		}
		for _, write := range ns.GetReadWrites() {
			if string(write.GetKey()) == key {
				if err != nil {
					return nil, errors.Wrapf(err, "ns [%v] in tx [%s] not found", ns.GetNsId(), txID)
				}
				txInfos[ns.GetNsId()] = finality.TxInfo{
					TxId:        txID,
					Namespace:   ns.GetNsId(),
					Status:      convertValidationCode(vc),
					Message:     vc.String(),
					RequestHash: write.GetValue(),
				}
				break
			}
		}
		if _, ok := txInfos[ns.GetNsId()]; !ok {
			logger.Debugf("TX [%s:%s] did not have key [%s]. Found:\n\nread-writes: %v\n\nblind writes: %v", txID, ns.GetNsId(), key, ns.GetReadWrites(), ns.GetBlindWrites())
		} else {
			logger.Debugf("tx found key for [%s]", txID)
		}
	}

	return txInfos, nil
}

func (m *messageTxInfoMapper) mapRWSet(rwSet *vault.ReadWriteSet, txID string, code fdriver.ValidationCode, message string) (map[driver3.Namespace]finality.TxInfo, error) {
	key, err := m.keyTranslator.CreateTokenRequestKey(txID)
	if err != nil {
		return nil, errors.Wrapf(err, "can't create for token request [%s]", txID)
	}
	txInfos := make(map[driver3.Namespace]finality.TxInfo, len(rwSet.Writes))
	logger.Debugf("TX [%s] has %d namespaces", txID, len(rwSet.Writes))
	for ns, write := range rwSet.Writes {
		logger.Debugf("TX [%s:%s] has %d writes", txID, ns, len(write))
		if requestHash, ok := write[key]; ok {
			txInfos[ns] = finality.TxInfo{
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

func convertValidationCode(status protoblocktx.Status) fdriver.ValidationCode {
	switch status {
	case protoblocktx.Status_COMMITTED:
		return fdriver.Valid
	default:
		return fdriver.Invalid
	}
}
