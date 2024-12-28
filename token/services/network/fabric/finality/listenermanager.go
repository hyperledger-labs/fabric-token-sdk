/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package finality

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/finality"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"go.opentelemetry.io/otel/trace"
)

var logger = logging.MustGetLogger("token-sdk.network.fabric")

type ListenerManagerProvider interface {
	NewManager(network, channel string) (ListenerManager, error)
}

type ListenerManager = driver.FinalityListenerManager

func NewListenerManagerProvider(fnsp *fabric.NetworkServiceProvider, tracerProvider trace.TracerProvider, keyTranslator translator.KeyTranslator, config ListenerManagerConfig) ListenerManagerProvider {
	logger.Infof("Create Finality Listener Manager provider with config: %s", config)
	switch config.Type() {
	case Delivery:
		return &deliveryBasedFLMProvider{
			fnsp:           fnsp,
			tracerProvider: tracerProvider,
			keyTranslator:  keyTranslator,
			config: finality.DeliveryListenerManagerConfig{
				MapperParallelism:       config.DeliveryMapperParallelism(),
				BlockProcessParallelism: config.DeliveryBlockProcessParallelism(),
				ListenerTimeout:         config.DeliveryListenerTimeout(),
				LRUSize:                 config.DeliveryLRUSize(),
				LRUBuffer:               config.DeliveryLRUBuffer(),
			},
		}
	case Committer:
		return &committerBasedFLMProvider{
			fnsp:              fnsp,
			tracerProvider:    tracerProvider,
			keyTranslator:     keyTranslator,
			maxRetries:        config.CommitterMaxRetries(),
			retryWaitDuration: config.CommitterRetryWaitDuration(),
		}
	}
	panic("unknown config type: " + config.Type())
}
