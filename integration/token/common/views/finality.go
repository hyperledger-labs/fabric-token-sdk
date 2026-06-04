/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

type TxFinality struct {
	TxID    string
	TMSID   *token.TMSID
	Timeout time.Duration
}

const defaultFinalityTimeout = 5 * time.Minute

type TxFinalityView struct {
	*TxFinality
}

func (r *TxFinalityView) Call(context view.Context) (any, error) {
	var tmsID token.TMSID
	if r.TMSID != nil {
		tmsID = *r.TMSID
	}

	effectiveTimeout := r.Timeout
	if effectiveTimeout <= 0 {
		effectiveTimeout = defaultFinalityTimeout
	}

	errs := make(chan error, 2)

	// Listen for finality from vault
	tms, err := token.GetManagementService(context, token.WithTMSID(tmsID))
	assert.NoError(err)
	nw := network.GetInstance(context, tms.Network(), tms.Channel())
	assert.NotNil(nw)
	assert.NoError(nw.AddFinalityListener(tms.Namespace(), r.TxID, newFinalityListener(effectiveTimeout, errs)))

	// Listen for finality from DBs
	go func() {
		_, err := context.RunView(ttx.NewFinalityWithOpts(ttx.WithTxID(r.TxID), ttx.WithTMSID(tms.ID()), ttx.WithTimeout(effectiveTimeout)))
		errs <- err
	}()

	select {
	case err := <-errs:
		if err != nil {
			return nil, err
		}
	case <-time.After(effectiveTimeout):
		return nil, errors.New("timeout waiting for finality")
	}

	select {
	case err := <-errs:
		return nil, err
	case <-time.After(effectiveTimeout):
		return nil, errors.New("timeout waiting for finality")
	}
}

type TxFinalityViewFactory struct{}

func (p *TxFinalityViewFactory) NewView(in []byte) (view.View, error) {
	f := &TxFinalityView{TxFinality: &TxFinality{}}
	err := json.Unmarshal(in, f.TxFinality)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

type finalityListener struct {
	success func()
}

func newFinalityListener(timeout time.Duration, errs chan error) *finalityListener {
	var once sync.Once

	time.AfterFunc(timeout, func() { once.Do(func() { errs <- errors.New("timeout exceeded") }) })

	return &finalityListener{
		success: func() { once.Do(func() { errs <- nil }) },
	}
}

func (l *finalityListener) OnStatus(ctx context.Context, txID string, status int, message string, tokenRequestHash []byte) {
	fmt.Printf("Received finality from network for TX [%s][%d]", txID, status)
	l.success()
}

func (l *finalityListener) OnError(ctx context.Context, txID string, err error) {
	fmt.Printf("Finality error for TX [%s]: %v", txID, err)
}
