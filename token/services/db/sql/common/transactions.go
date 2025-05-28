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
	"math/big"
	"strings"
	"time"

	"github.com/hashicorp/go-uuid"
	driver3 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	q "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/query"
	common3 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/query/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/query/cond"
	_select "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/query/select"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type transactionTables struct {
	Movements             string
	Transactions          string
	Requests              string
	Validations           string
	TransactionEndorseAck string
}

type TransactionStore struct {
	readDB  *sql.DB
	writeDB *sql.DB
	table   transactionTables
	ci      common3.CondInterpreter
	pi      common3.PagInterpreter
}

func newTransactionStore(readDB, writeDB *sql.DB, tables transactionTables, ci common3.CondInterpreter, pi common3.PagInterpreter) *TransactionStore {
	return &TransactionStore{
		readDB:  readDB,
		writeDB: writeDB,
		table:   tables,
		ci:      ci,
		pi:      pi,
	}
}

func NewAuditTransactionStore(readDB, writeDB *sql.DB, tables TableNames, ci common3.CondInterpreter, pi common3.PagInterpreter) (*TransactionStore, error) {
	return NewOwnerTransactionStore(readDB, writeDB, tables, ci, pi)
}

func NewOwnerTransactionStore(readDB, writeDB *sql.DB, tables TableNames, ci common3.CondInterpreter, pi common3.PagInterpreter) (*TransactionStore, error) {
	return newTransactionStore(readDB, writeDB, transactionTables{
		Movements:             tables.Movements,
		Transactions:          tables.Transactions,
		Requests:              tables.Requests,
		Validations:           tables.Validations,
		TransactionEndorseAck: tables.TransactionEndorseAck,
	}, ci, pi), nil
}

func (db *TransactionStore) CreateSchema() error {
	return common.InitSchema(db.writeDB, db.GetSchema())
}

func (db *TransactionStore) GetTokenRequest(txID string) ([]byte, error) {
	var tokenrequest []byte
	query, err := NewSelect("request").From(db.table.Requests).Where("tx_id=$1").Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query, txID)

	row := db.readDB.QueryRow(query, txID)
	err = row.Scan(&tokenrequest)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "error querying db")
	}
	return tokenrequest, nil
}

func orderBy(f common3.FieldName, direction driver.SearchDirection) _select.OrderBy {
	if direction == driver.FromBeginning {
		return q.Asc(f)
	}
	return q.Desc(f)
}

func (db *TransactionStore) QueryMovements(params driver.QueryMovementsParams) (res []*driver.MovementRecord, err error) {
	movementsTable, requestsTable := q.Table(db.table.Movements), q.Table(db.table.Requests)
	query, args := q.Select().
		Fields(
			movementsTable.Field("tx_id"), common3.FieldName("enrollment_id"), common3.FieldName("token_type"),
			common3.FieldName("amount"), requestsTable.Field("status"),
		).
		From(movementsTable.Join(requestsTable,
			cond.Cmp(movementsTable.Field("tx_id"), "=", requestsTable.Field("tx_id"))),
		).
		Where(HasMovementsParams(params)).
		OrderBy(orderBy("stored_at", params.SearchDirection)).
		Limit(params.NumRecords).
		Format(db.ci)

	logger.Debug(query, args)
	rows, err := db.readDB.Query(query, args...)
	if err != nil {
		return nil, err
	}

	it := common.NewIterator(rows, func(r *driver.MovementRecord) error {
		var amount int64
		if err := rows.Scan(&r.TxID, &r.EnrollmentID, &r.TokenType, &amount, &r.Status); err != nil {
			return err
		}
		r.Amount = big.NewInt(amount)
		logger.Debugf("movement [%s:%s:%d]", r.TxID, r.Status, r.Amount)
		return nil
	})
	return iterators.ReadAllPointers(it)
}

func (db *TransactionStore) QueryTransactions(params driver.QueryTransactionsParams, pagination driver3.Pagination) (*driver3.PageIterator[*driver.TransactionRecord], error) {
	transactionsTable, requestsTable := q.Table(db.table.Transactions), q.Table(db.table.Requests)
	query, args := q.Select().
		Fields(
			transactionsTable.Field("tx_id"), common3.FieldName("action_type"), common3.FieldName("sender_eid"),
			common3.FieldName("recipient_eid"), common3.FieldName("token_type"), common3.FieldName("amount"),
			requestsTable.Field("status"), requestsTable.Field("application_metadata"),
			requestsTable.Field("public_metadata"), common3.FieldName("stored_at"),
		).
		From(transactionsTable.Join(requestsTable,
			cond.Cmp(transactionsTable.Field("tx_id"), "=", requestsTable.Field("tx_id"))),
		).
		Where(HasTransactionParams(params, transactionsTable)).
		OrderBy(q.Asc(common3.FieldName("stored_at"))).
		Paginated(pagination).
		FormatPaginated(db.ci, db.pi)

	logger.Debug(query, args)
	rows, err := db.readDB.Query(query, args...)
	if err != nil {
		return nil, err
	}

	results := common.NewIterator(rows, func(r *driver.TransactionRecord) error {
		var amount int64
		var appMeta []byte
		var pubMeta []byte
		if err := rows.Scan(&r.TxID, &r.ActionType, &r.SenderEID, &r.RecipientEID, &r.TokenType, &amount, &r.Status, &appMeta, &pubMeta, &r.Timestamp); err != nil {
			return err
		}
		r.Amount = big.NewInt(amount)
		return errors2.Join(
			unmarshal(appMeta, &r.ApplicationMetadata),
			unmarshal(pubMeta, &r.PublicMetadata),
		)
	})

	return &driver3.PageIterator[*driver.TransactionRecord]{
		Items:      results,
		Pagination: pagination,
	}, nil
}

func (db *TransactionStore) GetStatus(txID string) (driver.TxStatus, string, error) {
	var status driver.TxStatus
	var statusMessage string
	query, err := NewSelect("status, status_message").From(db.table.Requests).Where("tx_id=$1").Compile()
	if err != nil {
		return driver.Unknown, "", errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query, txID)

	row := db.readDB.QueryRow(query, txID)
	if err := row.Scan(&status, &statusMessage); err != nil {
		if err == sql.ErrNoRows {
			logger.Debugf("tried to get status for non-existent tx [%s], returning unknown", txID)
			return driver.Unknown, "", nil
		}
		return driver.Unknown, "", errors.Wrapf(err, "error querying db")
	}
	return status, statusMessage, nil
}

func (db *TransactionStore) QueryValidations(params driver.QueryValidationRecordsParams) (driver.ValidationRecordsIterator, error) {
	validationsTable, requestsTable := q.Table(db.table.Validations), q.Table(db.table.Requests)
	query, args := q.Select().
		Fields(
			validationsTable.Field("tx_id"), requestsTable.Field("request"), common3.FieldName("metadata"),
			requestsTable.Field("status"), validationsTable.Field("stored_at"),
		).
		From(validationsTable.Join(requestsTable,
			cond.Cmp(validationsTable.Field("tx_id"), "=", requestsTable.Field("tx_id"))),
		).
		Where(HasValidationParams(params)).
		Format(db.ci)

	logger.Debug(query, args)
	rows, err := db.readDB.Query(query, args...)
	if err != nil {
		return nil, err
	}

	it := common.NewIterator(rows, func(r *driver.ValidationRecord) error {
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

// QueryTokenRequests returns an iterator over the token requests matching the passed params
func (db *TransactionStore) QueryTokenRequests(params driver.QueryTokenRequestsParams) (driver.TokenRequestIterator, error) {
	query, args := q.Select().
		FieldsByName("tx_id", "request", "status").
		From(q.Table(db.table.Requests)).
		Where(cond.In("status", params.Statuses...)).
		Format(db.ci)

	logger.Debug(query, args)
	rows, err := db.readDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	// TODO: AF remove r.TokenRequest. Not used
	return common.NewIterator(rows, func(r *driver.TokenRequestRecord) error { return rows.Scan(&r.TxID, &r.TokenRequest, &r.Status) }), nil
}

func (db *TransactionStore) AddTransactionEndorsementAck(txID string, endorser token.Identity, sigma []byte) (err error) {
	logger.Debugf("adding transaction endorse ack record [%s]", txID)

	now := time.Now().UTC()
	query, err := NewInsertInto(db.table.TransactionEndorseAck).Rows("id, tx_id, endorser, sigma, stored_at").Compile()
	if err != nil {
		return errors.Wrapf(err, "error compiling query")
	}
	logger.Debug(query, txID, fmt.Sprintf("(%d bytes)", len(endorser)), fmt.Sprintf("(%d bytes)", len(sigma)), now)
	id, err := uuid.GenerateUUID()
	if err != nil {
		return errors.Wrapf(err, "error generating uuid")
	}
	if _, err = db.writeDB.Exec(query, id, txID, endorser, sigma, now); err != nil {
		return ttxDBError(err)
	}
	return
}

func (db *TransactionStore) GetTransactionEndorsementAcks(txID string) (map[string][]byte, error) {
	query, err := NewSelect("endorser, sigma").From(db.table.TransactionEndorseAck).Where("tx_id=$1").Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed compiling query")
	}
	logger.Debug(query, txID)

	rows, err := db.readDB.Query(query, txID)
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
				logger.Debugf("tried to get status for non-existent tx [%s], returning unknown", txID)
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

func (db *TransactionStore) SetStatus(ctx context.Context, txID string, status driver.TxStatus, message string) (err error) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent("start_db_update")
	defer span.AddEvent("end_db_update")
	var query string
	if len(message) != 0 {
		query = fmt.Sprintf("UPDATE %s SET status = $1, status_message = $2 WHERE tx_id = $3;", db.table.Requests)
		logger.Debug(query)
		_, err = db.writeDB.Exec(query, status, message, txID)
	} else {
		query = fmt.Sprintf("UPDATE %s SET status = $1 WHERE tx_id = $2;", db.table.Requests)
		logger.Debug(query)
		_, err = db.writeDB.Exec(query, status, txID)
	}
	if err != nil {
		return errors.Wrapf(err, "error updating tx [%s]", txID)
	}
	return
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
			pp_hash BYTEA NOT NULL
		);

		-- transactions
		CREATE TABLE IF NOT EXISTS %s (
			id CHAR(36) NOT NULL PRIMARY KEY,
			tx_id TEXT NOT NULL REFERENCES %s,
			action_type INT NOT NULL,
			sender_eid TEXT NOT NULL,
			recipient_eid TEXT NOT NULL,
			token_type TEXT NOT NULL,
			amount BIGINT NOT NULL,
			stored_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_tx_id_%s ON %s ( tx_id );

		-- movements
		CREATE TABLE IF NOT EXISTS %s (
			id CHAR(36) NOT NULL PRIMARY KEY,
			tx_id TEXT NOT NULL REFERENCES %s,
			enrollment_id TEXT NOT NULL,
			token_type TEXT NOT NULL,
			amount BIGINT NOT NULL,
			stored_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_tx_id_%s ON %s ( tx_id );

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
		db.table.Requests,
		db.table.Transactions, db.table.Requests, db.table.Transactions, db.table.Transactions,
		db.table.Movements, db.table.Requests, db.table.Movements, db.table.Movements,
		db.table.Validations, db.table.Requests,
		db.table.TransactionEndorseAck, db.table.TransactionEndorseAck, db.table.TransactionEndorseAck,
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
	if len(in) == 0 {
		return nil
	}
	return json.Unmarshal(in, out)
}

func (db *TransactionStore) BeginAtomicWrite() (driver.AtomicWrite, error) {
	txn, err := db.writeDB.Begin()
	if err != nil {
		return nil, err
	}

	return &AtomicWrite{
		txn:   txn,
		table: &db.table,
	}, nil
}

type AtomicWrite struct {
	txn   *sql.Tx
	table *transactionTables
}

func (w *AtomicWrite) Commit() error {
	if err := w.txn.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction: %w", err)
	}
	w.txn = nil
	return nil
}

func (w *AtomicWrite) Rollback() {
	if w.txn == nil {
		logger.Debug("nothing to roll back")
		return
	}
	if err := w.txn.Rollback(); err != nil && err != sql.ErrTxDone {
		logger.Errorf("error rolling back (ignoring...): %s", err.Error())
	}
	w.txn = nil
}

func (w *AtomicWrite) AddTransaction(r *driver.TransactionRecord) error {
	logger.Debugf("adding transaction record [%s:%d,%s:%s:%s:%s]", r.TxID, r.ActionType, r.TokenType, r.SenderEID, r.RecipientEID, r.Amount)
	if w.txn == nil {
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

	query, err := NewInsertInto(w.table.Transactions).Rows("id, tx_id, action_type, sender_eid, recipient_eid, token_type, amount, stored_at").Compile()
	if err != nil {
		return errors.Wrapf(err, "error compiling insert")
	}
	args := []any{id, r.TxID, actionType, r.SenderEID, r.RecipientEID, r.TokenType, amount, r.Timestamp.UTC()}
	logger.Debug(query, args)
	_, err = w.txn.Exec(query, args...)

	return ttxDBError(err)
}

func (w *AtomicWrite) AddTokenRequest(txID string, tr []byte, applicationMetadata, publicMetadata map[string][]byte, ppHash driver2.PPHash) error {
	logger.Debugf("adding token request [%s]", txID)
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

	query, err := NewInsertInto(w.table.Requests).Rows("tx_id, request, status, status_message, application_metadata, public_metadata, pp_hash").Compile()
	if err != nil {
		return errors.Wrapf(err, "error compiling query")
	}
	logger.Debug(query, txID, fmt.Sprintf("(%d bytes)", len(tr)), len(applicationMetadata), len(publicMetadata), len(ppHash))

	_, err = w.txn.Exec(query, txID, tr, driver.Pending, "", ja, jp, ppHash)
	return ttxDBError(err)
}

func (w *AtomicWrite) AddMovement(r *driver.MovementRecord) error {
	logger.Debugf("adding movement record [%s:%s:%s:%d:%s]", r.TxID, r.EnrollmentID, r.TokenType, r.Amount.Int64(), r.Status)
	if w.txn == nil {
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

	query, err := NewInsertInto(w.table.Movements).Rows("id, tx_id, enrollment_id, token_type, amount, stored_at").Compile()
	if err != nil {
		return errors.Wrapf(err, "error compiling query")
	}
	args := []any{id, r.TxID, r.EnrollmentID, r.TokenType, amount, now}
	logger.Debug(query, args)
	_, err = w.txn.Exec(query, args...)

	return ttxDBError(err)
}

func (w *AtomicWrite) AddValidationRecord(txID string, meta map[string][]byte) error {
	logger.Debugf("adding validation record [%s]", txID)
	if w.txn == nil {
		return errors.New("no db transaction in progress")
	}
	md, err := marshal(meta)
	if err != nil {
		return errors.New("can't marshal metadata")
	}
	now := time.Now().UTC()

	query, err := NewInsertInto(w.table.Validations).Rows("tx_id, metadata, stored_at").Compile()
	if err != nil {
		return errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query, txID, len(md), now)

	_, err = w.txn.Exec(query, txID, md, now)
	return ttxDBError(err)
}

func ttxDBError(err error) error {
	if err == nil {
		return nil
	}
	logger.Error(err)
	e := strings.ToLower(err.Error())
	if strings.Contains(e, "foreign key constraint") {

		return driver.ErrTokenRequestDoesNotExist
	}
	return err
}
