/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package recovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/ttxdb"
)

const (
	defaultLockID        int64 = 0x74746b7265636f76
	defaultBatchSize           = 100
	defaultWorkers             = 4
	defaultLeaseDuration       = 30 * time.Second
)

//go:generate counterfeiter -o mock/storage.go -fake-name Storage . Storage

// Storage defines the interface for querying pending transactions and transaction details
type Storage interface {
	AcquireRecoveryLeadership(ctx context.Context, lockID int64) (Leadership, bool, error)
	ClaimPendingTransactions(ctx context.Context, olderThan time.Duration, leaseDuration time.Duration, limit int, owner string) ([]*ttxdb.TransactionRecord, error)
	ReleaseRecoveryClaim(ctx context.Context, txID string, owner string, message string) error
}

//go:generate counterfeiter -o mock/handler.go -fake-name Handler . Handler

// Handler handles the recovery of a single transaction
type Handler interface {
	// Recover attempts to recover a transaction by re-registering its finality listener
	// Returns an error if recovery fails
	Recover(ctx context.Context, txID string) error
}

//go:generate counterfeiter -o mock/leadership.go -fake-name Leadership . Leadership

// Leadership represents an acquired advisory lock leadership session.
type Leadership = dbdriver.RecoveryLeadership

// Manager handles the recovery of transactions that may have lost their finality listeners
type Manager struct {
	logger  logging.Logger
	storage Storage
	handler Handler
	config  Config
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	started bool
	mu      sync.Mutex
}

// NewManager creates a new recovery manager
func NewManager(
	logger logging.Logger,
	storage Storage,
	handler Handler,
	config Config,
) *Manager {
	return &Manager{
		logger:  logger,
		storage: storage,
		handler: handler,
		config:  config,
	}
}

// Start begins the recovery process
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.config.Enabled {
		m.logger.Debugf("transaction recovery is disabled")

		return nil
	}

	if m.started {
		return errors.Errorf("recovery manager already started")
	}

	if err := m.validateConfig(); err != nil {
		return err
	}

	if m.config.InstanceID == "" {
		m.config.InstanceID = fmt.Sprintf("recovery-%p", m)
	}

	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.started = true

	m.wg.Add(1)
	go m.recoveryLoop()

	m.logger.Infof("transaction recovery manager started (TTL: %s, Scan Interval: %s, Batch Size: %d, Workers: %d, Lease Duration: %s, Lock ID: %d, Instance ID: %s)",
		m.config.TTL, m.config.ScanInterval, m.config.BatchSize, m.config.WorkerCount, m.config.LeaseDuration, m.config.AdvisoryLockID, m.config.InstanceID)

	return nil
}

// Stop gracefully stops the recovery process
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return nil
	}

	m.logger.Infof("stopping transaction recovery manager")
	m.cancel()
	m.wg.Wait()
	m.started = false
	m.logger.Infof("transaction recovery manager stopped")

	return nil
}

// recoveryLoop is the main loop that periodically scans for transactions needing recovery
func (m *Manager) recoveryLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.ScanInterval)
	defer ticker.Stop()

	if err := m.runSweep(m.ctx); err != nil {
		m.logger.Warnf("initial transaction recovery sweep failed: %v", err)
	}

	for {
		select {
		case <-m.ctx.Done():
			m.logger.Debugf("recovery loop stopped")

			return
		case <-ticker.C:
			if err := m.runSweep(m.ctx); err != nil {
				m.logger.Warnf("transaction recovery sweep failed: %v", err)
			}
		}
	}
}

func (m *Manager) validateConfig() error {
	switch {
	case m.config.TTL <= 0:
		return errors.Errorf("invalid recovery TTL [%s]", m.config.TTL)
	case m.config.ScanInterval <= 0:
		return errors.Errorf("invalid recovery scan interval [%s]", m.config.ScanInterval)
	case m.config.BatchSize <= 0:
		return errors.Errorf("invalid recovery batch size [%d]", m.config.BatchSize)
	case m.config.WorkerCount <= 0:
		return errors.Errorf("invalid recovery worker count [%d]", m.config.WorkerCount)
	case m.config.LeaseDuration <= 0:
		return errors.Errorf("invalid recovery lease duration [%s]", m.config.LeaseDuration)
	default:
		return nil
	}
}

func (m *Manager) runSweep(ctx context.Context) error {
	leadership, acquired, err := m.storage.AcquireRecoveryLeadership(ctx, m.config.AdvisoryLockID)
	if err != nil {
		return errors.Wrapf(err, "failed to acquire recovery leadership")
	}
	if !acquired {
		m.logger.Debugf("recovery leadership not acquired")

		return nil
	}
	defer func() {
		if err := leadership.Close(); err != nil {
			m.logger.Warnf("failed to release recovery leadership: %v", err)
		}
	}()

	return m.recoverTransactions(ctx)
}

// recoverTransactions claims pending transactions and re-registers finality listeners using local workers.
func (m *Manager) recoverTransactions(ctx context.Context) error {
	m.logger.Debugf("claiming pending transactions older than %s (batch size: %d, lease duration: %s, owner: %s)",
		m.config.TTL, m.config.BatchSize, m.config.LeaseDuration, m.config.InstanceID)

	records, err := m.storage.ClaimPendingTransactions(
		ctx,
		m.config.TTL,
		m.config.LeaseDuration,
		m.config.BatchSize,
		m.config.InstanceID,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to claim pending transactions")
	}

	if len(records) == 0 {
		m.logger.Debugf("no pending transactions found needing recovery")

		return nil
	}

	m.logger.Infof("claimed %d pending transaction(s) needing recovery", len(records))

	work := make(chan string)
	errCh := make(chan error, len(records))
	var workerWG sync.WaitGroup

	for range m.config.WorkerCount {
		workerWG.Add(1)
		go m.worker(ctx, &workerWG, work, errCh)
	}

	for _, record := range records {
		if record == nil {
			continue
		}
		work <- record.TxID
	}
	close(work)

	workerWG.Wait()
	close(errCh)

	var firstErr error
	failures := 0
	for err := range errCh {
		if err == nil {
			continue
		}
		failures++
		if firstErr == nil {
			firstErr = err
		}
	}

	m.logger.Infof("completed recovery sweep: claimed=%d, failed=%d", len(records), failures)

	return firstErr
}

func (m *Manager) worker(ctx context.Context, wg *sync.WaitGroup, work <-chan string, errCh chan<- error) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case txID, ok := <-work:
			if !ok {
				return
			}
			if err := m.recoverTransaction(ctx, txID); err != nil {
				errCh <- err
			}
		}
	}
}

// recoverTransaction attempts to recover a transaction using the injected handler
// and always releases the claim with an appropriate message
func (m *Manager) recoverTransaction(ctx context.Context, txID string) error {
	m.logger.Debugf("recovering transaction [%s]", txID)

	// Attempt recovery using the injected handler
	err := m.handler.Recover(ctx, txID)

	// Always release the claim with appropriate message
	var message string
	if err != nil {
		message = fmt.Sprintf("recovery failed: %v", err)
	} else {
		message = "recovered successfully"
	}

	if releaseErr := m.releaseClaim(ctx, txID, message); releaseErr != nil {
		m.logger.Warnf("failed to release recovery claim for transaction [%s]: %v", txID, releaseErr)
	}

	if err != nil {
		return errors.Wrapf(err, "failed to recover transaction [%s]", txID)
	}

	m.logger.Infof("successfully recovered transaction [%s]", txID)

	return nil
}

func (m *Manager) releaseClaim(ctx context.Context, txID string, message string) error {
	return m.storage.ReleaseRecoveryClaim(ctx, txID, m.config.InstanceID, message)
}
