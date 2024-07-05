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

// NewWalletService returns a new instance of the driver.WalletService interface for the passed public parameters
func NewWalletService(sp view.ServiceProvider, network string, channel string, namespace string, pp driver.PublicParameters) (driver.WalletService, error) {
	s, err := driver.GetTokenDriverService(sp)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting token driver service")
	}
	return s.NewWalletService(driver.TMSID{Network: network, Channel: channel, Namespace: namespace}, pp)
}
