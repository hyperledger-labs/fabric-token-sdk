/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package user

import (
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen/model"
	api2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen/model/api"
)

type Provider interface {
	Get(alias model.UserAlias) User
}

type User interface {
	Username() model.Username
	InitiateTransfer(value api2.Amount, nonce api2.UUID) api2.Error
	Transfer(value api2.Amount, recipient model.Username, nonce api2.UUID) api2.Error
	Withdraw(value api2.Amount) api2.Error
	GetBalance() (api2.Amount, api2.Error)
}
