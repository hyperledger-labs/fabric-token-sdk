/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"context"
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
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

	errs := make(chan error, 2)

	// Listen for finality from vault
	tms := token.GetManagementService(context, token.WithTMSID(tmsID))
	assert.NotNil(tms)
	nw, err := network.GetInstance(context, tms.Network(), tms.Channel())
	assert.NoError(err, "failed getting network [%s:%s]", tms.Network(), tms.Channel())
	assert.NoError(nw.AddFinalityListener(tms.Namespace(), r.TxID, &finalityListener{errs: errs}))

	// Listen for finality from DBs
	go func() {
		_, err := context.RunView(ttx.NewFinalityWithOpts(ttx.WithTxID(r.TxID), ttx.WithTMSID(tms.ID())))
		errs <- err
	}()

	// When both arrive, return
	if err := <-errs; err != nil {
		return nil, err
	}
	return nil, <-errs
}

type TxFinalityViewFactory struct{}

func (p *TxFinalityViewFactory) NewView(in []byte) (view.View, error) {
	f := &TxFinalityView{TxFinality: &TxFinality{}}
	err := json.Unmarshal(in, f.TxFinality)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

type finalityListener struct {
	errs chan error
}

func (l *finalityListener) OnStatus(ctx context.Context, txID string, status int, message string, tokenRequestHash []byte) {
	//fmt.Printf("Received finality from network for TX [%s][%d]", txID, status)
	l.errs <- nil
}
