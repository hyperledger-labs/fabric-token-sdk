/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	v2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/membership"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

// WalletServiceFactory is a factory for fabtoken wallet services.
type WalletServiceFactory struct {
	*base

	storageProvider identity.StorageProvider
}

// NewWalletServiceFactory returns a new factory for fabtoken wallet services.
func NewWalletServiceFactory(storageProvider identity.StorageProvider) core.NamedFactory[driver.WalletServiceFactory] {
	return core.NamedFactory[driver.WalletServiceFactory]{
		Name:   core.DriverIdentifier(v2.FabTokenDriverName, v2.ProtocolV1),
		Driver: &WalletServiceFactory{storageProvider: storageProvider},
	}
}

// NewWalletService returns a new fabtoken wallet service for the passed configuration and parameters.
func (d *WalletServiceFactory) NewWalletService(tmsConfig driver.Configuration, params driver.PublicParameters) (driver.WalletService, error) {
	tmsID := tmsConfig.ID()
	logger := logging.DriverLogger("token-sdk.driver.fabtoken", tmsID.Network, tmsID.Channel, tmsID.Namespace)

	return d.newWalletService(
		tmsConfig,
		&membership.NoBinder{},
		d.storageProvider,
		nil,
		logger,
		nil,
		nil,
		params,
		true,
	)
}
