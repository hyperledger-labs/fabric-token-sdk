/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package finality

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabricx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/finality"
)

var logger = logging.MustGetLogger()

func NewFLMProvider(fnsProvider *fabricx.NetworkServiceProvider) (finality.ListenerManagerProvider, error) {
	logger.Infof("Creating flm provider")
	return NewNotificationServiceBased(fnsProvider)
}
