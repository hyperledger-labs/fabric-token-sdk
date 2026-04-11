/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package recovery

import (
	"context"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep"
)

// Config holds the configuration for the recovery manager
type Config struct {
	// Enabled indicates whether transaction recovery is enabled
	Enabled bool
	// TTL is the time-to-live for transactions before they are considered for recovery
	TTL time.Duration
	// ScanInterval is how often to scan for transactions needing recovery
	ScanInterval time.Duration
}

// DefaultConfig returns the default recovery configuration
func DefaultConfig() Config {
	return Config{
		Enabled:      false, // Disabled by default for backward compatibility
		TTL:          30 * time.Second,
		ScanInterval: 30 * time.Second,
	}
}

// TTXDatabase defines the interface for querying pending transactions and transaction details
type TTXDatabase interface {
	QueryPendingTransactions(ctx context.Context, olderThan time.Duration) ([]*ttxdb.TransactionRecord, error)
	GetTokenRequest(ctx context.Context, txID string) ([]byte, error)
	SetStatus(ctx context.Context, txID string, status storage.TxStatus, message string) error
}

// FinalityListenerFactory creates finality listeners for transaction recovery
type FinalityListenerFactory interface {
	// NewFinalityListener creates a new finality listener for the given transaction ID
	NewFinalityListener(txID string) (network.FinalityListener, error)
}

// Manager handles the recovery of transactions that may have lost their finality listeners
type Manager struct {
	logger          logging.Logger
	ttxDB           TTXDatabase
	network         dep.Network
	namespace       string
	listenerFactory FinalityListenerFactory
	config          Config
	recoveredTxs    sync.Map // map[string]bool - tracks recovered transaction IDs
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	started         bool
	mu              sync.Mutex
}

// NewManager creates a new recovery manager
func NewManager(
	logger logging.Logger,
	ttxDB TTXDatabase,
	network dep.Network,
	namespace string,
	listenerFactory FinalityListenerFactory,
	config Config,
) *Manager {
	return &Manager{
		logger:          logger,
		ttxDB:           ttxDB,
		network:         network,
		namespace:       namespace,
		listenerFactory: listenerFactory,
		config:          config,
	}
}

// Start begins the recovery process
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.config.Enabled {
		m.logger.Debugf("transaction recovery is disabled for namespace [%s]", m.namespace)

		return nil
	}

	if m.started {
		return errors.Errorf("recovery manager already started for namespace [%s]", m.namespace)
	}

	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.started = true

	m.wg.Add(1)
	go m.recoveryLoop()

	m.logger.Infof("transaction recovery manager started for namespace [%s] (TTL: %s, Scan Interval: %s)",
		m.namespace, m.config.TTL, m.config.ScanInterval)

	return nil
}

// Stop gracefully stops the recovery process
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return nil
	}

	m.logger.Infof("stopping transaction recovery manager for namespace [%s]", m.namespace)
	m.cancel()
	m.wg.Wait()
	m.started = false
	m.logger.Infof("transaction recovery manager stopped for namespace [%s]", m.namespace)

	return nil
}

// recoveryLoop is the main loop that periodically scans for transactions needing recovery
func (m *Manager) recoveryLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.ScanInterval)
	defer ticker.Stop()

	// Perform an initial scan immediately
	if err := m.recoverTransactions(m.ctx); err != nil {
		m.logger.Warnf("initial transaction recovery scan failed for namespace [%s]: %v", m.namespace, err)
	}

	for {
		select {
		case <-m.ctx.Done():
			m.logger.Debugf("recovery loop stopped for namespace [%s]", m.namespace)

			return
		case <-ticker.C:
			if err := m.recoverTransactions(m.ctx); err != nil {
				m.logger.Warnf("transaction recovery scan failed for namespace [%s]: %v", m.namespace, err)
			}
		}
	}
}

// recoverTransactions queries for pending transactions and re-registers finality listeners
func (m *Manager) recoverTransactions(ctx context.Context) error {
	m.logger.Debugf("scanning for pending transactions older than %s in namespace [%s]", m.config.TTL, m.namespace)

	// Query for pending transactions older than TTL
	records, err := m.ttxDB.QueryPendingTransactions(ctx, m.config.TTL)
	if err != nil {
		return errors.Wrapf(err, "failed to query pending transactions")
	}

	if len(records) == 0 {
		m.logger.Debugf("no pending transactions found needing recovery in namespace [%s]", m.namespace)

		return nil
	}

	m.logger.Infof("found %d pending transaction(s) needing recovery in namespace [%s]", len(records), m.namespace)

	// Group records by transaction ID (multiple records can have the same txID)
	txIDs := make(map[string]bool)
	for _, record := range records {
		txIDs[record.TxID] = true
	}

	// Recover each unique transaction
	recovered := 0
	for txID := range txIDs {
		// Skip if already recovered
		if _, alreadyRecovered := m.recoveredTxs.Load(txID); alreadyRecovered {
			m.logger.Debugf("transaction [%s] already recovered, skipping", txID)

			continue
		}

		if err := m.recoverTransaction(ctx, txID); err != nil {
			m.logger.Warnf("failed to recover transaction [%s]: %v", txID, err)

			continue
		}

		// Mark as recovered
		m.recoveredTxs.Store(txID, true)
		recovered++
	}

	m.logger.Infof("successfully recovered %d transaction(s) in namespace [%s]", recovered, m.namespace)

	return nil
}

// recoverTransaction re-registers the finality listener for a specific transaction
func (m *Manager) recoverTransaction(ctx context.Context, txID string) error {
	m.logger.Debugf("recovering transaction [%s] in namespace [%s]", txID, m.namespace)

	// Create a new finality listener using the factory
	listener, err := m.listenerFactory.NewFinalityListener(txID)
	if err != nil {
		return errors.Wrapf(err, "failed to create finality listener for transaction [%s]", txID)
	}

	// Wrap the listener to clean up recoveredTxs when finality is reached
	wrappedListener := &recoveryListenerWrapper{
		delegate:  listener,
		manager:   m,
		txID:      txID,
		logger:    m.logger,
		namespace: m.namespace,
	}

	// Re-register the wrapped finality listener
	if err := m.network.AddFinalityListener(m.namespace, txID, wrappedListener); err != nil {
		return errors.Wrapf(err, "failed to add finality listener for transaction [%s]", txID)
	}

	m.logger.Infof("successfully recovered transaction [%s] in namespace [%s]", txID, m.namespace)

	return nil
}

// recoveryListenerWrapper wraps a finality listener to clean up the recoveredTxs map
// when the transaction reaches a final state
type recoveryListenerWrapper struct {
	delegate  network.FinalityListener
	manager   *Manager
	txID      string
	logger    logging.Logger
	namespace string
}

// OnStatus delegates to the wrapped listener and cleans up on final status
func (w *recoveryListenerWrapper) OnStatus(ctx context.Context, txID string, status int, message string, tokenRequestHash []byte) {
	// Delegate to the original listener
	w.delegate.OnStatus(ctx, txID, status, message, tokenRequestHash)

	// Check if this is a final status (Valid or Invalid)
	// Valid = 0, Invalid = 1 (from network package constants)
	if status == network.Valid || status == network.Invalid {
		// Transaction reached finality, remove from recovered set
		w.manager.recoveredTxs.Delete(w.txID)
		w.logger.Debugf("removed transaction [%s] from recovered set after reaching finality in namespace [%s]", w.txID, w.namespace)
	}
}

// OnError delegates to the wrapped listener
func (w *recoveryListenerWrapper) OnError(ctx context.Context, txID string, err error) {
	w.delegate.OnError(ctx, txID, err)
}
