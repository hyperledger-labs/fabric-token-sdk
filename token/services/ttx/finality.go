/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/common"
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

const finalityTimeout = 10 * time.Minute

type finalityDB interface {
	AddStatusListener(txID string, ch chan common.StatusEvent)
	DeleteStatusListener(txID string, ch chan common.StatusEvent)
	GetStatus(txID string) (TxStatus, string, error)
	GetStatuses(txIDs ...driver.TxID) (driver3.StatusResponseIterator, error)
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
	logger.Infof("call finality for [%s]", txID)
	fs, err := GetFinalityService(ctx, tmsID)
	if err != nil {
		logger.Errorf("error finding finality service: %v", err)
		return nil, errors.Wrapf(err, "could not find finality service")
	}
	logger.Infof("found finality service for [%s]: %v", txID, fs)
	return nil, fs.GetFinality(ctx.Context(), txID, timeout)
}

func GetFinalityService(sp view2.ServiceProvider, tmsID driver2.TMSID) (FinalityService, error) {
	logger.Infof("get finality service for [%v]", tmsID)
	fsp, err := sp.GetService(reflect.TypeOf((*FinalityServiceProvider)(nil)))
	if err != nil {
		return nil, err
	}
	logger.Infof("get auditor wallet")
	if isAuditor := MyAuditorWallet(sp) != nil; isAuditor {
		logger.Infof("is auditor")
		return fsp.(FinalityServiceProvider).NewAuditFinalityService(tmsID)
	}
	logger.Infof("is not auditor")
	return fsp.(FinalityServiceProvider).NewTTxFinalityService(tmsID)
}

type FinalityService interface {
	GetFinality(ctx context.Context, txID driver.TxID, timeout time.Duration) error
}

type FinalityServiceProvider interface {
	NewAuditFinalityService(tmsID driver2.TMSID) (FinalityService, error)
	NewTTxFinalityService(tmsID driver2.TMSID) (FinalityService, error)
}

func NewFinalityServiceProvider(auditDBProvider auditor.AuditDBProvider, ttxDBProvider network.TTXDBProvider, tracerProvider trace.TracerProvider) *finalityServiceProvider {
	return &finalityServiceProvider{
		auditFinalityProvider: finalityProvider[*auditdb.DB](auditDBProvider.DBByTMSId, tracerProvider),
		ttxFinalityProvider:   finalityProvider[*ttxdb.DB](ttxDBProvider.DBByTMSId, tracerProvider),
	}
}

type dbProvider[T finalityDB] func(tmsid driver2.TMSID) (T, error)

func key(tmsID driver2.TMSID) string {
	return tmsID.String()
}

func finalityProvider[T finalityDB](provider dbProvider[T], tracerProvider trace.TracerProvider) lazy.Provider[driver2.TMSID, *finalityService] {
	return lazy.NewProviderWithKeyMapper(key, func(tmsID driver2.TMSID) (*finalityService, error) {
		logger.Infof("create new finality provider for [%v]", tmsID)
		defer logger.Infof("created new finality provider for [%v]", tmsID)
		db, err := provider(tmsID)
		if err != nil {
			logger.Errorf("failed to provide db for [%s]: %v", tmsID, err)
			return nil, err
		}
		logger.Infof("provided db for [%s]: %v", tmsID, err)
		return newFinalityService(db, tracerProvider), nil
	})
}

type finalityServiceProvider struct {
	auditFinalityProvider lazy.Provider[driver2.TMSID, *finalityService]
	ttxFinalityProvider   lazy.Provider[driver2.TMSID, *finalityService]
}

func (p *finalityServiceProvider) NewAuditFinalityService(tmsID driver2.TMSID) (FinalityService, error) {
	return p.auditFinalityProvider.Get(tmsID)
}

func (p *finalityServiceProvider) NewTTxFinalityService(tmsID driver2.TMSID) (FinalityService, error) {
	return p.ttxFinalityProvider.Get(tmsID)
}

func newFinalityService(db finalityDB, tracerProvider trace.TracerProvider) *finalityService {
	logger.Infof("create new finality service")
	defer logger.Infof("created new finality service")
	s := &finalityService{
		db:      db,
		tracer:  tracerProvider.Tracer("finality_tracer"),
		timeout: finalityTimeout,
		polling: 1 * time.Second,
		pending: make(map[driver.TxID]pendingTx),
	}
	go s.startQueryPending(context.Background())
	return s
}

type finalityService struct {
	db     finalityDB
	tracer trace.Tracer

	timeout time.Duration
	polling time.Duration

	pending map[driver.TxID]pendingTx
	mu      sync.RWMutex
}

type pendingTx struct {
	status     chan common.StatusEvent
	iterations int
	ctx        context.Context
}

func (s *finalityService) GetFinality(ctx context.Context, txID driver.TxID, timeout time.Duration) error {
	logger.Infof("get finality for [%s]", txID)
	ctx, span := s.tracer.Start(ctx, "get_finality")
	defer span.End()
	status, _, err := s.db.GetStatus(txID)
	if err != nil {
		return errors.Wrapf(err, "error fetching tx [%s]", txID)
	}
	logger.Infof("got db result for [%s]: %v", txID, status)

	if status == ttxdb.Unknown {
		return errors.New("unknown status")
	}

	span.AddEvent("add_listener")
	ch := make(chan common.StatusEvent, 2) // TODO: Why 200 capacity?
	logger.Infof("add status listener for [%s]", txID)
	s.db.AddStatusListener(txID, ch)
	span.AddEvent("append_tx")
	s.mu.Lock()
	s.pending[txID] = pendingTx{
		status:     ch,
		iterations: max(1, int(s.timeout/s.polling)),
		ctx:        ctx,
	}
	s.mu.Unlock()

	defer s.db.DeleteStatusListener(txID, ch)
	span.AddEvent("wait_for_event")
	timer := time.NewTimer(timeout)
	logger.Infof("start timer for [%s]", txID)
	select {
	case <-ctx.Done():
		logger.Infof("context done for [%s]", txID)
		span.AddEvent("context_done")
		return errors.Errorf("context done for tx [%s]", txID)
	case <-timer.C:
		logger.Infof("timeout for [%s]", txID)
		span.AddEvent("timeout")
		return errors.Errorf("tx [%s] timed out", txID)
	case event := <-ch:
		logger.Infof("received event for [%s]: %v", txID, event)
		span.AddEvent("receive_db_event")
		span.AddLink(trace.LinkFromContext(event.Ctx))
		logger.Debugf("Got an answer to finality of [%s]: [%s]", txID, event)
		if event.ValidationCode == ttxdb.Confirmed {
			return nil
		}

		span.RecordError(errors.New("not confirmed transaction"))
		return errors.Errorf("transaction [%s] is not valid [%s]", txID, TxStatusMessage[event.ValidationCode])
	}
}

func (s *finalityService) startQueryPending(ctx context.Context) {
	ticker := time.NewTicker(s.polling)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			s.queryPending(ctx)
		}
	}
}

func (s *finalityService) queryPending(ctx context.Context) {
	logger.Infof("query pending cycle")
	ctx, span := s.tracer.Start(ctx, "query_db")
	s.mu.RLock()
	defer span.End()
	if len(s.pending) == 0 {
		s.mu.RUnlock()
		return
	}
	queryTxs := make([]driver.TxID, 0, len(s.pending))
	pendingTxs := make(map[driver.TxID]chan common.StatusEvent, len(s.pending))
	cleanupTxs := make([]driver.TxID, 0, len(s.pending))
	for txID, tx := range s.pending {
		queryTxs = append(queryTxs, txID)
		pendingTxs[txID] = tx.status
		tx.iterations--
		if tx.iterations <= 0 {
			cleanupTxs = append(cleanupTxs, txID)
		}
	}
	s.mu.RUnlock()
	defer s.cleanup(cleanupTxs...)

	span.AddEvent("query_statuses")
	logger.Infof("get statuses for [%v]", queryTxs)
	it, err := s.db.GetStatuses(queryTxs...)
	if err != nil {
		logger.Errorf("error while fetching txs [%v]: %v", queryTxs, err)
		return
	}
	span.AddEvent("read_iterator")
	statuses, err := collections.ReadAll(it)
	if err != nil {
		logger.Errorf("failed reading iterator for txs [%s]: %v", queryTxs, err)
		return
	}
	logger.Infof("received results for statuses: %v", statuses)
	span.AddEvent("push_to_queue")
	for _, status := range statuses {
		if status.ValidationCode == ttxdb.Confirmed || status.ValidationCode == ttxdb.Deleted {
			logger.Infof("pushing result for [%s]", status.TxID)
			cleanupTxs = append(cleanupTxs, status.TxID)
			pendingTxs[status.TxID] <- common.StatusEvent{
				Ctx:               ctx,
				TxID:              status.TxID,
				ValidationCode:    status.ValidationCode,
				ValidationMessage: status.ValidationMessage,
			}
			logger.Infof("pushed result for [%s]", status.TxID)
		} else {
			logger.Infof("not pushing result for [%s]", status.TxID)
		}
	}
	logger.Infof("will cleanup [%v]", cleanupTxs)
}

func (s *finalityService) cleanup(txIDs ...driver.TxID) {
	if len(txIDs) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, txID := range txIDs {
		delete(s.pending, txID)
	}
	logger.Infof("cleanup done!")
}
