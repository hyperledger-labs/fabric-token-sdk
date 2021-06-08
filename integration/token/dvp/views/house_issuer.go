/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package views

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/state"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type IssueHouse struct {
	Address   string
	Valuation uint64
	Owner     view.Identity

	Approver view.Identity
}

type IssueHouseView struct {
	*IssueHouse
}

func (p *IssueHouseView) Call(context view.Context) (interface{}, error) {
	// Prepare transaction
	tx, err := state.NewTransaction(context)
	assert.NoError(err)
	tx.SetNamespace("house")
	assert.NoError(tx.AddCommand("issue", context.Me(), p.Owner))

	h := &House{
		Address:   p.Address,
		Valuation: p.Valuation,
		Owner:     p.Owner,
	}
	err = tx.AddOutput(h)
	assert.NoError(err, "failed adding output")

	_, err = context.RunView(state.NewCollectEndorsementsView(tx, context.Me(), p.Owner, p.Approver))
	assert.NoError(err, "failed collecting endorsement")

	// Send to the ordering service and wait for confirmation
	_, err = context.RunView(state.NewOrderingView(tx))
	assert.NoError(err, "failed asking ordering")
	return h.LinearID, nil
}

type IssueHouseViewFactory struct{}

func (p *IssueHouseViewFactory) NewView(in []byte) (view.View, error) {
	f := &IssueHouseView{IssueHouse: &IssueHouse{}}
	err := json.Unmarshal(in, f.IssueHouse)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
