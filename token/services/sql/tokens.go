/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"database/sql"
	"fmt"
	"time"

	tdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokenRecord struct {
	// TxID is the ID of the transaction that created the token
	TxID string
	// Index is the index in the transaction
	Index uint64
	// Namespace is the namespace where the token was created
	Namespace string
	// IssuerRaw represents the serialization of the issuer identity
	// if this is an IssuedToken.
	IssuerRaw []byte
	// OwnerRaw is the serialization of the owner identity
	OwnerRaw []byte
	// Ledger is the raw token as stored on the ledger
	Ledger string
	// LedgerMetadata is the metadata that is stored on the ledger
	LedgerMetadata string
	// Quantity is the number of units of Type carried in the token.
	// It is encoded as a string containing a number in base 16. The string has prefix ``0x''.
	Quantity string
	// Type is the type of token
	Type string
	// Amount is the Quantity converted to decimal
	Amount uint64
}

type TokenDB struct {
	db    *sql.DB
	table tokenTables
}

type tokenTables struct {
	Tokens       string
	Ownership    string
	AuditTokens  string
	IssuedTokens string
}

func newTokenDB(db *sql.DB, tables tokenTables) *TokenDB {
	return &TokenDB{
		db:    db,
		table: tables,
	}
}

func (db *TokenDB) StoreOwnerToken(tr TokenRecord, owners []string) error {
	return db.storeToken(tr, owners, db.table.Tokens)
}

func (db *TokenDB) StoreIssuedToken(tr TokenRecord) error {
	return db.storeToken(tr, []string{}, db.table.IssuedTokens)
}

func (db *TokenDB) StoreAuditToken(tr TokenRecord) error {
	return db.storeToken(tr, []string{}, db.table.AuditTokens)
}

func (db *TokenDB) storeToken(tr TokenRecord, owners []string, table string) error {
	tx, err := db.db.Begin()
	if err != nil {
		return errors.New("failed starting a db transaction")
	}
	defer tx.Rollback()

	// Store token
	now := time.Now().UTC()
	query := fmt.Sprintf("INSERT INTO %s (ns, tx_id, idx, issuer_raw, owner_raw, ledger, ledger_metadata, token_type, quantity, amount, stored_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)", table)
	logger.Debug(query, tr.Namespace, tr.TxID, tr.Index, len(tr.IssuerRaw), len(tr.OwnerRaw), len(tr.Ledger), len(tr.LedgerMetadata), tr.Type, tr.Quantity, tr.Amount, now)
	if _, err := db.db.Exec(query, tr.Namespace, tr.TxID, tr.Index, tr.IssuerRaw, tr.OwnerRaw, tr.Ledger, tr.LedgerMetadata, tr.Type, tr.Quantity, tr.Amount, now); err != nil {
		return errors.Wrapf(err, "error storing token [%s]", tr.TxID)
	}

	// Store ownership
	for _, eid := range owners {
		query = fmt.Sprintf("INSERT INTO %s (ns, tx_id, idx, enrollment_id) VALUES ($1, $2, $3, $4)", db.table.Ownership)
		logger.Debug(query, tr.Namespace, tr.TxID, tr.Index, eid)
		if _, err := db.db.Exec(query, tr.Namespace, tr.TxID, tr.Index, eid); err != nil {
			return errors.Wrapf(err, "error storing token [%s]", tr.TxID)
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed committing token storage")
	}

	return err
}

// Delete is called when spending a token
func (db *TokenDB) Delete(ns, txID string, index uint64, deletedBy string) error {
	// We don't delete audit tokens, and we keep the 'ownership' relation.
	now := time.Now().UTC()
	query := fmt.Sprintf("UPDATE %s SET is_deleted = true, spent_by = $1, spent_at = $2 WHERE ns = $3 AND tx_id = $4 AND idx = $5;", db.table.Tokens)
	logger.Debug(query, deletedBy, now, ns, txID, index)
	if _, err := db.db.Exec(query, deletedBy, now, ns, txID, index); err != nil {
		return errors.Wrapf(err, "error setting token to deleted [%s]", txID)
	}
	return nil
}

// Delete multiple tokens at the same time (e.g. when invalid or expired)
func (db *TokenDB) DeleteTokens(ns string, ids ...*token.ID) error {
	now := time.Now().UTC()

	args := []interface{}{"", now}
	where := whereTokenIDs(&args, ns, ids)

	query := fmt.Sprintf("UPDATE %s SET is_deleted = true, spent_by = $1, spent_at = $2 WHERE %s", db.table.Tokens, where)
	logger.Debug(query, args)
	if _, err := db.db.Exec(query, args...); err != nil {
		return errors.Wrapf(err, "error setting tokens to deleted [%v]", ids)
	}
	return nil
}

// IsMine just checks if the token is in the local storage and not deleted
func (db *TokenDB) IsMine(ns, txID string, index uint64) (bool, error) {
	id := ""
	query := fmt.Sprintf("SELECT tx_id FROM %s WHERE ns = $1 AND tx_id = $2 AND idx = $3 AND is_deleted = false LIMIT 1;", db.table.Tokens)
	logger.Debug(query, ns, txID, index)

	row := db.db.QueryRow(query, ns, txID, index)
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.Wrapf(err, "error querying db")
	}
	return id == txID, nil
}

// UnspentTokensIterator returns an iterator over all unspent tokens
func (db *TokenDB) UnspentTokensIterator(ns string) (tdriver.UnspentTokensIterator, error) {
	var uti UnspentTokensIterator

	query := fmt.Sprintf("SELECT tx_id, idx, owner_raw, token_type, quantity FROM %s WHERE ns = $1 AND is_deleted = false", db.table.Tokens)
	logger.Debug(query, ns)
	rows, err := db.db.Query(query, ns)
	uti.txs = rows
	return &uti, err
}

// UnspentTokensIteratorBy returns an iterator of unspent tokens owned by the passed id and whose type is the passed on.
// The token type can be empty. In that case, tokens of any type are returned.
func (db *TokenDB) UnspentTokensIteratorBy(ns, ownerEID, typ string) (tdriver.UnspentTokensIterator, error) {
	var uti UnspentTokensIterator

	args := []interface{}{ns}
	if ownerEID != "" {
		args = append(args, ownerEID)
	}
	if typ != "" {
		args = append(args, typ)
	}
	query := fmt.Sprintf("SELECT %s.tx_id, %s.idx, owner_raw, token_type, quantity FROM %s INNER JOIN %s ON %s.ns = $1 AND %s.tx_id = %s.tx_id AND %s.idx = %s.idx AND %s.is_deleted = false",
		db.table.Tokens, db.table.Tokens, // select
		db.table.Tokens,                     // from
		db.table.Ownership,                  // inner join
		db.table.Ownership,                  // .ns
		db.table.Tokens, db.table.Ownership, // .txid
		db.table.Tokens, db.table.Ownership, // .idx
		db.table.Tokens) // Unspent
	if ownerEID != "" {
		query += " AND enrollment_id = $2"
	}
	if typ != "" {
		query += fmt.Sprintf(" AND token_type = $%d", len(args))
	}
	logger.Debug(query, args)
	rows, err := db.db.Query(query, args...)
	uti.txs = rows
	return &uti, err
}

// ListUnspentTokensBy returns the list of unspent tokens, filtered by owner and token type
func (db *TokenDB) ListUnspentTokensBy(ns, ownerEID, typ string) (*token.UnspentTokens, error) {
	logger.Debugf("List unspent token...")
	it, err := db.UnspentTokensIteratorBy(ns, ownerEID, typ)
	if err != nil {
		return nil, err
	}
	defer it.Close()
	tokens := make([]*token.UnspentToken, 0)
	for {
		next, err := it.Next()
		switch {
		case err != nil:
			logger.Errorf("scan failed [%s]", err)
			return nil, err
		case next == nil:
			return &token.UnspentTokens{Tokens: tokens}, nil
		default:
			tokens = append(tokens, next)
		}
	}
}

// ListUnspentTokens returns the list of unspent tokens
func (db *TokenDB) ListUnspentTokens(ns string) (*token.UnspentTokens, error) {
	logger.Debugf("List unspent token...")
	it, err := db.UnspentTokensIterator(ns)
	if err != nil {
		return nil, err
	}
	defer it.Close()
	tokens := make([]*token.UnspentToken, 0)
	for {
		next, err := it.Next()
		switch {
		case err != nil:
			logger.Errorf("scan failed [%s]", err)
			return nil, err
		case next == nil:
			return &token.UnspentTokens{Tokens: tokens}, nil
		default:
			tokens = append(tokens, next)
		}
	}
}

// ListAuditTokens returns the audited tokens associated to the passed ids
func (db *TokenDB) ListAuditTokens(ns string, ids ...*token.ID) ([]*token.Token, error) {
	if len(ids) == 0 {
		return []*token.Token{}, nil
	}
	args := make([]interface{}, 0)
	where := whereTokenIDs(&args, ns, ids)

	query := fmt.Sprintf("SELECT owner_raw, token_type, quantity FROM %s WHERE %s", db.table.AuditTokens, where)
	logger.Debug(query, args)
	rows, err := db.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := []*token.Token{}
	for rows.Next() {
		tok := token.Token{
			Owner: &token.Owner{
				Raw: []byte{},
			},
			Type:     "",
			Quantity: "",
		}
		if err := rows.Scan(&tok.Owner.Raw, &tok.Type, &tok.Quantity); err != nil {
			return tokens, err
		}
		tokens = append(tokens, &tok)
	}
	return tokens, rows.Err()
}

// ListHistoryIssuedTokens returns the list of issued tokens
func (db *TokenDB) ListHistoryIssuedTokens(ns string) (*token.IssuedTokens, error) {
	query := fmt.Sprintf("SELECT tx_id, idx, owner_raw, token_type, quantity, issuer_raw FROM %s WHERE ns = $1", db.table.IssuedTokens)
	logger.Debug(query, ns)
	rows, err := db.db.Query(query, ns)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := []*token.IssuedToken{}
	for rows.Next() {
		tok := token.IssuedToken{
			Id: &token.ID{
				TxId:  "",
				Index: 0,
			},
			Owner: &token.Owner{
				Raw: []byte{},
			},
			Type:     "",
			Quantity: "",
			Issuer: &token.Owner{
				Raw: []byte{},
			},
		}
		if err := rows.Scan(&tok.Id.TxId, &tok.Id.Index, &tok.Owner.Raw, &tok.Type, &tok.Quantity, &tok.Issuer.Raw); err != nil {
			return nil, err
		}
		tokens = append(tokens, &tok)
	}
	return &token.IssuedTokens{Tokens: tokens}, rows.Err()
}

// GetTokenInfos retrieves the token metadata for the passed ids.
// For each id, the callback is invoked to unmarshal the token metadata
func (db *TokenDB) GetTokenInfos(ns string, ids []*token.ID, callback tdriver.QueryCallbackFunc) error {
	if len(ids) == 0 {
		return nil
	}
	_, metas, err := db.getLedgerTokenAndMeta(ns, ids)
	if err != nil {
		return err
	}
	for i := 0; i < len(ids); i++ {
		if err := callback(ids[i], metas[i]); err != nil {
			return err
		}
	}
	return nil
}

// GetTokenInfoAndOutputs retrieves both the token output and information for the passed ids.
func (db *TokenDB) GetTokenInfoAndOutputs(ns string, ids []*token.ID, callback tdriver.QueryCallback2Func) error {
	tokens, metas, err := db.getLedgerTokenAndMeta(ns, ids)
	if err != nil {
		return err
	}
	for i := 0; i < len(ids); i++ {
		outputID, err := keys.CreateTokenKey(ids[i].TxId, ids[i].Index)
		if err != nil {
			return errors.Wrapf(err, "error creating output ID: %v", ids[i])
		}
		if err := callback(ids[i], outputID, tokens[i], metas[i]); err != nil {
			return err
		}
	}
	return nil
}

// GetAllTokenInfos retrieves the token information for the passed ids.
func (db *TokenDB) GetAllTokenInfos(ns string, ids []*token.ID) ([][]byte, error) {
	if len(ids) == 0 {
		return [][]byte{}, nil
	}
	_, metas, err := db.getLedgerTokenAndMeta(ns, ids)
	if err != nil {
		return metas, err
	}

	return metas, nil
}

func (db *TokenDB) getLedgerTokenAndMeta(ns string, ids []*token.ID) ([][]byte, [][]byte, error) {
	tokens := make([][]byte, len(ids))
	metas := make([][]byte, len(ids))
	if len(ids) == 0 {
		return tokens, metas, nil
	}
	args := make([]interface{}, 0)
	where := whereTokenIDs(&args, ns, ids)

	query := fmt.Sprintf("SELECT tx_id, idx, ledger, ledger_metadata FROM %s WHERE %s AND is_deleted = false", db.table.Tokens, where)
	logger.Debug(query, args)
	rows, err := db.db.Query(query, args...)
	if err != nil {
		return tokens, metas, err
	}
	defer rows.Close()

	for rows.Next() {
		var tok []byte
		var metadata []byte
		var id token.ID
		if err := rows.Scan(&id.TxId, &id.Index, &tok, &metadata); err != nil {
			return tokens, metas, err
		}
		// the callback is expected to be called in order of the ids
		for i := 0; i < len(ids); i++ {
			if *ids[i] == id {
				tokens[i] = tok
				metas[i] = metadata
				break
			}
		}
	}
	if err = rows.Err(); err != nil {
		return tokens, metas, err
	}
	return tokens, metas, nil
}

// GetTokens returns the list of tokens with their respective vault keys
func (db *TokenDB) GetTokens(ns string, inputs ...*token.ID) ([]string, []*token.Token, error) {
	if len(inputs) == 0 {
		return []string{}, []*token.Token{}, nil
	}
	args := make([]interface{}, 0)
	where := whereTokenIDs(&args, ns, inputs)

	query := fmt.Sprintf("SELECT owner_raw, token_type, quantity FROM %s WHERE %s AND is_deleted = false", db.table.Tokens, where)
	logger.Debug(query, args)
	rows, err := db.db.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	tok := make([]*token.Token, len(inputs))
	ids := make([]string, len(inputs))
	i := 0
	for rows.Next() {
		var typ, quantity string
		var ownerRaw []byte
		err := rows.Scan(
			&ownerRaw,
			&typ,
			&quantity,
		)
		if err != nil {
			return nil, tok, err
		}
		tok[i] = &token.Token{
			Owner:    &token.Owner{Raw: ownerRaw},
			Type:     typ,
			Quantity: quantity,
		}
		// The token keys are used to refer to tokens as stored in the world state by the tokenchaincode
		// so that they can be validated as inputs for the transaction
		ids[i], err = keys.CreateTokenKey(inputs[i].TxId, inputs[i].Index)
		logger.Debugf("input: ", ids[i])
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed generating id key [%v]", ids[i])
		}
		i++
	}
	if err = rows.Err(); err != nil {
		return nil, tok, err
	}
	return ids, tok, nil
}

// WhoDeletedTokens returns information about which transaction deleted the passed tokens.
// The bool array is an indicator used to tell if the token at a given position has been deleted or not
func (db *TokenDB) WhoDeletedTokens(ns string, inputs ...*token.ID) ([]string, []bool, error) {
	if len(inputs) == 0 {
		return []string{}, []bool{}, nil
	}
	args := make([]interface{}, 0)
	where := whereTokenIDs(&args, ns, inputs)

	query := fmt.Sprintf("SELECT tx_id, idx, spent_by, is_deleted FROM %s WHERE %s", db.table.Tokens, where)
	logger.Debug(query, args)
	rows, err := db.db.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	spentBy := make([]string, len(inputs))
	isSpent := make([]bool, len(inputs))

	for rows.Next() {
		var txid string
		var idx uint64
		var spBy string
		var isSp bool
		if err := rows.Scan(&txid, &idx, &spBy, &isSp); err != nil {
			return spentBy, isSpent, err
		}
		// order is not necessarily the same, so we have to set it in a loop
		for i, inp := range inputs {
			if inp.TxId == txid && inp.Index == idx {
				isSpent[i] = isSp
				spentBy[i] = spBy
				break // stop searching for this id but continue looping over rows
			}
		}
	}
	return spentBy, isSpent, rows.Err()
}

type UnspentTokensIterator struct {
	txs *sql.Rows
}

func (u *UnspentTokensIterator) Close() {
	u.txs.Close()
}

func (u *UnspentTokensIterator) Next() (*token.UnspentToken, error) {
	if !u.txs.Next() {
		return nil, nil
	}

	var typ, quantity string
	var owner []byte
	var id token.ID
	// tx_id, idx, owner_raw, token_type, quantity
	err := u.txs.Scan(
		&id.TxId,
		&id.Index,
		&owner,
		&typ,
		&quantity,
	)
	if err != nil {
		return nil, err
	}
	return &token.UnspentToken{
		Id: &id,
		Owner: &token.Owner{
			Raw: owner,
		},
		Type:     typ,
		Quantity: quantity,
	}, err
}

func (db *TokenDB) GetSchema() string {
	// owner_raw is as1 encoded Type(string), Identity([]byte) (see token/core/identity/owner.go).
	// If Type is "htlc", Identity is a json encoded Script.
	return fmt.Sprintf(`
		-- Tokens
		CREATE TABLE IF NOT EXISTS %s (
			ns TEXT NOT NULL,
			tx_id TEXT NOT NULL,
			idx INT NOT NULL,
			amount BIGINT NOT NULL,
			token_type TEXT NOT NULL,
			quantity TEXT NOT NULL,
			issuer_raw BYTEA,
			owner_raw BYTEA NOT NULL,
			ledger TEXT NOT NULL,
			ledger_metadata TEXT NOT NULL,
			stored_at TIMESTAMP NOT NULL,
			is_deleted BOOL NOT NULL DEFAULT false,
			spent_by TEXT NOT NULL DEFAULT '',
			spent_at TIMESTAMP,
			PRIMARY KEY (tx_id, idx, ns)
		);
		CREATE INDEX IF NOT EXISTS idx_spent_%s ON %s ( is_deleted );

		-- Audit tokens
		CREATE TABLE IF NOT EXISTS %s (
			ns TEXT NOT NULL,
			tx_id TEXT NOT NULL,
			idx INT NOT NULL,
			amount BIGINT NOT NULL,
			token_type TEXT NOT NULL,
			quantity TEXT NOT NULL,
			issuer_raw BYTEA,
			owner_raw BYTEA NOT NULL,
			ledger TEXT NOT NULL,
			ledger_metadata TEXT NOT NULL,
			stored_at TIMESTAMP NOT NULL,
			PRIMARY KEY (tx_id, idx, ns)
		);

		-- Issued tokens
		CREATE TABLE IF NOT EXISTS %s (
			ns TEXT NOT NULL,
			tx_id TEXT NOT NULL,
			idx INT NOT NULL,
			amount BIGINT NOT NULL,
			token_type TEXT NOT NULL,
			quantity TEXT NOT NULL,
			owner_raw BYTEA NOT NULL,
			issuer_raw BYTEA NOT NULL,
			ledger TEXT NOT NULL,
			ledger_metadata TEXT NOT NULL,
			stored_at TIMESTAMP NOT NULL,
			PRIMARY KEY (tx_id, idx, ns)
		);

		-- Ownership
		CREATE TABLE IF NOT EXISTS %s (
			ns TEXT NOT NULL,
			tx_id TEXT NOT NULL,
			idx INT NOT NULL,
			enrollment_id TEXT NOT NULL,
			PRIMARY KEY (tx_id, idx, ns, enrollment_id)
		);
		`,
		db.table.Tokens,
		db.table.Tokens,
		db.table.Tokens,
		db.table.AuditTokens,
		db.table.IssuedTokens,
		db.table.Ownership,
	)
}
