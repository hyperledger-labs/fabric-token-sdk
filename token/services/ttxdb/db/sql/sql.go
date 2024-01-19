/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"sync"
	"time"

	"github.com/hashicorp/go-uuid"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type Persistence struct {
	db     *sql.DB
	closed bool
	table  tableNames

	txn     *sql.Tx
	txnLock sync.Mutex
}

func (db *Persistence) Close() error {
	logger.Info("closing database")
	db.txnLock.Lock()
	defer db.txnLock.Unlock()

	db.txn = nil

	err := db.db.Close()
	if err != nil {
		return errors.Wrap(err, "could not close DB")
	}
	db.closed = true

	return nil
}

func (db *Persistence) BeginUpdate() error {
	logger.Debug("begin update")
	db.txnLock.Lock()
	defer db.txnLock.Unlock()

	if db.txn != nil {
		return errors.New("previous commit in progress")
	}

	tx, err := db.db.Begin()
	if err != nil {
		return errors.Wrap(err, "error starting db transaction")
	}
	db.txn = tx

	return nil
}

func (db *Persistence) Commit() error {
	logger.Debug("commit")
	db.txnLock.Lock()
	defer db.txnLock.Unlock()

	if db.txn == nil {
		return errors.New("no commit in progress")
	}

	err := db.txn.Commit()
	if err != nil {
		return errors.Wrap(err, "could not commit transaction")
	}
	db.txn = nil

	return nil
}

func (db *Persistence) Discard() error {
	logger.Debug("rollback")

	db.txnLock.Lock()
	defer db.txnLock.Unlock()

	if db.txn == nil {
		logger.Debug("no commit in progress")
		return nil
	}
	err := db.txn.Rollback()
	if err != nil {
		return errors.Wrap(err, "error rolling back")
	}

	db.txn = nil

	return nil
}

func (db *Persistence) AddTokenRequest(txID string, tr []byte) error {
	logger.Debugf("adding token request [%s]", txID)
	if db.txn == nil {
		return errors.New("no db transaction in progress")
	}
	query := fmt.Sprintf("INSERT INTO %s (tx_id, request) VALUES ($1, $2)", db.table.Requests)
	logger.Debug(query, txID, fmt.Sprintf("(%d bytes)", len(tr)))

	_, err := db.txn.Exec(query, txID, tr)
	return err
}

func (db *Persistence) GetTokenRequest(txID string) ([]byte, error) {
	var tokenrequest []byte
	query := fmt.Sprintf("SELECT request FROM %s WHERE tx_id=$1;", db.table.Requests)
	logger.Debug(query, txID)

	row := db.db.QueryRow(query, txID)
	err := row.Scan(&tokenrequest)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.Errorf("not found: [%s]", txID)
		}
		return nil, errors.Wrapf(err, "error querying db")
	}
	return tokenrequest, nil
}

func (db *Persistence) QueryMovements(params driver.QueryMovementsParams) (res []*driver.MovementRecord, err error) {
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

		res = append(res, &r)
	}
	if err = rows.Err(); err != nil {
		return res, err
	}
	return res, nil
}

func (db *Persistence) AddMovement(r *driver.MovementRecord) error {
	logger.Debugf("adding movement record [%s:%s:%s:%d:%s]", r.TxID, r.EnrollmentID, r.TokenType, r.Amount, r.Status)
	if db.txn == nil {
		return errors.New("no db transaction in progress")
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

	query := fmt.Sprintf(`INSERT INTO %s (id, tx_id, enrollment_id, token_type, amount, status, stored_at) VALUES ($1, $2, $3, $4, $5, $6, $7);`, db.table.Movements)
	logger.Debug(query, id, r.TxID, r.EnrollmentID, r.TokenType, amount, r.Status, now)
	_, err = db.txn.Exec(query, id, r.TxID, r.EnrollmentID, r.TokenType, amount, r.Status, now)

	return err
}

func (db *Persistence) QueryTransactions(params driver.QueryTransactionsParams) (driver.TransactionIterator, error) {
	conditions, args := transactionsConditionsSql(params)
	query := fmt.Sprintf("SELECT tx_id, action_type, sender_eid, recipient_eid, token_type, amount, status, stored_at FROM %s ", db.table.Transactions) + conditions

	logger.Debug(query, args)
	rows, err := db.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	return &TransactionIterator{txs: rows}, nil
}

func (db *Persistence) AddTransaction(r *driver.TransactionRecord) error {
	logger.Debugf("adding transaction record [%s:%d:%s:%s:%s:%s]", r.TxID, r.ActionType, r.TokenType, r.SenderEID, r.RecipientEID, r.Amount)
	if db.txn == nil {
		return errors.New("no db transaction in progress")
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

	query := fmt.Sprintf("INSERT INTO %s (id, tx_id, action_type, sender_eid, recipient_eid, token_type, amount, status, stored_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);", db.table.Transactions)
	logger.Debug(query, id, r.TxID, actionType, r.SenderEID, r.RecipientEID, r.TokenType, amount, r.Status, r.Timestamp.UTC())
	_, err = db.txn.Exec(query, id, r.TxID, actionType, r.SenderEID, r.RecipientEID, r.TokenType, amount, r.Status, r.Timestamp.UTC())

	return err
}

func (db *Persistence) SetStatus(txID string, status driver.TxStatus) error {
	logger.Debugf("setting [%s] status to [%s]", txID, status)
	tx, err := db.db.Begin()
	if err != nil {
		return errors.New("failed starting a transaction")
	}
	defer tx.Rollback()

	if err := db.setStatusIfExists(tx, db.table.Movements, txID, status); err != nil {
		return err
	}
	if err := db.setStatusIfExists(tx, db.table.Transactions, txID, status); err != nil {
		return err
	}
	if err := db.setStatusIfExists(tx, db.table.Validations, txID, status); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed committing status update")
	}

	return nil
}

func (db *Persistence) GetStatus(txID string) (driver.TxStatus, error) {
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

func (db *Persistence) AddValidationRecord(txID string, tokenrequest []byte, meta map[string][]byte) error {
	logger.Debugf("adding validation record [%s]", txID)
	if db.txn == nil {
		return errors.New("no db transaction in progress")
	}

	status := "" // analogous to badger implementation
	md, err := marshal(meta)
	if err != nil {
		return errors.New("can't marshal metadata")
	}
	now := time.Now().UTC()

	query := fmt.Sprintf("INSERT INTO %s (tx_id, request, metadata, status, stored_at) VALUES ($1, $2, $3, $4, $5)", db.table.Validations)
	logger.Debug(query, txID, fmt.Sprintf("(%d bytes)", len(tokenrequest)), fmt.Sprintf("(%d bytes)", len(md)), now)

	_, err = db.txn.Exec(query, txID, tokenrequest, md, status, now)
	return err
}

func (db *Persistence) QueryValidations(params driver.QueryValidationRecordsParams) (driver.ValidationRecordsIterator, error) {
	conditions, args := validationConditionsSql(params)
	query := fmt.Sprintf("SELECT tx_id, request, metadata, status, stored_at FROM %s ", db.table.Validations) + conditions

	logger.Debug(query, args)
	rows, err := db.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	return &ValidationRecordsIterator{txs: rows, filter: params.Filter}, nil
}

func (db *Persistence) AddTransactionEndorsementAck(txID string, endorser view.Identity, sigma []byte) error {
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
	defer tx.Rollback()
	if _, err := tx.Exec(query, id, txID, endorser, sigma, now); err != nil {
		return errors.Wrapf(err, "failed to execute")
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed committing status update")
	}
	return nil
}

func (db *Persistence) GetTransactionEndorsementAcks(txID string) (map[string][]byte, error) {
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

func (db *Persistence) StoreCertifications(certifications map[*token.ID][]byte) error {
	now := time.Now().UTC()
	query := fmt.Sprintf("INSERT INTO %s (token_id, tx_id, tx_index, certification, stored_at) VALUES ($1, $2, $3, $4, $5)", db.table.Certifications)

	tx, err := db.db.Begin()
	if err != nil {
		return errors.New("failed starting a transaction")
	}
	defer tx.Rollback()
	for tokenID, certification := range certifications {
		tokenIDStr := fmt.Sprintf("%s%d", tokenID.TxId, tokenID.Index)
		logger.Debug(query, tokenIDStr, fmt.Sprintf("(%d bytes)", len(certification)), now)
		if _, err := tx.Exec(query, tokenIDStr, tokenID.TxId, tokenID.Index, certification, now); err != nil {
			return errors.Wrapf(err, "failed to execute")
		}
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed committing status update")
	}
	return nil
}

func (db *Persistence) ExistsCertification(tokenID *token.ID) bool {
	tokenIDStr := fmt.Sprintf("%s%d", tokenID.TxId, tokenID.Index)
	query := fmt.Sprintf("SELECT certification FROM %s WHERE token_id=$1;", db.table.Certifications)
	logger.Debug(query, tokenIDStr)

	row := db.db.QueryRow(query, tokenIDStr)
	var certification []byte
	if err := row.Scan(&certification); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false
		}
		logger.Warnf("tried to check certification existence for token id %s, err %s", tokenIDStr, err)
		return false
	}
	result := len(certification) != 0
	if !result {
		logger.Warnf("tried to check certification existence for token id %s, got an empty certification", tokenIDStr)
	}
	return result
}

func (db *Persistence) GetCertifications(ids []*token.ID, callback func(*token.ID, []byte) error) error {
	if len(ids) == 0 {
		// nothing to do here
		return nil

	}
	// build query
	conditions, tokenIDs := certificationsQuerySql(ids)
	query := fmt.Sprintf("SELECT tx_id, tx_index, certification FROM %s WHERE ", db.table.Certifications) + conditions

	rows, err := db.db.Query(query, tokenIDs...)
	if err != nil {
		return errors.Wrapf(err, "failed to query")
	}
	defer rows.Close()
	for rows.Next() {
		var txID string
		var txIndex int
		var certification []byte
		if err := rows.Scan(&txID, &txIndex, &certification); err != nil {
			return errors.Wrapf(err, "error querying db")
		}
		tokenID := &token.ID{
			TxId:  txID,
			Index: uint64(txIndex),
		}
		if err := callback(tokenID, certification); err != nil {
			return errors.WithMessagef(err, "failed callback for [%s]", tokenID)
		}
	}
	if err = rows.Err(); err != nil {
		return err
	}

	return nil
}

// setStatusIfExists checks if the record exists before updating it, because some sql drivers return an
// error on update of a non-existent record
func (db *Persistence) setStatusIfExists(tx *sql.Tx, table, txID string, status driver.TxStatus) error {
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

func (db *Persistence) CreateSchema() error {
	logger.Info("creating tables")
	tx, err := db.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	schema := fmt.Sprintf(`
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

		CREATE TABLE IF NOT EXISTS %s (
			tx_id TEXT NOT NULL PRIMARY KEY,
			request BYTEA NOT NULL
		);
		CREATE TABLE IF NOT EXISTS %s (
			tx_id TEXT NOT NULL PRIMARY KEY,
			request BYTEA NOT NULL,
			metadata BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL,
			status TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS %s (
			id CHAR(36) NOT NULL PRIMARY KEY,
			tx_id TEXT NOT NULL,
			endorser BYTEA NOT NULL,
            sigma BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_tx_id_%s ON %s ( tx_id );

		CREATE TABLE IF NOT EXISTS %s (
			token_id TEXT NOT NULL,
			tx_id TEXT NOT NULL,
			tx_index INT NOT NULL,
			certification BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS token_id_id_%s ON %s ( token_id );`,
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
		db.table.Certifications,
		db.table.Certifications,
		db.table.Certifications,
	)

	logger.Debug(schema)
	if _, err = tx.Exec(schema); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
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

type tableNames struct {
	Movements             string
	Transactions          string
	Requests              string
	Validations           string
	TransactionEndorseAck string
	Certifications        string
}

func getTableNames(prefix, name string) (tableNames, error) {
	if prefix != "" {
		r := regexp.MustCompile("^[a-zA-Z_]+$")
		if !r.MatchString(prefix) {
			return tableNames{}, errors.New("Illegal character in table prefix, only letters and underscores allowed")
		}
		prefix = prefix + "_"
	}

	// name is usually something like "default,testchannel,token-chaincode",
	// so we shorten it to the first 6 hex characters of the hash.
	h := sha256.New()
	if _, err := h.Write([]byte(name)); err != nil {
		return tableNames{}, errors.Wrapf(err, "error hashing name [%s]", name)
	}
	suffix := "_" + hex.EncodeToString(h.Sum(nil)[:3])

	return tableNames{
		Transactions:          fmt.Sprintf("%stransactions%s", prefix, suffix),
		Movements:             fmt.Sprintf("%smovements%s", prefix, suffix),
		Requests:              fmt.Sprintf("%srequests%s", prefix, suffix),
		Validations:           fmt.Sprintf("%svalidations%s", prefix, suffix),
		TransactionEndorseAck: fmt.Sprintf("%stea%s", prefix, suffix),
		Certifications:        fmt.Sprintf("%sertifications%s", prefix, suffix),
	}, nil
}
