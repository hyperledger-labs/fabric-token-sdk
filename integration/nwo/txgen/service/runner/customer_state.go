/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package runner

import (
	"sync/atomic"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen/model"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen/model/api"
)

type customerState struct {
	Name            model.UserAlias
	PaidAmount      api.Amount
	ReceivedAmount  api.Amount
	StartingAmount  api.Amount
	WithdrawnAmount api.Amount
	//Transactions    map[string]int // TODO think better how to store transactions
}

func (p *customerState) AddWithdrawn(m api.Amount) {
	atomic.AddInt64(&p.WithdrawnAmount, m)
}

func (p *customerState) AddPaidMount(m api.Amount) {
	atomic.AddInt64(&p.PaidAmount, m)
}

func (p *customerState) AddReceivedMount(m api.Amount) {
	atomic.AddInt64(&p.ReceivedAmount, m)
}
