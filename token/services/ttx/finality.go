/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"context"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
)

const finalityTimeout = 10 * time.Minute

type finalityDB interface {
	AddStatusListener(txID string, ch chan common.StatusEvent)
	DeleteStatusListener(txID string, ch chan common.StatusEvent)
	GetStatus(txID string) (TxStatus, string, error)
}

type finalityView struct {
	pollingTimeout time.Duration
	opts           []TxOption
}

func (f *finalityView) Call(ctx view.Context) (interface{}, error) {
	return ctx.RunView(&FinalityView{
		pollingTimeout: f.pollingTimeout,
		opts:           f.opts,
		ttxdbManager:   utils.MustGet(ctx.GetService(&ttxdb.Manager{})).(*ttxdb.Manager),
		auditdbManager: utils.MustGet(ctx.GetService(&auditdb.Manager{})).(*auditdb.Manager),
	})
}

type FinalityView struct {
	pollingTimeout time.Duration
	opts           []TxOption

	ttxdbManager   *ttxdb.Manager
	auditdbManager *auditdb.Manager
}

// NewFinalityView returns an instance of the FinalityView.
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
func (f *FinalityView) Call(ctx view.Context) (interface{}, error) {
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
	return f.call(ctx.Context(), txID, tmsID, timeout)
}

func (f *FinalityView) call(ctx context.Context, txID string, tmsID token.TMSID, timeout time.Duration) (interface{}, error) {
	span := trace.SpanFromContext(ctx)

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("Listen to finality of [%s]", txID)
	}

	c := ctx
	if timeout != 0 {
		var cancel context.CancelFunc
		c, cancel = context.WithTimeout(c, timeout)
		defer cancel()
	}

	transactionDB, err := f.ttxdbManager.DBByTMSId(tmsID)
	if err != nil {
		return nil, err
	}
	auditDB, err := f.auditdbManager.DBByTMSId(tmsID)
	if err != nil {
		return nil, err
	}
	counter := 0
	span.AddEvent("get_ttxdb_status")
	statusTTXDB, _, err := transactionDB.GetStatus(txID)
	if err == nil && statusTTXDB != ttxdb.Unknown {
		counter++
	}
	span.AddEvent("get_auditdb_status")
	statusAuditDB, _, err := auditDB.GetStatus(txID)
	if err == nil && statusAuditDB != ttxdb.Unknown {
		counter++
	}
	if counter == 0 {
		return nil, errors.Errorf("transaction [%s] is unknown for [%s]", txID, tmsID)
	}

	span.AddEvent("listen_db_finality")
	iterations := int(timeout.Milliseconds() / f.pollingTimeout.Milliseconds())
	if iterations == 0 {
		iterations = 1
	}
	index := 0
	if statusTTXDB != ttxdb.Unknown {
		span.AddEvent("request_ttxdb_finality")
		index, err = f.dbFinality(c, txID, transactionDB, index, iterations)
		if err != nil {
			return nil, err
		}
	}
	if statusAuditDB != ttxdb.Unknown {
		span.AddEvent("request_auditdb_finality")
		_, err = f.dbFinality(c, txID, auditDB, index, iterations)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (f *FinalityView) dbFinality(c context.Context, txID string, finalityDB finalityDB, startCounter, iterations int) (int, error) {
	span := trace.SpanFromContext(c)
	// notice that adding the listener can happen after the event we are looking for has already happened
	// therefore we need to check more often before the timeout happens
	dbChannel := make(chan common.StatusEvent, 200)
	span.AddEvent("start_add_status_listener")
	finalityDB.AddStatusListener(txID, dbChannel)
	span.AddEvent("end_add_status_listener")
	defer func() {
		span.AddEvent("start_delete_status_listener")
		finalityDB.DeleteStatusListener(txID, dbChannel)
		span.AddEvent("end_delete_status_listener")
	}()

	span.AddEvent("get_status")
	status, _, err := finalityDB.GetStatus(txID)
	if err == nil {
		if status == ttxdb.Confirmed {
			return startCounter, nil
		}
		if status == ttxdb.Deleted {
			span.RecordError(errors.New("deleted transaction"))
			return startCounter, errors.Errorf("transaction [%s] is not valid", txID)
		}
	}

	span.AddEvent("listen_db_channels")
	for i := startCounter; i < iterations; i++ {
		span.AddEvent("start_new_iteration")
		timeout := time.NewTimer(f.pollingTimeout)

		select {
		case <-c.Done():
			timeout.Stop()
			return i, errors.Errorf("failed to listen to transaction [%s], timeout due to context done received [%s]", txID, c.Err())
		case event := <-dbChannel:
			span.AddEvent("receive_db_event")
			span.AddLink(trace.LinkFromContext(event.Ctx))
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("Got an answer to finality of [%s]: [%s]", txID, event)
			}
			timeout.Stop()
			if event.ValidationCode == ttxdb.Confirmed {
				return i, nil
			}
			span.RecordError(errors.New("not confirmed transaction"))
			return i, errors.Errorf("transaction [%s] is not valid [%s]", txID, TxStatusMessage[event.ValidationCode])
		case <-timeout.C:
			timeout.Stop()
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("Got a timeout for finality of [%s], check the status", txID)
			}
			vd, _, err := finalityDB.GetStatus(txID)
			if err != nil {
				logger.Debugf("Is [%s] final? not available yet, wait [err:%s, vc:%d]", txID, err, vd)
				break
			}
			switch vd {
			case ttxdb.Confirmed:
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("Listen to finality of [%s]. VALID", txID)
				}
				return i, nil
			case ttxdb.Deleted:
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("Listen to finality of [%s]. NOT VALID", txID)
				}
				span.RecordError(errors.New("deleted transactino"))
				return i, errors.Errorf("transaction [%s] is not valid", txID)
			}
		}
	}
	span.RecordError(errors.Errorf("timeout reached"))
	logger.Debugf("Is [%s] final? Failed to listen to transaction for timeout", txID)
	return iterations, errors.Errorf("failed to listen to transaction [%s] for timeout", txID)
}

type FinalityViewFactory struct {
	ttxdbManager   *ttxdb.Manager
	auditdbManager *auditdb.Manager
}

func NewFinalityViewFactory(
	ttxdbManager *ttxdb.Manager,
	auditdbManager *auditdb.Manager,
) *FinalityViewFactory {
	return &FinalityViewFactory{
		ttxdbManager:   ttxdbManager,
		auditdbManager: auditdbManager,
	}
}

func (f *FinalityViewFactory) New(pollingTimeout time.Duration, opts ...TxOption) (*FinalityView, error) {
	return &FinalityView{
		pollingTimeout: pollingTimeout,
		opts:           opts,
		ttxdbManager:   f.ttxdbManager,
		auditdbManager: f.auditdbManager,
	}, nil
}
