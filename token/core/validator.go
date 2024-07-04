/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

// NewValidator returns a new instance of driver.Validator for the passed parameters.
// If no driver is registered for the public params' identifier, it returns an error.
func NewValidator(sp driver.ServiceProvider, tmsID driver.TMSID, pp driver.PublicParameters) (driver.Validator, error) {
	d, ok := holder.Drivers[pp.Identifier()]
	if !ok {
		return nil, errors.Errorf("cannot load public paramenters, driver [%s] not found", pp.Identifier())
	}
	return d.NewValidator(sp, tmsID, pp)
}
