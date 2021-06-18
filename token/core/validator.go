/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package core

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

func NewValidator(pp driver.PublicParameters) (driver.Validator, error) {
	d, ok := drivers[pp.Identifier()]
	if !ok {
		return nil, errors.Errorf("cannot load public paramenters, driver [%s] not found", pp.Identifier())
	}
	return d.NewValidator(pp)
}
