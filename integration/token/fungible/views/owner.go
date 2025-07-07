/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

type SetTransactionOwnerStatus struct {
	TxID    string
	Status  ttx.TxStatus
	Message string
}

// SetTransactionOwnerStatusView is used to set the status of a given transaction in the audit db
type SetTransactionOwnerStatusView struct {
	*SetTransactionOwnerStatus
}

func (r *SetTransactionOwnerStatusView) Call(context view.Context) (interface{}, error) {
	owner := ttx.NewOwner(context, token.GetManagementService(context))
	assert.NoError(owner.SetStatus(context.Context(), r.TxID, r.Status, r.Message), "failed to set status of [%s] to [%d]", r.TxID, r.Status)

	return nil, nil
}

type SetTransactionOwnerStatusViewFactory struct{}

func (p *SetTransactionOwnerStatusViewFactory) NewView(in []byte) (view.View, error) {
	f := &SetTransactionOwnerStatusView{SetTransactionOwnerStatus: &SetTransactionOwnerStatus{}}
	err := json.Unmarshal(in, f.SetTransactionOwnerStatus)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
