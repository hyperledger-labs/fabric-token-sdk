/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"database/sql"
	"encoding/json"
	errors2 "errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/hashicorp/go-uuid"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	driver3 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/common"
	qcommon "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/pagination"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
)

// maxAmountBits is the maximum bit length supported by NUMERIC(78, 0).
// NUMERIC(78, 0) stores up to 10^78 values, fitting in ~259 bits.
// 255 is used as a conservative safe upper bound.
const maxAmountBits = 255

type transactionTables struct {
	Movements             string
	Transactions          string
	Requests              string
	Validations           string
	TransactionEndorseAck string
}

type TransactionStore struct {
	readDB                *sql.DB
	writeDB               *sql.DB
	table                 transactionTables
	pf                    sq.PlaceholderFormat
	notifier              dbdriver.TransactionNotifier
	recoveryLeaderFactory func(context.Context, *sql.DB, int64) (dbdriver.RecoveryLeadership, bool, error)
}

func newTransactionStore(
	readDB, writeDB *sql.DB,
	tables transactionTables,
	pf sq.PlaceholderFormat,
	notifier dbdriver.TransactionNotifier,
	recoveryLeaderFactory func(context.Context, *sql.DB, int64) (dbdriver.RecoveryLeadership, bool, error),
) *TransactionStore {
	return &TransactionStore{
		readDB:                readDB,
		writeDB:               writeDB,
		table:                 tables,
		pf:                    pf,
		notifier:              notifier,
		recoveryLeaderFactory: recoveryLeaderFactory,
	}
}

func NewAuditTransactionStore(readDB, writeDB *sql.DB, tables TableNames, pf sq.PlaceholderFormat) (*TransactionStore, error) {
	return NewOwnerTransactionStore(readDB, writeDB, tables, pf)
}

func NewOwnerTransactionStore(readDB, writeDB *sql.DB, tables TableNames, pf sq.PlaceholderFormat) (*TransactionStore, error) {
	return newTransactionStore(readDB, writeDB, transactionTables{
		Movements:             tables.Movements,
		Transactions:          tables.Transactions,
		Requests:              tables.Requests,
		Validations:           tables.Validations,
		TransactionEndorseAck: tables.TransactionEndorseAck,
	}, pf, nil, nil), nil
}

func NewTransactionStoreWithNotifierAndRecovery(
	readDB, writeDB *sql.DB,
	tables TableNames,
	pf sq.PlaceholderFormat,
	notifier dbdriver.TransactionNotifier,
	recoveryLeaderFactory func(context.Context, *sql.DB, int64) (dbdriver.RecoveryLeadership, bool, error),
) (*TransactionStore, error) {
	return newTransactionStore(readDB, writeDB, transactionTables{
		Movements:             tables.Movements,
		Transactions:          tables.Transactions,
		Requests:              tables.Requests,
		Validations:           tables.Validations,
		TransactionEndorseAck: tables.TransactionEndorseAck,
	}, pf, notifier, recoveryLeaderFactory), nil
}

func (db *TransactionStore) CreateSchema() error {
	return common.InitSchema(db.writeDB, db.GetSchema())
}

func (db *TransactionStore) GetTokenRequest(ctx context.Context, txID string) ([]byte, error) {
	query, args, err := sq.Select("request").
		From(db.table.Requests).
		Where(sq.Eq{"tx_id": txID}).
		PlaceholderFormat(db.pf).
		ToSql()
	if err != nil {
		return nil, err
	}

	return common.QueryUniqueContext[[]byte](ctx, db.readDB, query, args...)
}

// GetTokenRequests fetches the token requests for the given tx ids in a
// single SELECT. Missing tx ids are absent from the returned map — callers
// should treat a missing key identically to GetTokenRequest returning nil.
// Empty input returns an empty map without querying.
func (db *TransactionStore) GetTokenRequests(ctx context.Context, txIDs []string) (map[string][]byte, error) {
	if len(txIDs) == 0 {
		return map[string][]byte{}, nil
	}
	query, args, err := sq.Select("tx_id", "request").
		From(db.table.Requests).
		Where(sq.Eq{"tx_id": txIDs}).
		PlaceholderFormat(db.pf).
		ToSql()
	if err != nil {
		return nil, err
	}

	logging.Debug(logger, query, args)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer Close(rows)

	result := make(map[string][]byte, len(txIDs))
	for rows.Next() {
		var txID string
		var request []byte
		if err := rows.Scan(&txID, &request); err != nil {
			return nil, err
		}
		result[txID] = request
	}

	return result, rows.Err()
}

func (db *TransactionStore) QueryMovements(ctx context.Context, params dbdriver.QueryMovementsParams) (res []*dbdriver.MovementRecord, err error) {
	b := sq.Select(
		db.table.Movements+".tx_id",
		"enrollment_id",
		"token_type",
		"amount",
		db.table.Requests+".status",
	).
		From(db.table.Movements).
		LeftJoin(db.table.Requests + " ON " + db.table.Movements + ".tx_id = " + db.table.Requests + ".tx_id").
		Where(HasMovementsParams(params)).
		OrderBy(db.table.Movements + ".stored_at " + directionStr(params.SearchDirection)).
		PlaceholderFormat(db.pf)

	if params.NumRecords > 0 {
		b = b.Limit(uint64(params.NumRecords))
	}

	query, args, err := b.ToSql()
	if err != nil {
		return nil, err
	}

	logging.Debug(logger, query, args)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	it := common.NewIterator(rows, func(r *dbdriver.MovementRecord) error {
		var amount BigInt
		if err := rows.Scan(&r.TxID, &r.EnrollmentID, &r.TokenType, &amount, &r.Status); err != nil {
			return err
		}
		r.Amount = amount.Int
		logger.DebugfContext(ctx, "movement [%s:%s:%s]", r.TxID, r.Status, r.Amount)

		return nil
	})

	return iterators.ReadAllPointers(it)
}

func (db *TransactionStore) QueryTransactions(ctx context.Context, params dbdriver.QueryTransactionsParams, p driver3.Pagination) (*driver3.PageIterator[*dbdriver.TransactionRecord], error) {
	b := sq.Select(
		db.table.Transactions+".tx_id",
		"action_type",
		"sender_eid",
		"recipient_eid",
		"token_type",
		"amount",
		db.table.Requests+".status",
		db.table.Requests+".application_metadata",
		db.table.Requests+".public_metadata",
		db.table.Transactions+".stored_at",
	).
		From(db.table.Transactions).
		LeftJoin(db.table.Requests + " ON " + db.table.Transactions + ".tx_id = " + db.table.Requests + ".tx_id").
		OrderBy(db.table.Transactions + ".stored_at " + directionStr(params.SearchDirection)).
		PlaceholderFormat(db.pf)

	if where := HasTransactionParams(params, db.table.Transactions); where != nil {
		b = b.Where(where)
	}
	b = applyPagination(b, p)

	query, args, err := b.ToSql()
	if err != nil {
		return nil, err
	}

	logging.Debug(logger, query, args)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	results := common.NewIterator(rows, func(r *dbdriver.TransactionRecord) error {
		var amount BigInt
		var appMeta []byte
		var pubMeta []byte
		if err := rows.Scan(&r.TxID, &r.ActionType, &r.SenderEID, &r.RecipientEID, &r.TokenType, &amount, &r.Status, &appMeta, &pubMeta, &r.Timestamp); err != nil {
			return err
		}
		r.Amount = amount.Int

		return errors2.Join(
			unmarshal(appMeta, &r.ApplicationMetadata),
			unmarshal(pubMeta, &r.PublicMetadata),
		)
	})

	return &driver3.PageIterator[*dbdriver.TransactionRecord]{
		Items:      results,
		Pagination: p,
	}, nil
}

func (db *TransactionStore) GetStatus(ctx context.Context, txID string) (dbdriver.TxStatus, string, error) {
	var status dbdriver.TxStatus
	var statusMessage string
	query, args, err := sq.Select("status", "status_message").
		From(db.table.Requests).
		Where(sq.Eq{"tx_id": txID}).
		PlaceholderFormat(db.pf).
		ToSql()
	if err != nil {
		return dbdriver.Unknown, "", err
	}
	logging.Debug(logger, query, txID)

	row := db.readDB.QueryRowContext(ctx, query, args...)
	if err := row.Scan(&status, &statusMessage); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.DebugfContext(ctx, "tried to get status for non-existent tx [%s], returning unknown", txID)

			return dbdriver.Unknown, "", nil
		}

		return dbdriver.Unknown, "", errors.Wrapf(err, "error querying db")
	}

	return status, statusMessage, nil
}

func (db *TransactionStore) QueryValidations(ctx context.Context, params dbdriver.QueryValidationRecordsParams) (dbdriver.ValidationRecordsIterator, error) {
	b := sq.Select(
		db.table.Validations+".tx_id",
		db.table.Requests+".request",
		"metadata",
		db.table.Requests+".status",
		db.table.Validations+".stored_at",
	).
		From(db.table.Validations).
		LeftJoin(db.table.Requests + " ON " + db.table.Validations + ".tx_id = " + db.table.Requests + ".tx_id").
		PlaceholderFormat(db.pf)

	if where := HasValidationParams(params, db.table.Validations); where != nil {
		b = b.Where(where)
	}

	query, args, err := b.ToSql()
	if err != nil {
		return nil, err
	}

	logging.Debug(logger, query, args)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	it := common.NewIterator(rows, func(r *dbdriver.ValidationRecord) error {
		var meta []byte
		if err := rows.Scan(&r.TxID, &r.TokenRequest, &meta, &r.Status, &r.Timestamp); err != nil {
			return err
		}

		return unmarshal(meta, &r.Metadata)
	})
	if params.Filter == nil {
		return it, nil
	}

	return iterators.Filter(it, params.Filter), nil
}

func (db *TransactionStore) Notifier() (dbdriver.TransactionNotifier, error) {
	if db.notifier == nil {
		return nil, storage.ErrNotSupported
	}

	return db.notifier, nil
}

// QueryTokenRequests returns an iterator over the token requests matching the passed params
func (db *TransactionStore) QueryTokenRequests(ctx context.Context, params dbdriver.QueryTokenRequestsParams) (dbdriver.TokenRequestIterator, error) {
	b := sq.Select("tx_id", "request", "status").
		From(db.table.Requests).
		PlaceholderFormat(db.pf)

	if len(params.Statuses) > 0 {
		b = b.Where(sq.Eq{"status": params.Statuses})
	}

	query, args, err := b.ToSql()
	if err != nil {
		return nil, err
	}

	logging.Debug(logger, query, args)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	// TODO: AF remove r.TokenRequest. Not used
	return common.NewIterator(rows, func(r *dbdriver.TokenRequestRecord) error { return rows.Scan(&r.TxID, &r.TokenRequest, &r.Status) }), nil
}

// AcquireRecoveryLeadership returns a leadership handle for recovery sweeping.
// When no leader factory is configured, leadership is granted locally.
func (db *TransactionStore) AcquireRecoveryLeadership(ctx context.Context, lockID int64) (dbdriver.RecoveryLeadership, bool, error) {
	if db.recoveryLeaderFactory == nil {
		return noopRecoveryLeadership{}, true, nil
	}

	return db.recoveryLeaderFactory(ctx, db.writeDB, lockID)
}

// ClaimPendingTransactions returns a claimed batch of Pending transactions.
// The default SQL implementation is permissive and does not persist recovery claims.
// tx_id and stored_at are projected directly from the requests table — the
// transactions table is no longer joined since it carries no information
// the recovery loop needs and adding it would re-introduce a fan-out by
// movement/output that the caller would have to dedupe by tx_id.
func (db *TransactionStore) ClaimPendingTransactions(ctx context.Context, params dbdriver.RecoveryClaimParams) ([]*dbdriver.RecoveryClaim, error) {
	query, args, err := sq.Select("tx_id", "stored_at").
		From(db.table.Requests).
		Where(sq.And{
			sq.Eq{"status": dbdriver.Pending},
			sq.Lt{"stored_at": params.OlderThan},
		}).
		OrderBy("stored_at ASC").
		Limit(uint64(params.Limit)).PlaceholderFormat(db.pf). //nolint:gosec
		ToSql()
	if err != nil {
		return nil, err
	}

	logging.Debug(logger, query, args)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	results := common.NewIterator(rows, func(r *dbdriver.RecoveryClaim) error {
		return rows.Scan(&r.TxID, &r.StoredAt)
	})

	return iterators.ReadAllPointers(results)
}

// ReleaseRecoveryClaim clears persisted recovery claim metadata and stores the recovery message.
// The default SQL implementation is a no-op.
func (db *TransactionStore) ReleaseRecoveryClaim(context.Context, string, string, string) error {
	return nil
}

func (db *TransactionStore) AddTransactionEndorsementAck(ctx context.Context, txID string, endorser token.Identity, sigma []byte) (err error) {
	logger.DebugfContext(ctx, "adding transaction endorse ack record [%s]", txID)

	now := time.Now().UTC()
	id, err := uuid.GenerateUUID()
	if err != nil {
		return errors.Wrapf(err, "error generating uuid")
	}
	query, args, err := sq.Insert(db.table.TransactionEndorseAck).
		Columns("id", "tx_id", "endorser", "sigma", "stored_at").
		Values(id, txID, endorser, sigma, now).
		PlaceholderFormat(db.pf).
		ToSql()
	if err != nil {
		return err
	}

	logging.Debug(logger, query, txID, fmt.Sprintf("(%d bytes)", len(endorser)), fmt.Sprintf("(%d bytes)", len(sigma)), now)
	if _, err = db.writeDB.ExecContext(ctx, query, args...); err != nil {
		return ttxDBError(err)
	}

	return
}

func (db *TransactionStore) GetTransactionEndorsementAcks(ctx context.Context, txID string) (map[string][]byte, error) {
	query, args, err := sq.Select("endorser", "sigma").
		From(db.table.TransactionEndorseAck).
		Where(sq.Eq{"tx_id": txID}).
		PlaceholderFormat(db.pf).
		ToSql()
	if err != nil {
		return nil, err
	}
	logging.Debug(logger, query, txID)

	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query")
	}
	defer Close(rows)
	acks := make(map[string][]byte)
	for rows.Next() {
		var endorser []byte
		var sigma []byte
		if err := rows.Scan(&endorser, &sigma); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// not an error for compatibility with badger.
				logger.DebugfContext(ctx, "tried to get status for non-existent tx [%s], returning unknown", txID)

				continue
			}

			return nil, errors.Wrapf(err, "error querying db")
		}
		acks[token.Identity(endorser).String()] = sigma
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return acks, nil
}

func (db *TransactionStore) Close() error {
	return common2.Close(db.readDB, db.writeDB)
}

func (db *TransactionStore) SetStatus(ctx context.Context, txID string, status dbdriver.TxStatus, message string) error {
	b := sq.Update(db.table.Requests).
		Set("status", status).
		Where(sq.Eq{"tx_id": txID})
	if len(message) != 0 {
		b = b.Set("status_message", message)
	}
	query, args, err := b.PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return errors.Wrapf(err, "error building update query for tx [%s]", txID)
	}

	logging.Debug(logger, query, args)
	if _, err = db.writeDB.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "error updating tx [%s]", txID)
	}

	return nil
}

func (db *TransactionStore) GetSchema() string {
	return fmt.Sprintf(`
		-- requests
		CREATE TABLE IF NOT EXISTS %s (
			tx_id TEXT NOT NULL PRIMARY KEY,
			request BYTEA NOT NULL,
			status INT NOT NULL,
			status_message TEXT NOT NULL,
			application_metadata JSONB NOT NULL,
			public_metadata JSONB NOT NULL,
			pp_hash BYTEA NOT NULL,
			recovery_claimed_by TEXT,
			recovery_claim_expires_at TIMESTAMP,
			stored_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_status_%s ON %s ( status );
		CREATE INDEX IF NOT EXISTS idx_recovery_claim_%s ON %s ( status, recovery_claim_expires_at, stored_at ) WHERE status = 1;

		-- transactions
		CREATE TABLE IF NOT EXISTS %s (
			id CHAR(36) NOT NULL PRIMARY KEY,
			tx_id TEXT NOT NULL REFERENCES %s,
			action_type INT NOT NULL,
			sender_eid TEXT NOT NULL,
			recipient_eid TEXT NOT NULL,
			token_type TEXT NOT NULL,
			amount NUMERIC(78, 0) NOT NULL,
			stored_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_tx_id_%s ON %s ( tx_id );

		-- movements
		CREATE TABLE IF NOT EXISTS %s (
			id CHAR(36) NOT NULL PRIMARY KEY,
			tx_id TEXT NOT NULL REFERENCES %s,
			enrollment_id TEXT NOT NULL,
			token_type TEXT NOT NULL,
			amount NUMERIC(78, 0) NOT NULL,
			stored_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_tx_id_%s ON %s ( tx_id );
		CREATE INDEX IF NOT EXISTS idx_eid_storedat_%s ON %s ( enrollment_id, stored_at );

		-- validations
		CREATE TABLE IF NOT EXISTS %s (
			tx_id TEXT NOT NULL PRIMARY KEY REFERENCES %s,
			metadata BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL
		);

		-- tea
		CREATE TABLE IF NOT EXISTS %s (
			id CHAR(36) NOT NULL PRIMARY KEY,
			tx_id TEXT NOT NULL,
			endorser BYTEA NOT NULL,
            sigma BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_tx_id_%s ON %s ( tx_id );
		`,
		db.table.Requests, db.table.Requests, db.table.Requests, db.table.Requests, db.table.Requests,
		db.table.Transactions, db.table.Requests, db.table.Transactions, db.table.Transactions,
		db.table.Movements, db.table.Requests, db.table.Movements, db.table.Movements, db.table.Movements, db.table.Movements,
		db.table.Validations, db.table.Requests,
		db.table.TransactionEndorseAck, db.table.TransactionEndorseAck, db.table.TransactionEndorseAck,
	)
}

func (db *TransactionStore) NewTransactionStoreTransaction() (dbdriver.TransactionStoreTransaction, error) {
	txn, err := db.writeDB.Begin()
	if err != nil {
		return nil, err
	}

	return &TransactionStoreTransaction{
		txn:   txn,
		table: &db.table,
		pf:    db.pf,
	}, nil
}

type TransactionStoreTransaction struct {
	txn   *sql.Tx
	table *transactionTables
	pf    sq.PlaceholderFormat
}

func (w *TransactionStoreTransaction) Impl() dbdriver.TransactionImpl {
	return w.txn
}

func (w *TransactionStoreTransaction) Commit() error {
	if err := w.txn.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction: %w", err)
	}
	w.txn = nil

	return nil
}

func (w *TransactionStoreTransaction) Rollback() {
	if w.txn == nil {
		logging.Debug(logger, "nothing to roll back")

		return
	}
	if err := w.txn.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		logger.Errorf("error rolling back (ignoring...): %s", err.Error())
	}
	w.txn = nil
}

func (w *TransactionStoreTransaction) AddTransaction(ctx context.Context, rs ...dbdriver.TransactionRecord) error {
	if w.txn == nil {
		return errors.New("no db transaction in progress")
	}
	insert := sq.Insert(w.table.Transactions).
		Columns("id", "tx_id", "action_type", "sender_eid", "recipient_eid", "token_type", "amount", "stored_at")
	for _, r := range rs {
		logger.DebugfContext(ctx, "adding transaction record [%s:%d,%s:%s:%s:%s]", r.TxID, r.ActionType, r.TokenType, r.SenderEID, r.RecipientEID, r.Amount)
		id, err := uuid.GenerateUUID()
		if err != nil {
			return errors.Wrapf(err, "error generating uuid")
		}
		if r.Amount.BitLen() > maxAmountBits {
			return errors.Errorf("amount [%s] exceeds maximum supported size of %d bits", r.Amount, maxAmountBits)
		}
		insert = insert.Values(id, r.TxID, int(r.ActionType), r.SenderEID, r.RecipientEID, r.TokenType, r.Amount.String(), r.Timestamp.UTC())
	}

	query, args, err := insert.PlaceholderFormat(w.pf).ToSql()
	if err != nil {
		return err
	}
	logging.Debug(logger, query, args)
	_, err = w.txn.ExecContext(ctx, query, args...)

	return ttxDBError(err)
}

func (w *TransactionStoreTransaction) AddTokenRequest(ctx context.Context, txID string, tr []byte, applicationMetadata, publicMetadata map[string][]byte, ppHash driver2.PPHash) error {
	logger.DebugfContext(ctx, "adding token request [%s]", txID)
	if w.txn == nil {
		return errors.New("no db transaction in progress")
	}
	if applicationMetadata == nil {
		applicationMetadata = make(map[string][]byte)
	}
	ja, err := marshal(applicationMetadata)
	if err != nil {
		return errors.New("error marshaling application metadata")
	}
	if publicMetadata == nil {
		publicMetadata = make(map[string][]byte)
	}
	jp, err := marshal(publicMetadata)
	if err != nil {
		return errors.New("error marshaling application metadata")
	}

	query, args, err := sq.Insert(w.table.Requests).
		Columns("tx_id", "request", "status", "status_message", "application_metadata", "public_metadata", "pp_hash", "stored_at").
		Values(txID, tr, dbdriver.Pending, "", ja, jp, ppHash, time.Now().UTC()).
		PlaceholderFormat(w.pf).
		ToSql()
	if err != nil {
		return err
	}
	logging.Debug(logger, query, txID, fmt.Sprintf("(%d bytes)", len(tr)), len(applicationMetadata), len(publicMetadata), len(ppHash))
	_, err = w.txn.ExecContext(ctx, query, args...)

	return ttxDBError(err)
}

func (w *TransactionStoreTransaction) AddMovement(ctx context.Context, rs ...dbdriver.MovementRecord) error {
	if w.txn == nil {
		return errors.New("no db transaction in progress")
	}
	if len(rs) == 0 {
		// nothing to do here
		return nil
	}

	now := time.Now().UTC()
	insert := sq.Insert(w.table.Movements).
		Columns("id", "tx_id", "enrollment_id", "token_type", "amount", "stored_at")
	for _, r := range rs {
		logger.DebugfContext(ctx, "adding movement record [%s]", r)
		id, err := uuid.GenerateUUID()
		if err != nil {
			return errors.Wrapf(err, "error generating uuid")
		}
		if r.Amount.BitLen() > maxAmountBits {
			return errors.Errorf("amount [%s] exceeds maximum supported size of %d bits", r.Amount, maxAmountBits)
		}
		insert = insert.Values(id, r.TxID, r.EnrollmentID, r.TokenType, r.Amount.String(), now)
	}

	query, args, err := insert.PlaceholderFormat(w.pf).ToSql()
	if err != nil {
		return err
	}
	logging.Debug(logger, query, args)
	_, err = w.txn.ExecContext(ctx, query, args...)

	return ttxDBError(err)
}

func (w *TransactionStoreTransaction) AddValidationRecord(ctx context.Context, txID string, meta map[string][]byte) error {
	logger.DebugfContext(ctx, "adding validation record [%s]", txID)
	if w.txn == nil {
		return errors.New("no db transaction in progress")
	}
	md, err := marshal(meta)
	if err != nil {
		return errors.New("can't marshal metadata")
	}
	now := time.Now().UTC()

	query, args, err := sq.Insert(w.table.Validations).
		Columns("tx_id", "metadata", "stored_at").
		Values(txID, md, now).
		PlaceholderFormat(w.pf).
		ToSql()
	if err != nil {
		return err
	}
	logging.Debug(logger, query, txID, len(md), now)

	_, err = w.txn.ExecContext(ctx, query, args...)

	return ttxDBError(err)
}

func (w *TransactionStoreTransaction) SetStatus(ctx context.Context, txID string, status dbdriver.TxStatus, message string) error {
	b := sq.Update(w.table.Requests).
		Set("status", status).
		Where(sq.Eq{"tx_id": txID})
	if len(message) != 0 {
		b = b.Set("status_message", message)
	}
	query, args, err := b.PlaceholderFormat(w.pf).ToSql()
	if err != nil {
		return errors.Wrapf(err, "error building update query for tx [%s]", txID)
	}

	logging.Debug(logger, query, args)
	if _, err = w.txn.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "error updating tx [%s]", txID)
	}

	return nil
}

func ttxDBError(err error) error {
	if err == nil {
		return nil
	}
	logger.Error(err)
	e := strings.ToLower(err.Error())
	if strings.Contains(e, "foreign key constraint") {
		return dbdriver.ErrTokenRequestDoesNotExist
	}

	return err
}

func directionStr(dir dbdriver.SearchDirection) string {
	if dir == dbdriver.FromBeginning {
		return "ASC"
	}

	return "DESC"
}

func marshal(in map[string][]byte) (string, error) {
	if b, err := json.Marshal(in); err != nil {
		return "", err
	} else {
		return string(b), err
	}
}

func unmarshal(in []byte, out *map[string][]byte) error {
	if len(in) == 0 {
		return nil
	}

	return json.Unmarshal(in, out)
}

type noopRecoveryLeadership struct{}

func (noopRecoveryLeadership) Close() error {
	return nil
}

// squirrelPag implements qcommon.ModifiableQuery to bridge the FSC pagination
// system with squirrel query builders.
type squirrelPag struct {
	limit    int
	offset   int
	fields   []string
	orderBys []string
	wheres   []sq.Sqlizer
}

func (s *squirrelPag) AddField(f qcommon.Field) {
	buf := &renderBuf{}
	f.WriteString(buf)
	s.fields = append(s.fields, buf.str.String())
}

func (s *squirrelPag) AddWhere(c qcommon.Condition) {
	buf := &renderBuf{}
	c.WriteString(panicCI{}, buf)
	s.wheres = append(s.wheres, sq.Expr(buf.str.String(), buf.args...))
}

func (s *squirrelPag) AddOrderBy(o qcommon.OrderBy) {
	buf := &renderBuf{}
	o.WriteString(buf)
	s.orderBys = append(s.orderBys, buf.str.String())
}

func (s *squirrelPag) AddLimit(n int) {
	s.limit = n
}

func (s *squirrelPag) AddOffset(n int) {
	s.offset = n
}

func (s *squirrelPag) Apply(b sq.SelectBuilder) sq.SelectBuilder {
	for _, f := range s.fields {
		b = b.Column(f)
	}
	for _, w := range s.wheres {
		b = b.Where(w)
	}
	for _, o := range s.orderBys {
		b = b.OrderBy(o)
	}
	if s.limit == qcommon.ZeroLimit {
		b = b.Limit(0)
	} else if s.limit > 0 {
		b = b.Limit(uint64(s.limit))
	}
	if s.offset > 0 {
		b = b.Offset(uint64(s.offset))
	}

	return b
}

// applyPagination applies FSC pagination to a squirrel SelectBuilder.
// A nil pagination is treated as no-op (same as pagination.None()).
func applyPagination(b sq.SelectBuilder, p driver3.Pagination) sq.SelectBuilder {
	if p == nil {
		return b
	}
	pag := &squirrelPag{}
	pagination.NewDefaultInterpreter().PreProcess(p, pag)

	return pag.Apply(b)
}

// renderBuf captures SQL text and parameters from old Serializable/ConditionSerializable types.
type renderBuf struct {
	str  strings.Builder
	args []interface{}
}

func (b *renderBuf) WriteParam(p qcommon.Param) qcommon.Builder {
	b.str.WriteString("?")
	b.args = append(b.args, p)

	return b
}

func (b *renderBuf) WriteTuples(_ []qcommon.Tuple) qcommon.Builder {
	panic("WriteTuples not supported in pagination context")
}

func (b *renderBuf) WriteString(s string) qcommon.Builder {
	b.str.WriteString(s)

	return b
}

func (b *renderBuf) WriteRune(r rune) qcommon.Builder {
	b.str.WriteRune(r)

	return b
}

func (b *renderBuf) WriteSerializables(ss ...qcommon.Serializable) qcommon.Builder {
	for _, s := range ss {
		s.WriteString(b)
	}

	return b
}

func (b *renderBuf) WriteConditionSerializable(c qcommon.ConditionSerializable, ci qcommon.CondInterpreter) qcommon.Builder {
	c.WriteString(ci, b)

	return b
}

func (b *renderBuf) Build() (string, []qcommon.Param) {
	return b.str.String(), b.args
}

// panicCI is a CondInterpreter that panics if called.
// keyset pagination only uses cond.CmpVal which doesn't call CondInterpreter methods.
type panicCI struct{}

func (panicCI) TimeOffset(_ time.Duration, _ qcommon.Builder) {
	panic("TimeOffset not supported in pagination context")
}

func (panicCI) InTuple(_ []qcommon.Serializable, _ []qcommon.Tuple, _ qcommon.Builder) {
	panic("InTuple not supported in pagination context")
}
