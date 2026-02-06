/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package finality

import (
	finalityx "github.com/hyperledger-labs/fabric-smart-client/platform/fabricx/core/finality"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabricx/core/vault/queryservice"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/finality"
)

var logger = logging.MustGetLogger()

func NewFLMProvider(
	queryServiceProvider queryservice.Provider,
	finalityProvider *finalityx.Provider,
) (finality.ListenerManagerProvider, error) {
	logger.Infof("Creating flm provider")
	return NewNotificationServiceBased(queryServiceProvider, finalityProvider)
}
