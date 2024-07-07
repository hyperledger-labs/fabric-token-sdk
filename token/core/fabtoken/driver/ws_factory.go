/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

type WalletServiceFactory struct {
	*base

	storageProvider identity.StorageProvider
}

func NewWalletServiceFactory(storageProvider identity.StorageProvider) driver.NamedFactory[driver.WalletServiceFactory] {
	return driver.NamedFactory[driver.WalletServiceFactory]{
		Name:   fabtoken.PublicParameters,
		Driver: &WalletServiceFactory{storageProvider: storageProvider},
	}
}

func (d *WalletServiceFactory) NewWalletService(tmsConfig driver.Config, _ driver.PublicParameters) (driver.WalletService, error) {
	tmsID := tmsConfig.ID()
	logger := logging.DriverLogger("token-sdk.driver.fabtoken", tmsID.Network, tmsID.Channel, tmsID.Namespace)
	return d.base.newWalletService(tmsConfig, nil, d.storageProvider, nil, logger, nil, nil, nil, true)
}
