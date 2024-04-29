/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"context"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

const finalityTimeout = 10 * time.Minute

type finalityView struct {
	pollingTimeout time.Duration
	opts           []TxOption
}

// NewFinalityView returns an instance of the finalityView.
// The view does the following: It waits for the finality of the passed transaction.
// If the transaction is final, the vault is updated.
func NewFinalityView(tx *Transaction, opts ...TxOption) *finalityView {
	return NewFinalityWithOpts(append([]TxOption{WithTransactions(tx)}, opts...)...)
}

func NewFinalityWithOpts(opts ...TxOption) *finalityView {
	return &finalityView{opts: opts, pollingTimeout: 1 * time.Second}
}

// Call executes the view.
// The view does the following: It waits for the finality of the passed transaction.
// If the transaction is final, the vault is updated.
func (f *finalityView) Call(ctx view.Context) (interface{}, error) {
	// Compile options
	options, err := compile(f.opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}
	txID := options.TxID
	tmsID := options.TMSID
	timeout := options.Timeout
	if options.Transaction != nil {
		txID = options.Transaction.ID()
		tmsID = options.Transaction.TMSID()
	}
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	return f.call(ctx, txID, tmsID, timeout)
}

func (f *finalityView) call(ctx view.Context, txID string, tmsID token.TMSID, timeout time.Duration) (interface{}, error) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("Listen to finality of [%s]", txID)
	}

	c := ctx.Context()
	if timeout != 0 {
		var cancel context.CancelFunc
		c, cancel = context.WithTimeout(c, timeout)
		defer cancel()
	}

	// notice that adding the listener can happen after the event we are looking for has already happened
	// therefore we need to check more often before the timeout happens
	dbChannel := make(chan db.StatusEvent, 200)
	transactionDB, err := ttxdb.GetByTMSId(ctx, tmsID)
	if err != nil {
		return nil, err
	}
	transactionDB.AddStatusListener(txID, dbChannel)
	defer transactionDB.DeleteStatusListener(txID, dbChannel)

	auditDB, err := auditdb.GetByTMSId(ctx, tmsID)
	if err != nil {
		return nil, err
	}
	auditDB.AddStatusListener(txID, dbChannel)
	defer auditDB.DeleteStatusListener(txID, dbChannel)

	iterations := int(timeout.Milliseconds() / f.pollingTimeout.Milliseconds())
	if iterations == 0 {
		iterations = 1
	}
	for i := 0; i < iterations; i++ {
		timeout := time.NewTimer(f.pollingTimeout)

		select {
		case <-c.Done():
			timeout.Stop()
			return nil, errors.Errorf("failed to listen to transaction [%s] for timeout", txID)
		case event := <-dbChannel:
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("Got an answer to finality of [%s]: [%s]", txID, event)
			}
			timeout.Stop()
			if event.ValidationCode == ttxdb.Confirmed {
				return nil, nil
			}
			return nil, errors.Errorf("transaction [%s] is not valid [%s]", txID, TxStatusMessage[event.ValidationCode])
		case <-timeout.C:
			timeout.Stop()
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("Got a timeout for finality of [%s], check the status", txID)
			}
			vd, _, err := transactionDB.GetStatus(txID)
			if err != nil || vd != ttxdb.Confirmed && vd != ttxdb.Deleted {
				vd, _, err = auditDB.GetStatus(txID)
			}
			if err != nil {
				logger.Debugf("Is [%s] final? not available yet, wait [err:%s, vc:%d]", txID, err, vd)
				break
			}

			switch vd {
			case ttxdb.Confirmed:
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("Listen to finality of [%s]. VALID", txID)
				}
				return nil, nil
			case ttxdb.Deleted:
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("Listen to finality of [%s]. NOT VALID", txID)
				}
				return nil, errors.Errorf("transaction [%s] is not valid", txID)
			}
		}
	}

	logger.Debugf("Is [%s] final? Failed to listen to transaction for timeout", txID)
	return nil, errors.Errorf("failed to listen to transaction [%s] for timeout", txID)
}
