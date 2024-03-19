/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"runtime/debug"
	"time"

	"github.com/hashicorp/go-uuid"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
)

type transactionTables struct {
	Movements             string
	Transactions          string
	Requests              string
	Validations           string
	TransactionEndorseAck string
}

type TransactionDB struct {
	db    *sql.DB
	table transactionTables
}

func newTransactionDB(db *sql.DB, tables transactionTables) *TransactionDB {
	return &TransactionDB{
		db:    db,
		table: tables,
	}
}

func NewTransactionDB(db *sql.DB, tablePrefix string, createSchema bool) (*TransactionDB, error) {
	tables, err := getTableNames(tablePrefix)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table names")
	}
	transactionsDB := newTransactionDB(db, transactionTables{
		Movements:             tables.Movements,
		Transactions:          tables.Transactions,
		Requests:              tables.Requests,
		Validations:           tables.Validations,
		TransactionEndorseAck: tables.TransactionEndorseAck,
	})
	if createSchema {
		if err = initSchema(db, transactionsDB.GetSchema()); err != nil {
			return nil, err
		}
	}
	return transactionsDB, nil
}

func (db *TransactionDB) GetTokenRequest(txID string) ([]byte, error) {
	var tokenrequest []byte
	query := fmt.Sprintf("SELECT request FROM %s WHERE tx_id=$1;", db.table.Requests)
	logger.Debug(query, txID)

	row := db.db.QueryRow(query, txID)
	err := row.Scan(&tokenrequest)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "error querying db")
	}
	return tokenrequest, nil
}

func (db *TransactionDB) QueryMovements(params driver.QueryMovementsParams) (res []*driver.MovementRecord, err error) {
	conditions, args := movementConditionsSql(params)
	query := fmt.Sprintf("SELECT tx_id, enrollment_id, token_type, amount, status FROM %s ", db.table.Movements) + conditions

	logger.Debug(query, args)
	rows, err := db.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var r driver.MovementRecord
		var amount int64
		err = rows.Scan(
			&r.TxID,
			&r.EnrollmentID,
			&r.TokenType,
			&amount,
			&r.Status,
		)
		if err != nil {
			return res, err
		}
		r.Amount = big.NewInt(amount)
		logger.Debugf("movement [%s:%s:%d]", r.TxID, r.Status, r.Amount)

		res = append(res, &r)
	}
	if err = rows.Err(); err != nil {
		return res, err
	}
	return res, nil
}

func (db *TransactionDB) QueryTransactions(params driver.QueryTransactionsParams) (driver.TransactionIterator, error) {
	conditions, args := transactionsConditionsSql(params)
	query := fmt.Sprintf("SELECT tx_id, action_type, sender_eid, recipient_eid, token_type, amount, status, stored_at FROM %s ", db.table.Transactions) + conditions

	logger.Debug(query, args)
	rows, err := db.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	return &TransactionIterator{txs: rows}, nil
}

func (db *TransactionDB) SetStatus(txID string, status driver.TxStatus) (err error) {
	logger.Debugf("setting [%s] status to [%s]", txID, status)
	tx, err := db.db.Begin()
	if err != nil {
		return errors.New("failed starting a transaction")
	}
	defer func() {
		if err != nil && tx != nil {
			if err := tx.Rollback(); err != nil {
				logger.Errorf("failed to rollback [%s][%s]", err, debug.Stack())
			}
		}
	}()

	if err = db.setStatusIfExists(tx, db.table.Movements, txID, status); err != nil {
		return err
	}
	if err = db.setStatusIfExists(tx, db.table.Transactions, txID, status); err != nil {
		return err
	}
	if err = db.setStatusIfExists(tx, db.table.Validations, txID, status); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(err, "failed committing status update")
	}

	return
}

// setStatusIfExists checks if the record exists before updating it, because some sql drivers return an
// error on update of a non-existent record
func (db *TransactionDB) setStatusIfExists(tx *sql.Tx, table, txID string, status driver.TxStatus) error {
	curStatus := driver.Unknown
	query := fmt.Sprintf("SELECT status FROM %s WHERE tx_id = $1 LIMIT 1;", table)
	logger.Debug(query)

	err := tx.QueryRow(query, txID).Scan(&curStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Debugf("no %s found for txID %s, skipping", table, txID)
			return nil
		} else {
			return errors.Wrapf(err, "db error")
		}
	}
	if status == curStatus {
		logger.Debugf("status for %s %s is already %s, skipping", table, txID, status)
		return nil
	}

	query = fmt.Sprintf("UPDATE %s SET status = $1 WHERE tx_id = $2;", table)
	logger.Debug(query)

	_, err = tx.Exec(query, status, txID)
	if err != nil {
		return errors.Wrapf(err, "error updating tx [%s]", txID)
	}

	return nil
}

func (db *TransactionDB) GetStatus(txID string) (driver.TxStatus, error) {
	var status driver.TxStatus
	query := fmt.Sprintf("SELECT status FROM %s WHERE tx_id=$1;", db.table.Transactions)
	logger.Debug(query, txID)

	row := db.db.QueryRow(query, txID)
	if err := row.Scan(&status); err != nil {
		if err == sql.ErrNoRows {
			// not an error for compatibility with badger.
			logger.Warnf("tried to get status for non-existent tx %s, returning unknown", txID)
			return driver.Unknown, nil
		}
		return driver.Unknown, errors.Wrapf(err, "error querying db")
	}
	return status, nil
}

func (db *TransactionDB) QueryValidations(params driver.QueryValidationRecordsParams) (driver.ValidationRecordsIterator, error) {
	conditions, args := validationConditionsSql(params)
	query := fmt.Sprintf("SELECT tx_id, request, metadata, status, stored_at FROM %s ", db.table.Validations) + conditions

	logger.Debug(query, args)
	rows, err := db.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	return &ValidationRecordsIterator{txs: rows, filter: params.Filter}, nil
}

func (db *TransactionDB) AddTransactionEndorsementAck(txID string, endorser view.Identity, sigma []byte) (err error) {
	logger.Debugf("adding transaction endorse ack record [%s]", txID)

	now := time.Now().UTC()
	query := fmt.Sprintf("INSERT INTO %s (id, tx_id, endorser, sigma, stored_at) VALUES ($1, $2, $3, $4, $5)", db.table.TransactionEndorseAck)
	logger.Debug(query, txID, fmt.Sprintf("(%d bytes)", len(endorser)), fmt.Sprintf("(%d bytes)", len(sigma)), now)
	id, err := uuid.GenerateUUID()
	if err != nil {
		return errors.Wrapf(err, "error generating uuid")
	}

	tx, err := db.db.Begin()
	if err != nil {
		return errors.New("failed starting a transaction")
	}
	defer func() {
		if err != nil && tx != nil {
			if err := tx.Rollback(); err != nil {
				logger.Errorf("failed to rollback [%s][%s]", err, debug.Stack())
			}
		}
	}()
	if _, err = tx.Exec(query, id, txID, endorser, sigma, now); err != nil {
		return errors.Wrapf(err, "failed to execute")
	}
	if err = tx.Commit(); err != nil {
		return errors.Wrap(err, "failed committing status update")
	}
	return
}

func (db *TransactionDB) GetTransactionEndorsementAcks(txID string) (map[string][]byte, error) {
	query := fmt.Sprintf("SELECT endorser, sigma FROM %s WHERE tx_id=$1;", db.table.TransactionEndorseAck)
	logger.Debug(query, txID)

	rows, err := db.db.Query(query, txID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query")
	}
	defer rows.Close()
	acks := make(map[string][]byte)
	for rows.Next() {
		var endorser []byte
		var sigma []byte
		if err := rows.Scan(&endorser, &sigma); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// not an error for compatibility with badger.
				logger.Warnf("tried to get status for non-existent tx %s, returning unknown", txID)
				continue
			}
			return nil, errors.Wrapf(err, "error querying db")
		}
		acks[view.Identity(endorser).String()] = sigma
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return acks, nil
}

func (db *TransactionDB) Close() error {
	logger.Info("closing database")
	err := db.db.Close()
	if err != nil {
		return errors.Wrap(err, "could not close DB")
	}

	return nil
}

func (db *TransactionDB) GetSchema() string {
	return fmt.Sprintf(`
	  -- Transactions
		CREATE TABLE IF NOT EXISTS %s (
			id CHAR(36) NOT NULL PRIMARY KEY,
			tx_id TEXT NOT NULL,
			action_type INT NOT NULL,
			sender_eid TEXT NOT NULL,
			recipient_eid TEXT NOT NULL,
			token_type TEXT NOT NULL,
			amount BIGINT NOT NULL,
			stored_at TIMESTAMP NOT NULL,
			status TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_tx_id_%s ON %s ( tx_id );

		-- Movements
		CREATE TABLE IF NOT EXISTS %s (
			id CHAR(36) NOT NULL PRIMARY KEY,
			tx_id TEXT NOT NULL,
			enrollment_id TEXT NOT NULL,
			token_type TEXT NOT NULL,
			amount BIGINT NOT NULL,
			stored_at TIMESTAMP NOT NULL,
			status TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_tx_id_%s ON %s ( tx_id );

		-- Requests
		CREATE TABLE IF NOT EXISTS %s (
			tx_id TEXT NOT NULL PRIMARY KEY,
			request BYTEA NOT NULL
		);

		-- Validations
		CREATE TABLE IF NOT EXISTS %s (
			tx_id TEXT NOT NULL PRIMARY KEY,
			request BYTEA NOT NULL,
			metadata BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL,
			status TEXT NOT NULL
		);

		-- TransactionEndorseAck
		CREATE TABLE IF NOT EXISTS %s (
			id CHAR(36) NOT NULL PRIMARY KEY,
			tx_id TEXT NOT NULL,
			endorser BYTEA NOT NULL,
      sigma BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_tx_id_%s ON %s ( tx_id );
		`,
		db.table.Transactions,
		db.table.Transactions,
		db.table.Transactions,
		db.table.Movements,
		db.table.Movements,
		db.table.Movements,
		db.table.Requests,
		db.table.Validations,
		db.table.TransactionEndorseAck,
		db.table.TransactionEndorseAck,
		db.table.TransactionEndorseAck,
	)
}

func marshal(in map[string][]byte) (string, error) {
	if b, err := json.Marshal(in); err != nil {
		return "", err
	} else {
		return string(b), err
	}
}

func unmarshal(in []byte, out *map[string][]byte) error {
	return json.Unmarshal(in, out)
}

type TransactionIterator struct {
	txs *sql.Rows
}

func (t *TransactionIterator) Close() {
	t.txs.Close()
}

func (t *TransactionIterator) Next() (*driver.TransactionRecord, error) {
	var r driver.TransactionRecord
	if !t.txs.Next() {
		return nil, nil
	}
	var actionType int
	var amount int64
	err := t.txs.Scan(
		&r.TxID,
		&actionType,
		&r.SenderEID,
		&r.RecipientEID,
		&r.TokenType,
		&amount,
		&r.Status,
		&r.Timestamp,
	)

	r.ActionType = driver.ActionType(actionType)
	r.Amount = big.NewInt(amount)

	return &r, err
}

type ValidationRecordsIterator struct {
	txs *sql.Rows
	// Filter defines a custom filter function.
	// If specified, this filter will be applied.
	// the filter returns true if the record must be selected, false otherwise.
	filter func(record *driver.ValidationRecord) bool
}

func (t *ValidationRecordsIterator) Close() {
	t.txs.Close()
}

func (t *ValidationRecordsIterator) Next() (*driver.ValidationRecord, error) {
	var r driver.ValidationRecord
	if !t.txs.Next() {
		return nil, nil
	}

	var meta []byte
	var storedAt time.Time
	if err := t.txs.Scan(
		&r.TxID,
		&r.TokenRequest,
		&meta,
		&r.Status,
		&storedAt,
	); err != nil {
		return &r, err
	}
	if err := unmarshal(meta, &r.Metadata); err != nil {
		return &r, err
	}
	r.Timestamp = storedAt

	// sqlite database returns nil for empty slice
	if r.TokenRequest == nil {
		r.TokenRequest = []byte{}
	}

	// no filter supplied, or filter matches
	if t.filter == nil ||
		t.filter(&r) {
		return &r, nil
	}

	// Skipping this record causes a recursive call
	// to this function to parse next record
	return t.Next()
}

func (db *TransactionDB) BeginAtomicWrite() (driver.AtomicWrite, error) {
	txn, err := db.db.Begin()
	if err != nil {
		return nil, err
	}

	return &AtomicWrite{
		txn: txn,
		db:  db,
	}, nil
}

type AtomicWrite struct {
	txn *sql.Tx
	db  *TransactionDB
}

func (w *AtomicWrite) Commit() error {
	if err := w.txn.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction: %w", err)
	}
	w.txn = nil
	return nil
}

func (w *AtomicWrite) Discard() error {
	if err := w.txn.Rollback(); err != nil {
		return err
	}
	w.txn = nil
	return nil
}

func (w *AtomicWrite) AddTransaction(r *driver.TransactionRecord) error {
	logger.Debugf("adding transaction record [%s:%d:%s:%s:%s:%s]", r.TxID, r.ActionType, r.TokenType, r.SenderEID, r.RecipientEID, r.Amount)
	if w.txn == nil {
		panic("no db transaction in progress")
	}
	if !r.Amount.IsInt64() {
		return errors.New("the database driver does not support larger values than int64")
	}
	amount := r.Amount.Int64()
	actionType := int(r.ActionType)
	id, err := uuid.GenerateUUID()
	if err != nil {
		return errors.Wrapf(err, "error generating uuid")
	}

	query := fmt.Sprintf("INSERT INTO %s (id, tx_id, action_type, sender_eid, recipient_eid, token_type, amount, status, stored_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);", w.db.table.Transactions)
	logger.Debug(query, id, r.TxID, actionType, r.SenderEID, r.RecipientEID, r.TokenType, amount, r.Status, r.Timestamp.UTC())
	_, err = w.txn.Exec(query, id, r.TxID, actionType, r.SenderEID, r.RecipientEID, r.TokenType, amount, r.Status, r.Timestamp.UTC())

	return err
}

func (w *AtomicWrite) AddTokenRequest(txID string, tr []byte) error {
	logger.Debugf("adding token request [%s]", txID)
	if w.txn == nil {
		panic("no db transaction in progress")
	}
	query := fmt.Sprintf("INSERT INTO %s (tx_id, request) VALUES ($1, $2)", w.db.table.Requests)
	logger.Debug(query, txID, fmt.Sprintf("(%d bytes)", len(tr)))

	_, err := w.txn.Exec(query, txID, tr)
	return err
}

func (w *AtomicWrite) AddMovement(r *driver.MovementRecord) error {
	logger.Debugf("adding movement record [%s:%s:%s:%d:%s]", r.TxID, r.EnrollmentID, r.TokenType, r.Amount.Int64(), r.Status)
	if w.txn == nil {
		panic("no db transaction in progress")
	}
	if !r.Amount.IsInt64() {
		return errors.New("the database driver does not support larger values than int64")
	}
	amount := r.Amount.Int64()

	id, err := uuid.GenerateUUID()
	if err != nil {
		return errors.Wrapf(err, "error generating uuid")
	}
	now := time.Now().UTC()

	query := fmt.Sprintf(`INSERT INTO %s (id, tx_id, enrollment_id, token_type, amount, status, stored_at) VALUES ($1, $2, $3, $4, $5, $6, $7);`, w.db.table.Movements)
	logger.Debug(query, id, r.TxID, r.EnrollmentID, r.TokenType, amount, r.Status, now)
	_, err = w.txn.Exec(query, id, r.TxID, r.EnrollmentID, r.TokenType, amount, r.Status, now)

	return err
}

func (w *AtomicWrite) AddValidationRecord(txID string, tokenrequest []byte, meta map[string][]byte) error {
	logger.Debugf("adding validation record [%s]", txID)
	if w.txn == nil {
		return errors.New("no db transaction in progress")
	}

	status := "" // analogous to badger implementation
	md, err := marshal(meta)
	if err != nil {
		return errors.New("can't marshal metadata")
	}
	now := time.Now().UTC()

	query := fmt.Sprintf("INSERT INTO %s (tx_id, request, metadata, status, stored_at) VALUES ($1, $2, $3, $4, $5)", w.db.table.Validations)
	logger.Debug(query, txID, fmt.Sprintf("(%d bytes)", len(tokenrequest)), fmt.Sprintf("(%d bytes)", len(md)), now)

	_, err = w.txn.Exec(query, txID, tokenrequest, md, status, now)
	return err
}
