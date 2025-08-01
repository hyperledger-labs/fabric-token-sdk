/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package lookup

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/events"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/config"
	"go.opentelemetry.io/otel/trace"
)

var logger = logging.MustGetLogger()

type ListenerManagerProvider interface {
	NewManager(network, channel string) (ListenerManager, error)
}

type ListenerManager interface {
	PermanentLookupListenerSupported() bool
	AddPermanentLookupListener(namespace string, key string, listener Listener) error
	AddLookupListener(namespace string, key string, startingTxID string, stopOnLastTx bool, listener Listener) error
	RemoveLookupListener(id string, listener Listener) error
}

func NewListenerManagerProvider(fnsp *fabric.NetworkServiceProvider, tracerProvider trace.TracerProvider, keyTranslator translator.KeyTranslator, lmConfig config.ListenerManagerConfig) ListenerManagerProvider {
	logger.DebugfContext(ctx, "Create Lookup Listener Manager provider with config: %s", lmConfig)
	switch lmConfig.Type() {
	case config.Delivery:
		return newEndorserDeliveryBasedLLMProvider(fnsp, tracerProvider, keyTranslator, events.DeliveryListenerManagerConfig{
			MapperParallelism:       lmConfig.DeliveryMapperParallelism(),
			BlockProcessParallelism: lmConfig.DeliveryBlockProcessParallelism(),
			ListenerTimeout:         lmConfig.DeliveryListenerTimeout(),
			LRUSize:                 lmConfig.DeliveryLRUSize(),
			LRUBuffer:               lmConfig.DeliveryLRUBuffer(),
		})
	case config.Committer:
		return NewChannelBasedFLMProvider(fnsp, tracerProvider, keyTranslator, ChannelListenerManagerConfig{
			MaxRetries:        lmConfig.CommitterMaxRetries(),
			RetryWaitDuration: lmConfig.CommitterRetryWaitDuration(),
		})
	}
	panic("unknown config type: " + lmConfig.Type())
}
