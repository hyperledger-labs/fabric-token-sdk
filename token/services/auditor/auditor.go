/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditor

import (
	"context"
	"math"
	"math/rand/v2"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/finality"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"go.opentelemetry.io/otel/trace"
)

var logger = logging.MustGetLogger()

//go:generate counterfeiter -o mock/transaction.go -fake-name Transaction . Transaction
//go:generate counterfeiter -o mock/network_provider.go -fake-name NetworkProvider . NetworkProvider
//go:generate counterfeiter -o mock/check_service.go -fake-name CheckService . CheckService
//go:generate counterfeiter -o mock/network_driver.go -fake-name Network github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver.Network
//go:generate counterfeiter -o mock/audit_transaction_store.go -fake-name AuditTransactionStore github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver.AuditTransactionStore
//go:generate counterfeiter -o mock/tst.go -fake-name TransactionStoreTransaction github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver.TransactionStoreTransaction

// TxStatus is the status of a transaction
type TxStatus = auditdb.TxStatus

const (
	// Pending is the status of a transaction that has been submitted to the ledger
	Pending = auditdb.Pending
	// Confirmed is the status of a transaction that has been confirmed by the ledger
	Confirmed = auditdb.Confirmed
	// Deleted is the status of a transaction that has been deleted due to a failure to commit
	Deleted = auditdb.Deleted
	// Orphan is the status of a transaction that never reached the ledger
	Orphan = auditdb.Orphan
)

const txIdLabel tracing.LabelName = "tx_id"

var TxStatusMessage = auditdb.TxStatusMessage

// Transaction models a generic token transaction
type Transaction interface {
	ID() string
	Network() string
	Channel() string
	Namespace() string
	Request() *token.Request
}

type NetworkProvider interface {
	GetNetwork(network string, channel string) (*network.Network, error)
}

type CheckService interface {
	Check(ctx context.Context) ([]string, error)
}

const (
	// Default lock acquisition retry configuration constants
	defaultMaxLockRetries        = 10                    // Maximum number of retry attempts
	defaultInitialLockBackoff    = 10 * time.Millisecond // Initial backoff delay
	defaultMaxLockBackoff        = 5 * time.Second       // Maximum backoff delay
	defaultLockBackoffMultiplier = 2.0                   // Exponential backoff multiplier
	defaultLockJitterFactor      = 0.3                   // Randomization factor (30%)
)

// LockConfig holds the configuration for lock acquisition retry logic
type LockConfig struct {
	// MaxRetries is the maximum number of retry attempts for lock acquisition
	MaxRetries int
	// InitialBackoff is the initial backoff delay before the first retry
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff delay between retries
	MaxBackoff time.Duration
	// BackoffMultiplier is the exponential backoff multiplier
	BackoffMultiplier float64
	// JitterFactor is the randomization factor to prevent thundering herd (0.0 to 1.0)
	JitterFactor float64
}

// DefaultLockConfig returns the default lock configuration
func DefaultLockConfig() *LockConfig {
	return &LockConfig{
		MaxRetries:        defaultMaxLockRetries,
		InitialBackoff:    defaultInitialLockBackoff,
		MaxBackoff:        defaultMaxLockBackoff,
		BackoffMultiplier: defaultLockBackoffMultiplier,
		JitterFactor:      defaultLockJitterFactor,
	}
}

// lockConfigRaw is used to unmarshal lock configuration from YAML
type lockConfigRaw struct {
	MaxRetries        int     `yaml:"maxRetries"`
	InitialBackoff    string  `yaml:"initialBackoff"`
	MaxBackoff        string  `yaml:"maxBackoff"`
	BackoffMultiplier float64 `yaml:"backoffMultiplier"`
	JitterFactor      float64 `yaml:"jitterFactor"`
}

// LoadLockConfig loads lock configuration from the configuration provider.
// If configuration is not found or invalid, returns default configuration.
func LoadLockConfig(cp *config.Configuration) *LockConfig {
	cfg := DefaultLockConfig()

	if !cp.IsSet("auditor.lock") {
		return cfg
	}

	var raw lockConfigRaw
	if err := cp.UnmarshalKey("auditor.lock", &raw); err != nil {
		logger.Warnf("failed to unmarshal auditor lock configuration, using defaults: %v", err)

		return cfg
	}

	// Apply max retries if valid
	if raw.MaxRetries > 0 {
		cfg.MaxRetries = raw.MaxRetries
	}

	// Apply initial backoff if valid
	if raw.InitialBackoff != "" {
		if duration, err := time.ParseDuration(raw.InitialBackoff); err == nil && duration > 0 {
			cfg.InitialBackoff = duration
		} else {
			logger.Warnf("invalid initialBackoff value [%s], using default", raw.InitialBackoff)
		}
	}

	// Apply max backoff if valid
	if raw.MaxBackoff != "" {
		if duration, err := time.ParseDuration(raw.MaxBackoff); err == nil && duration > 0 {
			cfg.MaxBackoff = duration
		} else {
			logger.Warnf("invalid maxBackoff value [%s], using default", raw.MaxBackoff)
		}
	}

	// Apply backoff multiplier if valid
	if raw.BackoffMultiplier > 0 {
		cfg.BackoffMultiplier = raw.BackoffMultiplier
	}

	// Apply jitter factor if valid (must be between 0 and 1)
	if raw.JitterFactor >= 0 && raw.JitterFactor <= 1.0 {
		cfg.JitterFactor = raw.JitterFactor
	}

	return cfg
}

// Service is the interface for the auditor service
type Service struct {
	tmsID           token.TMSID
	networkProvider NetworkProvider
	auditDB         *auditdb.StoreService
	tokenDB         *tokens.Service
	tmsProvider     dep.TokenManagementServiceProvider
	finalityTracer  trace.Tracer
	metricsProvider metrics.Provider
	metrics         *Metrics
	checkService    CheckService
	lockConfig      *LockConfig
}

// NewService creates a new auditor Service with the provided dependencies.
// If lockConfig is nil, default lock configuration will be used.
func NewService(
	tmsID token.TMSID,
	networkProvider NetworkProvider,
	auditDB *auditdb.StoreService,
	tokenDB *tokens.Service,
	tmsProvider dep.TokenManagementServiceProvider,
	finalityTracer trace.Tracer,
	metricsProvider metrics.Provider,
	checkService CheckService,
	lockConfig *LockConfig,
) *Service {
	if lockConfig == nil {
		lockConfig = DefaultLockConfig()
	}

	return &Service{
		tmsID:           tmsID,
		networkProvider: networkProvider,
		auditDB:         auditDB,
		tokenDB:         tokenDB,
		tmsProvider:     tmsProvider,
		finalityTracer:  finalityTracer,
		metricsProvider: metricsProvider,
		metrics:         newMetrics(metricsProvider),
		checkService:    checkService,
		lockConfig:      lockConfig,
	}
}

// Validate validates the passed token request
func (a *Service) Validate(ctx context.Context, request *token.Request) error {
	return request.AuditCheck(ctx)
}

// Audit extracts the list of inputs and outputs from the passed transaction.
// In addition, the Audit locks the enrollment named ids with retry logic and exponential backoff
// to prevent livelock conditions.
// The caller MUST call Release() to unlock these enrollment IDs after processing.
//
// IMPORTANT: The defer Release() statement MUST be placed immediately after checking
// the error returned by Audit(). This ensures locks are released even if subsequent
// operations fail. Example:
//
//	inputs, outputs, err := auditor.Audit(ctx, tx)
//	if err != nil {
//	    return errors.Wrap(err, "audit failed")
//	}
//	defer auditor.Release(ctx, tx)
//
// Note: The semaphore-based locking mechanism handles context cancellation during
// lock acquisition (see PR #1616), ensuring proper cleanup in case of timeouts or
// cancellations.
func (a *Service) Audit(ctx context.Context, tx Transaction) (*token.InputStream, *token.OutputStream, error) {
	start := time.Now()
	logger.DebugfContext(ctx, "audit transaction [%s]....", tx.ID())
	request := tx.Request()
	record, err := request.AuditRecord(ctx)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed getting transaction audit record")
	}

	var eids []string
	eids = append(eids, record.Inputs.EnrollmentIDs()...)
	eids = append(eids, record.Outputs.EnrollmentIDs()...)

	// Acquire locks with retry and exponential backoff to prevent livelock
	logger.DebugfContext(ctx, "audit transaction [%s], acquire locks with retry", tx.ID())
	if err := a.acquireLocksWithRetry(ctx, string(request.Anchor), eids); err != nil {
		a.metrics.AuditLockConflicts.Add(1)

		return nil, nil, err
	}

	logger.DebugfContext(ctx, "audit transaction [%s], acquire locks done", tx.ID())
	a.metrics.AuditDuration.Observe(time.Since(start).Seconds())

	return record.Inputs, record.Outputs, nil
}

// acquireLocksWithRetry attempts to acquire locks with exponential backoff and randomized jitter
// to prevent livelock conditions when multiple auditors compete for the same enrollment IDs.
// This implements the mitigation strategy for deadlock/livelock prevention.
func (a *Service) acquireLocksWithRetry(ctx context.Context, anchor string, eids []string) error {
	var lastErr error

	for attempt := range a.lockConfig.MaxRetries {
		// Attempt to acquire locks
		err := a.auditDB.AcquireLocks(ctx, anchor, eids...)
		if err == nil {
			// Success
			if attempt > 0 {
				logger.DebugfContext(ctx, "Lock acquisition succeeded on attempt %d for anchor [%s]", attempt+1, anchor)
			}

			return nil
		}

		lastErr = err

		// Check if context is cancelled - don't retry if so
		if ctx.Err() != nil {
			return errors.WithMessagef(ctx.Err(), "lock acquisition cancelled after %d attempts for anchor [%s]", attempt+1, anchor)
		}

		// Calculate backoff with exponential growth and randomized jitter
		backoff := a.calculateBackoff(attempt)

		logger.DebugfContext(ctx, "Lock acquisition failed (attempt %d/%d) for anchor [%s], retrying after %v: %v",
			attempt+1, a.lockConfig.MaxRetries, anchor, backoff, err)

		// Wait with context cancellation support
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()

			return errors.WithMessagef(ctx.Err(), "lock acquisition cancelled during backoff after %d attempts for anchor [%s]", attempt+1, anchor)
		case <-timer.C:
			// Continue to next retry attempt
		}
	}

	return errors.WithMessagef(lastErr, "failed to acquire locks after %d attempts for anchor [%s]", a.lockConfig.MaxRetries, anchor)
}

// calculateBackoff computes the backoff duration with exponential growth and randomized jitter.
// The jitter breaks livelock symmetry when multiple auditors retry simultaneously.
func (a *Service) calculateBackoff(attempt int) time.Duration {
	// Calculate base delay with exponential growth
	delay := float64(a.lockConfig.InitialBackoff) * math.Pow(a.lockConfig.BackoffMultiplier, float64(attempt))

	// Cap at maximum delay
	if delay > float64(a.lockConfig.MaxBackoff) {
		delay = float64(a.lockConfig.MaxBackoff)
	}

	// Add randomized jitter to break symmetry
	// Jitter range: delay * (1 - jitterFactor/2) to delay * (1 + jitterFactor/2)
	jitterRange := delay * a.lockConfig.JitterFactor
	jitter := (rand.Float64() - 0.5) * jitterRange

	finalDelay := time.Duration(delay + jitter)

	// Ensure non-negative delay
	if finalDelay < 0 {
		finalDelay = a.lockConfig.InitialBackoff
	}

	return finalDelay
}

// Append adds the passed transaction to the auditor database.
// It also releases the locks acquired by Audit.
func (a *Service) Append(ctx context.Context, tx Transaction) error {
	start := time.Now()
	defer func() { a.metrics.AppendDuration.Observe(time.Since(start).Seconds()) }()
	defer a.Release(ctx, tx)

	tms, err := a.tmsProvider.TokenManagementService(token.WithTMSID(a.tmsID))
	if err != nil {
		return err
	}
	// append request to audit db
	if err := a.auditDB.Append(ctx, newRequestWrapper(tx.Request(), tms)); err != nil {
		a.metrics.AppendErrors.Add(1)

		return errors.WithMessagef(err, "failed appending request %s", tx.ID())
	}

	// lister to events
	net, err := a.networkProvider.GetNetwork(tx.Network(), tx.Channel())
	if err != nil {
		return errors.WithMessagef(err, "failed getting network instance for [%s:%s]", tx.Network(), tx.Channel())
	}
	logger.DebugfContext(ctx, "register tx status listener for tx [%s] at network [%s]", tx.ID(), tx.Network())
	var r driver.FinalityListener = finality.NewListener(
		logger,
		net,
		tx.Namespace(),
		finality.NewTokenRequestHasher(a.tmsProvider, a.tmsID),
		a.auditDB,
		a.tokenDB,
		a.finalityTracer,
		a.metricsProvider,
	)
	if err := net.AddFinalityListener(tx.Namespace(), tx.ID(), r); err != nil {
		return errors.WithMessagef(err, "failed listening to network [%s:%s]", tx.Network(), tx.Channel())
	}
	logger.DebugfContext(ctx, "append done for request [%s]", tx.ID())

	return nil
}

// Release releases the lock acquired of the passed transaction.
func (a *Service) Release(ctx context.Context, tx Transaction) {
	a.metrics.ReleasesTotal.Add(1)
	a.auditDB.ReleaseLocks(ctx, string(tx.Request().Anchor))
}

// SetStatus sets the status of the audit records with the passed transaction id to the passed status
func (a *Service) SetStatus(ctx context.Context, txID string, status storage.TxStatus, message string) error {
	return a.auditDB.SetStatus(ctx, txID, status, message)
}

// GetStatus return the status of the given transaction id.
// It returns an error if no transaction with that id is found
func (a *Service) GetStatus(ctx context.Context, txID string) (TxStatus, string, error) {
	return a.auditDB.GetStatus(ctx, txID)
}

// GetTokenRequest returns the token request bound to the passed transaction id, if available.
func (a *Service) GetTokenRequest(ctx context.Context, txID string) ([]byte, error) {
	return a.auditDB.GetTokenRequest(ctx, txID)
}

// Check performs a health check on the auditor service and returns any issues found.
func (a *Service) Check(ctx context.Context) ([]string, error) {
	return a.checkService.Check(ctx)
}

type requestWrapper struct {
	r   *token.Request
	tms dep.TokenManagementService
}

// newRequestWrapper creates a new requestWrapper that wraps a token request with its associated
// token management service for enhanced audit record processing.
func newRequestWrapper(r *token.Request, tms dep.TokenManagementService) *requestWrapper {
	return &requestWrapper{r: r, tms: tms}
}

// ID returns the unique identifier (anchor) of the wrapped token request.
func (r *requestWrapper) ID() token.RequestAnchor {
	return r.r.ID()
}

// Bytes returns the serialized byte representation of the wrapped token request.
func (r *requestWrapper) Bytes() ([]byte, error) { return r.r.Bytes() }

// AllApplicationMetadata returns all application-specific metadata associated with the token request.
func (r *requestWrapper) AllApplicationMetadata() map[string][]byte {
	return r.r.AllApplicationMetadata()
}

// PublicParamsHash returns the hash of the public parameters used in the token request.
func (r *requestWrapper) PublicParamsHash() token.PPHash { return r.r.PublicParamsHash() }

// AuditRecord retrieves the audit record for the wrapped token request and completes any
// inputs with missing enrollment IDs by querying the token vault.
func (r *requestWrapper) AuditRecord(ctx context.Context) (*token.AuditRecord, error) {
	record, err := r.r.AuditRecord(ctx)
	if err != nil {
		return nil, err
	}
	if err := r.completeInputsWithEmptyEID(ctx, record); err != nil {
		return nil, errors.WithMessagef(err, "failed filling gaps for request [%s]", r.r.Anchor)
	}

	return record, nil
}

// completeInputsWithEmptyEID fills in missing enrollment ID information for inputs in the audit record
// by querying the token vault. This is necessary when inputs don't have enrollment IDs explicitly set.
// It uses the first output's enrollment ID as the target and retrieves token details from the vault.
func (r *requestWrapper) completeInputsWithEmptyEID(ctx context.Context, record *token.AuditRecord) error {
	filter := record.Inputs.ByEnrollmentID("")
	if filter.Count() == 0 {
		return nil
	}
	// TODO: extract from the audit tokens
	targetEID := record.Outputs.EnrollmentIDs()[0]

	// fetch all the tokens
	tokens, err := r.tms.Vault().NewQueryEngine().ListAuditTokens(ctx, filter.IDs()...)
	if err != nil {
		return errors.WithMessagef(err, "failed listing tokens for [%s]", filter.IDs())
	}
	precision := r.tms.PublicParametersManager().PublicParameters().Precision()
	for i := range filter.Count() {
		item := filter.At(i)
		item.EnrollmentID = targetEID
		item.Owner = tokens[i].Owner
		item.Type = tokens[i].Type
		q, err := token2.ToQuantity(tokens[i].Quantity, precision)
		if err != nil {
			return errors.WithMessagef(err, "failed converting token quantity [%s]", tokens[i].Quantity)
		}
		item.Quantity = q
	}

	return nil
}

// String returns a string representation of the wrapped token request.
func (r *requestWrapper) String() string {
	return r.r.String()
}
