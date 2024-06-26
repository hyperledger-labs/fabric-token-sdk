/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package user

import (
	"github.com/google/uuid"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model"
	"github.com/hyperledger-labs/fabric-token-sdk/txgen/model/api"
)

type Provider interface {
	Get(alias model.UserAlias) User
}

type User interface {
	Username() model.Username
	InitiateTransfer(value api.Amount, nonce uuid.UUID) api.Error
	Transfer(value api.Amount, recipient model.Username, nonce uuid.UUID) api.Error
	Withdraw(value api.Amount) api.Error
	GetBalance() (api.Amount, api.Error)
}
