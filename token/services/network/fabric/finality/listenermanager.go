/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package finality

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/events"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/utils/logging"
	"go.opentelemetry.io/otel/trace"
)

var logger = logging.MustGetLogger()

type ListenerManagerProvider interface {
	NewManager(network, channel string) (ListenerManager, error)
}

type ListenerManager = driver.FinalityListenerManager

func NewListenerManagerProvider(fnsp *fabric.NetworkServiceProvider, tracerProvider trace.TracerProvider, keyTranslator translator.KeyTranslator, lmConfig config.ListenerManagerConfig) ListenerManagerProvider {
	logger.Debugf("Create Finality Listener Manager provider with config: %s", lmConfig)
	switch lmConfig.Type() {
	case config.Delivery:
		return newEndorserDeliveryBasedFLMProvider(fnsp, tracerProvider, keyTranslator, events.DeliveryListenerManagerConfig{
			MapperParallelism:       lmConfig.DeliveryMapperParallelism(),
			BlockProcessParallelism: lmConfig.DeliveryBlockProcessParallelism(),
			ListenerTimeout:         lmConfig.DeliveryListenerTimeout(),
			LRUSize:                 lmConfig.DeliveryLRUSize(),
			LRUBuffer:               lmConfig.DeliveryLRUBuffer(),
		})
	case config.Committer:
		return NewCommitterBasedFLMProvider(fnsp, tracerProvider, keyTranslator, CommitterListenerManagerConfig{
			MaxRetries:        lmConfig.CommitterMaxRetries(),
			RetryWaitDuration: lmConfig.CommitterRetryWaitDuration(),
		})
	}
	panic("unknown config type: " + lmConfig.Type())
}
