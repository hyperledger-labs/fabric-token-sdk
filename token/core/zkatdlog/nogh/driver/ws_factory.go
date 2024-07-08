/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

type WalletServiceFactory struct {
	*base

	storageProvider identity.StorageProvider
}

func NewWalletServiceFactory(storageProvider identity.StorageProvider) driver.NamedFactory[driver.WalletServiceFactory] {
	return driver.NamedFactory[driver.WalletServiceFactory]{
		Name:   crypto.DLogPublicParameters,
		Driver: &WalletServiceFactory{storageProvider: storageProvider},
	}
}

func (d *WalletServiceFactory) NewWalletService(tmsConfig driver.Config, params driver.PublicParameters) (driver.WalletService, error) {
	tmsID := tmsConfig.ID()
	logger := logging.DriverLogger("token-sdk.driver.zkatdlog", tmsID.Network, tmsID.Channel, tmsID.Namespace)

	pp, ok := params.(*crypto.PublicParams)
	if !ok {
		return nil, errors.Errorf("invalid public parameters type [%T]", params)
	}

	return d.base.newWalletService(tmsConfig, nil, d.storageProvider, nil, logger, nil, nil, pp, true)
}
