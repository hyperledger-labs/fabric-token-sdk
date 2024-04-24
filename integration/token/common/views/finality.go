/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

type TxFinality struct {
	TxID  string
	TMSID *token.TMSID
}

type TxFinalityView struct {
	*TxFinality
}

func (r *TxFinalityView) Call(context view.Context) (interface{}, error) {
	var tmsID token.TMSID
	if r.TMSID != nil {
		tmsID = *r.TMSID
	}
	tms := token.GetManagementService(context, token.WithTMSID(tmsID))
	assert.NotNil(tms)
	return context.RunView(ttx.NewFinalityWithOpts(ttx.WithTxID(r.TxID), ttx.WithTMSID(tms.ID())))
}

type TxFinalityViewFactory struct{}

func (p *TxFinalityViewFactory) NewView(in []byte) (view.View, error) {
	f := &TxFinalityView{TxFinality: &TxFinality{}}
	err := json.Unmarshal(in, f.TxFinality)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
