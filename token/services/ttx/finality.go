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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/ttxdb"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

const finalityTimeout = 10 * time.Minute

type finalityDB interface {
	AddStatusListener(txID string, ch chan common.StatusEvent)
	DeleteStatusListener(txID string, ch chan common.StatusEvent)
	GetStatus(ctx context.Context, txID string) (TxStatus, string, error)
}

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
	options, err := CompileOpts(f.opts...)
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

	logger.DebugfContext(ctx.Context(), "Listen to finality of [%s]", txID)

	c := ctx.Context()
	if timeout != 0 {
		var cancel context.CancelFunc
		c, cancel = context.WithTimeout(c, timeout)
		defer cancel()
	}

	transactionDB, err := ttxdb.GetByTMSId(ctx, tmsID)
	if err != nil {
		return nil, err
	}
	auditDB, err := auditdb.GetByTMSId(ctx, tmsID)
	if err != nil {
		return nil, err
	}
	counter := 0

	statusTTXDB, _, err := transactionDB.GetStatus(ctx.Context(), txID)
	if err == nil && statusTTXDB != ttxdb.Unknown {
		counter++
	}

	statusAuditDB, _, err := auditDB.GetStatus(ctx.Context(), txID)
	if err == nil && statusAuditDB != ttxdb.Unknown {
		counter++
	}
	if counter == 0 {
		return nil, errors.Errorf("transaction [%s] is unknown for [%s]", txID, tmsID)
	}

	logger.DebugfContext(ctx.Context(), "Listen for DB finality")
	iterations := int(timeout.Milliseconds() / f.pollingTimeout.Milliseconds())
	if iterations == 0 {
		iterations = 1
	}
	index := 0
	if statusTTXDB != ttxdb.Unknown {
		logger.DebugfContext(ctx.Context(), "Request TTXDB finality")
		index, err = f.dbFinality(c, txID, transactionDB, index, iterations)
		if err != nil {
			return nil, err
		}
	}
	if statusAuditDB != ttxdb.Unknown {
		logger.DebugfContext(ctx.Context(), "Request AuditDB finality")
		_, err = f.dbFinality(c, txID, auditDB, index, iterations)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (f *finalityView) dbFinality(ctx context.Context, txID string, finalityDB finalityDB, startCounter, iterations int) (int, error) {
	// notice that adding the listener can happen after the event we are looking for has already happened
	// therefore we need to check more often before the timeout happens
	dbChannel := make(chan common.StatusEvent, 200)
	logger.DebugfContext(ctx, "Add status listener")
	finalityDB.AddStatusListener(txID, dbChannel)
	logger.DebugfContext(ctx, "Added status listener")
	defer func() {
		logger.DebugfContext(ctx, "Remove status listener")
		finalityDB.DeleteStatusListener(txID, dbChannel)
		logger.DebugfContext(ctx, "Removed status listener")
	}()

	logger.DebugfContext(ctx, "Get status")
	status, _, err := finalityDB.GetStatus(ctx, txID)
	if err == nil {
		if status == ttxdb.Confirmed {
			return startCounter, nil
		}
		if status == ttxdb.Deleted {
			logger.ErrorfContext(ctx, "Deleted tx")
			return startCounter, errors.Errorf("transaction [%s] is not valid", txID)
		}
	}

	logger.DebugfContext(ctx, "Listen DB channels")
	for i := startCounter; i < iterations; i++ {
		logger.DebugfContext(ctx, "Start iteration [%d]", i)
		timeout := time.NewTimer(f.pollingTimeout)

		select {
		case <-ctx.Done():
			timeout.Stop()
			return i, errors.Errorf("failed to listen to transaction [%s], timeout due to context done received [%s]", txID, ctx.Err())
		case event := <-dbChannel:

			trace.SpanFromContext(ctx).AddLink(trace.LinkFromContext(event.Ctx))
			logger.DebugfContext(ctx, "Got an answer to finality of [%s]: [%s]", txID, event)
			timeout.Stop()
			if event.ValidationCode == ttxdb.Confirmed {
				return i, nil
			}
			logger.ErrorfContext(ctx, "transaction [%s] is not valid [%s]", txID, TxStatusMessage[event.ValidationCode])
			return i, errors.Errorf("transaction [%s] is not valid [%s]", txID, TxStatusMessage[event.ValidationCode])
		case <-timeout.C:
			timeout.Stop()
			logger.DebugfContext(ctx, "Got a timeout for finality of [%s], check the status", txID)
			vd, _, err := finalityDB.GetStatus(ctx, txID)
			if err != nil {
				logger.DebugfContext(ctx, "Is [%s] final? not available yet, wait [err:%s, vc:%d]", txID, err, vd)
				break
			}
			switch vd {
			case ttxdb.Confirmed:
				logger.DebugfContext(ctx, "Listen to finality of [%s]. VALID", txID)

				return i, nil
			case ttxdb.Deleted:
				logger.ErrorfContext(ctx, "Listen to finality of [%s]. NOT VALID", txID)
				return i, errors.Errorf("transaction [%s] is not valid", txID)
			}
		}
	}

	logger.ErrorfContext(ctx, "Is [%s] final? Failed to listen to transaction for timeout", txID)
	return iterations, errors.Errorf("failed to listen to transaction [%s] for timeout", txID)
}
