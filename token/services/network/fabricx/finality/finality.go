/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package finality

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabricx"
	finalityx "github.com/hyperledger-labs/fabric-smart-client/platform/fabricx/core/finality"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	fconfig "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/finality"
	"go.opentelemetry.io/otel/trace"
)

var logger = logging.MustGetLogger()

const (
	NotificationService fconfig.ManagerType = "notification_service"
)

func NewFLMProvider(
	fnsProvider *fabricx.NetworkServiceProvider,
	tracerProvider trace.TracerProvider,
	config fconfig.ListenerManagerConfig,
	fxFinalityProvider *finalityx.Provider,
) (finality.ListenerManagerProvider, error) {
	logger.Infof("Creating flm provider with config: %v", config)
	// switch config.Type() {
	// case fconfig.Delivery:
	// 	return NewDeliveryBasedFLMProvider(fnsProvider.FabricNetworkServiceProvider(), tracerProvider, config), nil
	// case NotificationService:
		return NewNotificationServiceBased(fnsProvider, fxFinalityProvider)
	// }
	// panic("unknown flm type " + config.Type())
}
