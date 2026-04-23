/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	scommon "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/postgres"
	q "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query"
	common3 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/cond"
	tokensdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	sqlcommon "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
)

// AuditTransactionStore is an alias for common.TransactionStore.
type AuditTransactionStore = sqlcommon.TransactionStore

// TransactionStore extends the common TransactionStore with PostgreSQL-specific atomic claim operations.
type TransactionStore struct {
	*sqlcommon.TransactionStore
	readDB  *sql.DB
	writeDB *sql.DB
	tables  sqlcommon.TableNames
}

// NewTransactionStoreWithNotifier creates a new TransactionStore with the provided notifier and recovery support.
func NewTransactionStoreWithNotifier(dbs *scommon.RWDB, tableNames sqlcommon.TableNames, notifier *TransactionNotifier) (*TransactionStore, error) {
	// Create recovery leader factory using PostgreSQL advisory locks
	recoveryLeaderFactory := NewAdvisoryLockFactory()

	commonStore, err := sqlcommon.NewTransactionStoreWithNotifierAndRecovery(
		dbs.ReadDB,
		dbs.WriteDB,
		tableNames,
		postgres.NewConditionInterpreter(),
		postgres.NewPaginationInterpreter(),
		notifier,
		recoveryLeaderFactory,
	)
	if err != nil {
		return nil, err
	}

	return &TransactionStore{
		TransactionStore: commonStore,
		readDB:           dbs.ReadDB,
		writeDB:          dbs.WriteDB,
		tables:           tableNames,
	}, nil
}

// NewAuditTransactionStore creates a new AuditTransactionStore.
func NewAuditTransactionStore(dbs *scommon.RWDB, tableNames sqlcommon.TableNames) (*AuditTransactionStore, error) {
	return sqlcommon.NewAuditTransactionStore(
		dbs.ReadDB,
		dbs.WriteDB,
		tableNames,
		postgres.NewConditionInterpreter(),
		postgres.NewPaginationInterpreter(),
	)
}

// ClaimPendingTransactions atomically claims a batch of pending transactions using PostgreSQL's UPDATE...RETURNING.
// This ensures only one recovery instance can claim a specific transaction.
func (db *TransactionStore) ClaimPendingTransactions(ctx context.Context, params tokensdriver.RecoveryClaimParams) ([]*tokensdriver.TransactionRecord, error) {
	logger.Debugf("Claiming pending transactions: owner=%s, olderThan=%s, limit=%d, lease=%s",
		params.Owner, params.OlderThan, params.Limit, params.LeaseDuration)

	// Build the atomic claim query using UPDATE...RETURNING with CTE
	// This query:
	// 1. Finds eligible pending transactions (old enough, not claimed or expired or owned by us)
	// 2. Atomically updates them with our claim
	// 3. Returns the full transaction details
	// #nosec G201
	query := fmt.Sprintf(`
		WITH claimed_txs AS (
			UPDATE %s
			SET
				recovery_claimed_by = $1,
				recovery_claim_expires_at = NOW() + $2::INTERVAL
			WHERE tx_id IN (
				SELECT r.tx_id
				FROM %s r
				INNER JOIN %s t ON r.tx_id = t.tx_id
				WHERE r.status = $3
				  AND t.stored_at < $4
				  AND (
					  r.recovery_claimed_by IS NULL
					  OR r.recovery_claim_expires_at < NOW()
					  OR r.recovery_claimed_by = $1
				  )
				ORDER BY t.stored_at ASC
				LIMIT $5
				FOR UPDATE SKIP LOCKED
			)
			RETURNING tx_id
		)
		SELECT
			t.tx_id, t.action_type, t.sender_eid, t.recipient_eid,
			t.token_type, t.amount, r.status, r.application_metadata,
			r.public_metadata, t.stored_at
		FROM claimed_txs c
		INNER JOIN %s t ON c.tx_id = t.tx_id
		INNER JOIN %s r ON c.tx_id = r.tx_id
		ORDER BY t.stored_at ASC`,
		db.tables.Requests,
		db.tables.Requests,
		db.tables.Transactions,
		db.tables.Transactions,
		db.tables.Requests,
	)

	// Convert lease duration to PostgreSQL interval format
	leaseInterval := fmt.Sprintf("%d seconds", int(params.LeaseDuration.Seconds()))

	args := []interface{}{
		params.Owner,
		leaseInterval,
		tokensdriver.Pending,
		params.OlderThan,
		params.Limit,
	}

	logger.Debug(query, args)

	// Execute the query
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to claim pending transactions")
	}

	// Parse results
	results := common.NewIterator(rows, func(r *tokensdriver.TransactionRecord) error {
		var amount sqlcommon.BigInt
		var appMeta []byte
		var pubMeta []byte
		if err := rows.Scan(&r.TxID, &r.ActionType, &r.SenderEID, &r.RecipientEID, &r.TokenType, &amount, &r.Status, &appMeta, &pubMeta, &r.Timestamp); err != nil {
			return err
		}
		r.Amount = amount.Int

		if err := unmarshalMetadata(appMeta, &r.ApplicationMetadata); err != nil {
			return err
		}

		return unmarshalMetadata(pubMeta, &r.PublicMetadata)
	})

	claimed, err := iterators.ReadAllPointers(results)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read claimed transactions")
	}

	logger.Infof("Claimed %d pending transactions for owner %s", len(claimed), params.Owner)

	return claimed, nil
}

// ReleaseRecoveryClaim releases the recovery claim on a transaction.
// This clears the claim metadata and optionally updates the status message.
func (db *TransactionStore) ReleaseRecoveryClaim(ctx context.Context, txID string, owner string, message string) error {
	logger.Debugf("Releasing recovery claim: txID=%s, owner=%s, message=%s", txID, owner, message)

	// Build the release query using query builder
	// Only release if the transaction is owned by the specified owner (safety check)
	var query string
	var args []interface{}

	if message != "" {
		query, args = q.Update(db.tables.Requests).
			Set("recovery_claimed_by", nil).
			Set("recovery_claim_expires_at", nil).
			Set("status_message", message).
			Where(cond.And(
				cond.Eq("tx_id", txID),
				cond.Eq("recovery_claimed_by", owner),
			)).
			Format(postgres.NewConditionInterpreter())
	} else {
		query, args = q.Update(db.tables.Requests).
			Set("recovery_claimed_by", nil).
			Set("recovery_claim_expires_at", nil).
			Where(cond.And(
				cond.Eq("tx_id", txID),
				cond.Eq("recovery_claimed_by", owner),
			)).
			Format(postgres.NewConditionInterpreter())
	}

	logger.Debug(query, args)

	result, err := db.writeDB.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrapf(err, "failed to release recovery claim for tx %s", txID)
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrapf(err, "failed to get rows affected for tx %s", txID)
	}

	if rowsAffected == 0 {
		logger.Warnf("No recovery claim released for tx %s (not owned by %s or already released)", txID, owner)
	} else {
		logger.Infof("Released recovery claim for tx %s", txID)
	}

	return nil
}

// CleanupExpiredClaims removes expired recovery claims.
// Returns the number of claims cleaned up.
func (db *TransactionStore) CleanupExpiredClaims(ctx context.Context) (int, error) {
	logger.Debug("Cleaning up expired recovery claims")

	query, args := q.Update(db.tables.Requests).
		Set("recovery_claimed_by", nil).
		Set("recovery_claim_expires_at", nil).
		Where(cond.Lt("recovery_claim_expires_at", common3.FieldName("NOW()"))).
		Format(postgres.NewConditionInterpreter())

	logger.Debug(query, args)

	result, err := db.writeDB.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to cleanup expired recovery claims")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get rows affected during cleanup")
	}

	logger.Infof("Cleaned up %d expired recovery claims", rowsAffected)

	return int(rowsAffected), nil
}

// TransactionNotifier handles notifications for transaction status changes.
type TransactionNotifier struct {
	*Notifier
}

// NewTransactionNotifier returns a new TransactionNotifier for the given RWDB and table names.
func NewTransactionNotifier(dbs *scommon.RWDB, tableNames sqlcommon.TableNames, dataSource string) (*TransactionNotifier, error) {
	return &TransactionNotifier{
		Notifier: NewNotifier(
			dbs.WriteDB,
			tableNames.Requests,
			dataSource,
			[]tokensdriver.Operation{tokensdriver.Update}, // Only listen to UPDATE operations for status changes
			*NewSimplePrimaryKey("tx_id"),
		),
	}, nil
}

// Subscribe registers a callback function to be called when a transaction request status is updated.
func (n *TransactionNotifier) Subscribe(callback func(tokensdriver.Operation, tokensdriver.TransactionRecordReference)) error {
	return n.Notifier.Subscribe(func(operation tokensdriver.Operation, m map[tokensdriver.ColumnKey]string) {
		callback(operation, tokensdriver.TransactionRecordReference{
			TxID: m["tx_id"],
		})
	})
}

// unmarshalMetadata unmarshals JSON metadata.
func unmarshalMetadata(in []byte, out *map[string][]byte) error {
	if len(in) == 0 {
		return nil
	}

	return json.Unmarshal(in, out)
}
