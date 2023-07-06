/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

// NewWalletService returns a new instance of the wallet service for the passed public parameters
func NewWalletService(sp view.ServiceProvider, network string, channel string, namespace string, pp driver.PublicParameters) (driver.WalletService, error) {
	d, ok := drivers[pp.Identifier()]
	if !ok {
		return nil, errors.Errorf("cannot load public paramenters, driver [%s] not found", pp.Identifier())
	}
	ed, ok := d.(driver.ExtendedDriver)
	if !ok {
		return nil, errors.Errorf("cannot instantiate wallet service, the driver [%s] does not support that", pp.Identifier())
	}
	return ed.NewWalletService(sp, network, channel, namespace, pp)
}
