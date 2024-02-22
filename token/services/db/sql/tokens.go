/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"database/sql"
	"fmt"
	"runtime/debug"
	"time"

	tdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type tokenTables struct {
	Tokens         string
	Ownership      string
	AuditTokens    string
	IssuedTokens   string
	PublicParams   string
	Ledger         string
	Certifications string
}

type TokenDB struct {
	db    *sql.DB
	table tokenTables
}

func newTokenDB(db *sql.DB, tables tokenTables) *TokenDB {
	return &TokenDB{
		db:    db,
		table: tables,
	}
}

func NewTokenDB(db *sql.DB, tablePrefix, name string, createSchema bool) (*TokenDB, error) {
	tables, err := getTableNames(tablePrefix, name)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table names")
	}

	tokenDB := newTokenDB(db, tokenTables{
		Tokens:         tables.Tokens,
		Ownership:      tables.Ownership,
		AuditTokens:    tables.AuditTokens,
		IssuedTokens:   tables.IssuedTokens,
		PublicParams:   tables.PublicParams,
		Ledger:         tables.Ledger,
		Certifications: tables.Certifications,
	})
	if createSchema {
		if err = initSchema(db, tokenDB.GetSchema()); err != nil {
			return nil, err
		}
	}
	return tokenDB, nil
}

func (db *TokenDB) StoreOwnerToken(tr driver.TokenRecord, owners []string) error {
	return db.storeToken(tr, owners, db.table.Tokens, true)
}

func (db *TokenDB) StoreIssuedToken(tr driver.TokenRecord) error {
	return db.storeToken(tr, nil, db.table.IssuedTokens, false)
}

func (db *TokenDB) StoreAuditToken(tr driver.TokenRecord) error {
	return db.storeToken(tr, nil, db.table.AuditTokens, true)
}

func (db *TokenDB) storeToken(tr driver.TokenRecord, owners []string, table string, ledgerInsert bool) error {
	logger.Debugf("store record [%s:%d,%v] in table [%s]", tr.TxID, tr.Index, owners, table)
	tx, err := db.db.Begin()
	if err != nil {
		return errors.New("failed starting a db transaction")
	}
	defer tx.Rollback()

	// Store token
	now := time.Now().UTC()
	query := fmt.Sprintf("INSERT INTO %s (tx_id, idx, issuer_raw, owner_raw, ledger, ledger_metadata, token_type, quantity, amount, stored_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)", table)
	logger.Debug(query, tr.TxID, tr.Index, len(tr.IssuerRaw), len(tr.OwnerRaw), len(tr.Ledger), len(tr.LedgerMetadata), tr.Type, tr.Quantity, tr.Amount, now)
	if _, err := db.db.Exec(query, tr.TxID, tr.Index, tr.IssuerRaw, tr.OwnerRaw, tr.Ledger, tr.LedgerMetadata, tr.Type, tr.Quantity, tr.Amount, now); err != nil {
		logger.Errorf("error storing token [%s] in table [%s]: [%s][%s]", tr.TxID, table, err, string(debug.Stack()))
		return errors.Wrapf(err, "error storing token [%s] in table [%s]", tr.TxID, table)
	}

	// Store ledger part
	if ledgerInsert {
		query = fmt.Sprintf("INSERT INTO %s (tx_id, idx, ledger, ledger_metadata) VALUES ($1, $2, $3, $4)", db.table.Ledger)
		logger.Debug(query, tr.TxID, tr.Index, len(tr.Ledger), len(tr.LedgerMetadata))
		if _, err := db.db.Exec(query, tr.TxID, tr.Index, tr.Ledger, tr.LedgerMetadata); err != nil {
			// is the token already here?
			ids := []*token.ID{{TxId: tr.TxID, Index: tr.Index}}
			logger.Debugf("retrieve ledger tokens for [%s][%s]", ids)
			args := make([]interface{}, 0)
			where := whereTokenIDs(&args, ids)
			query := fmt.Sprintf("SELECT tx_id, idx FROM %s WHERE %s", db.table.Ledger, where)
			logger.Debug(query, args)
			row := db.db.QueryRow(query, args...)
			var txID string
			var index int
			if err2 := row.Scan(&txID, &index); err2 != nil {
				if errors.Is(err2, sql.ErrNoRows) {
					logger.Errorf("error storing ledger token [%s] in table [%s]: [%s][%s]", tr.TxID, db.table.Ledger, err2, string(debug.Stack()))
					return errors.Wrapf(err, "error storing ledger token [%s] in table [%s]", tr.TxID, db.table.Ledger)
				}
			}
			// this was already inserted, skip
		}
	}

	// Store ownership
	for _, eid := range owners {
		query = fmt.Sprintf("INSERT INTO %s (tx_id, idx, enrollment_id) VALUES ($1, $2, $3)", db.table.Ownership)
		logger.Debug(query, tr.TxID, tr.Index, eid)
		if _, err := db.db.Exec(query, tr.TxID, tr.Index, eid); err != nil {
			return errors.Wrapf(err, "error storing token ownership [%s]", tr.TxID)
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed committing token storage")
	}

	return err
}

func (db *TokenDB) OwnersOf(txID string, index uint64) (*token.Token, []string, error) {
	args := make([]interface{}, 0)
	tokenIDs := []*token.ID{{TxId: txID, Index: index}}
	where := whereTokenIDs(&args, tokenIDs)

	// select token
	query := fmt.Sprintf("SELECT owner_raw, token_type, quantity FROM %s WHERE %s AND is_deleted = false", db.table.Tokens, where)
	logger.Debug(query, args)
	row := db.db.QueryRow(query, args...)
	var tokenOwner []byte
	var tokenType string
	var quantity string
	if err := row.Scan(&tokenOwner, &tokenType, &quantity); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	// select owners
	query = fmt.Sprintf("SELECT enrollment_id FROM %s WHERE %s", db.table.Ownership, where)
	logger.Debug(query, args)
	rows, err := db.db.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var owners []string
	for rows.Next() {
		var owner string
		if err := rows.Scan(&owner); err != nil {
			return nil, nil, err
		}
		owners = append(owners, owner)
	}
	if rows.Err() != nil {
		return nil, nil, rows.Err()
	}
	return &token.Token{
		Owner: &token.Owner{
			Raw: tokenOwner,
		},
		Type:     tokenType,
		Quantity: quantity,
	}, owners, nil
}

// Delete is called when spending a token
func (db *TokenDB) Delete(txID string, index uint64, deletedBy string) error {
	logger.Debugf("delete token [%s:%d:%s]", txID, index, deletedBy)
	// We don't delete audit tokens, and we keep the 'ownership' relation.
	now := time.Now().UTC()
	query := fmt.Sprintf("UPDATE %s SET is_deleted = true, spent_by = $1, spent_at = $2 WHERE tx_id = $3 AND idx = $4;", db.table.Tokens)
	logger.Debug(query, deletedBy, now, txID, index)
	if _, err := db.db.Exec(query, deletedBy, now, txID, index); err != nil {
		return errors.Wrapf(err, "error setting token to deleted [%s]", txID)
	}
	query = fmt.Sprintf("UPDATE %s SET is_deleted = true, spent_by = $1, spent_at = $2 WHERE tx_id = $3 AND idx = $4;", db.table.AuditTokens)
	logger.Debug(query, deletedBy, now, txID, index)
	if _, err := db.db.Exec(query, deletedBy, now, txID, index); err != nil {
		return errors.Wrapf(err, "error setting token to deleted [%s]", txID)
	}
	return nil
}

// DeleteTokens delete multiple tokens at the same time (e.g. when invalid or expired)
func (db *TokenDB) DeleteTokens(ids ...*token.ID) error {
	logger.Debugf("delete tokens [%s:%v]", ids)
	if len(ids) == 0 {
		return nil
	}
	now := time.Now().UTC()

	args := []interface{}{"", now}
	where := whereTokenIDs(&args, ids)

	query := fmt.Sprintf("UPDATE %s SET is_deleted = true, spent_by = $1, spent_at = $2 WHERE %s", db.table.Tokens, where)
	logger.Debug(query, args)
	if _, err := db.db.Exec(query, args...); err != nil {
		return errors.Wrapf(err, "error setting tokens to deleted [%v]", ids)
	}
	return nil
}

// IsMine just checks if the token is in the local storage and not deleted
func (db *TokenDB) IsMine(txID string, index uint64) (bool, error) {
	id := ""
	query := fmt.Sprintf("SELECT tx_id FROM %s WHERE tx_id = $1 AND idx = $2 AND is_deleted = false LIMIT 1;", db.table.Tokens)
	logger.Debug(query, txID, index)

	row := db.db.QueryRow(query, txID, index)
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.Wrapf(err, "error querying db")
	}
	return id == txID, nil
}

// UnspentTokensIterator returns an iterator over all unspent tokens
func (db *TokenDB) UnspentTokensIterator() (tdriver.UnspentTokensIterator, error) {
	var uti UnspentTokensIterator

	query := fmt.Sprintf("SELECT tx_id, idx, owner_raw, token_type, quantity FROM %s WHERE is_deleted = false", db.table.Tokens)
	logger.Debug(query)
	rows, err := db.db.Query(query)
	uti.txs = rows
	return &uti, err
}

// UnspentTokensIteratorBy returns an iterator of unspent tokens owned by the passed id and whose type is the passed on.
// The token type can be empty. In that case, tokens of any type are returned.
func (db *TokenDB) UnspentTokensIteratorBy(ownerEID, typ string) (tdriver.UnspentTokensIterator, error) {
	var uti UnspentTokensIterator

	var args []interface{}
	if ownerEID != "" {
		args = append(args, ownerEID)
	}
	if typ != "" {
		args = append(args, typ)
	}
	query := fmt.Sprintf("SELECT %s.tx_id, %s.idx, owner_raw, token_type, quantity FROM %s INNER JOIN %s ON %s.tx_id = %s.tx_id AND %s.idx = %s.idx AND %s.is_deleted = false",
		db.table.Tokens, db.table.Tokens, // select
		db.table.Tokens,                     // from
		db.table.Ownership,                  // inner join
		db.table.Tokens, db.table.Ownership, // .txid
		db.table.Tokens, db.table.Ownership, // .idx
		db.table.Tokens) // Unspent
	if ownerEID != "" {
		query += " AND enrollment_id = $1"
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
func (db *TokenDB) ListUnspentTokensBy(ownerEID, typ string) (*token.UnspentTokens, error) {
	logger.Debugf("List unspent token...")
	it, err := db.UnspentTokensIteratorBy(ownerEID, typ)
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
func (db *TokenDB) ListUnspentTokens() (*token.UnspentTokens, error) {
	logger.Debugf("List unspent token...")
	it, err := db.UnspentTokensIterator()
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
func (db *TokenDB) ListAuditTokens(ids ...*token.ID) ([]*token.Token, error) {
	if len(ids) == 0 {
		return []*token.Token{}, nil
	}
	args := make([]interface{}, 0)
	where := whereTokenIDs(&args, ids)

	query := fmt.Sprintf("SELECT tx_id, idx, owner_raw, token_type, quantity FROM %s WHERE %s", db.table.AuditTokens, where)
	logger.Debug(query, args)
	rows, err := db.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := make([]*token.Token, len(ids))
	counter := 0
	for rows.Next() {
		id := token.ID{}
		tok := token.Token{
			Owner: &token.Owner{
				Raw: []byte{},
			},
			Type:     "",
			Quantity: "",
		}
		if err := rows.Scan(&id.TxId, &id.Index, &tok.Owner.Raw, &tok.Type, &tok.Quantity); err != nil {
			return tokens, err
		}

		// the result is expected to be in order of the ids
		found := false
		for i := 0; i < len(ids); i++ {
			if ids[i].Equal(id) {
				tokens[i] = &tok
				found = true
				counter++
			}
		}
		if !found {
			return nil, errors.Errorf("retrieved wrong token [%s]", id)
		}
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}
	if counter == 0 {
		return nil, errors.Errorf("token not found for key [%s:%d]", ids[0].TxId, ids[0].Index)
	}
	if counter != len(ids) {
		for j, t := range tokens {
			if t == nil {
				return nil, errors.Errorf("token not found for key [%s:%d]", ids[j].TxId, ids[j].Index)
			}
		}
		panic("programming error: should not reach this point")
	}
	return tokens, nil
}

// ListHistoryIssuedTokens returns the list of issued tokens
func (db *TokenDB) ListHistoryIssuedTokens() (*token.IssuedTokens, error) {
	query := fmt.Sprintf("SELECT tx_id, idx, owner_raw, token_type, quantity, issuer_raw FROM %s", db.table.IssuedTokens)
	logger.Debug(query)
	rows, err := db.db.Query(query)
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

func (db *TokenDB) GetTokenOutputs(ids []*token.ID, callback tdriver.QueryCallbackFunc) error {
	tokens, err := db.getLedgerToken(ids)
	if err != nil {
		return err
	}
	for i := 0; i < len(ids); i++ {
		if err := callback(ids[i], tokens[i]); err != nil {
			return err
		}
	}
	return nil
}

// GetTokenInfos retrieves the token metadata for the passed ids.
// For each id, the callback is invoked to unmarshal the token metadata
func (db *TokenDB) GetTokenInfos(ids []*token.ID, callback tdriver.QueryCallbackFunc) error {
	if len(ids) == 0 {
		return nil
	}
	_, metas, err := db.getLedgerTokenAndMeta(ids)
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
func (db *TokenDB) GetTokenInfoAndOutputs(ids []*token.ID, callback tdriver.QueryCallback2Func) error {
	tokens, metas, err := db.getLedgerTokenAndMeta(ids)
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
func (db *TokenDB) GetAllTokenInfos(ids []*token.ID) ([][]byte, error) {
	if len(ids) == 0 {
		return [][]byte{}, nil
	}
	_, metas, err := db.getLedgerTokenAndMeta(ids)
	if err != nil {
		return metas, err
	}

	return metas, nil
}

func (db *TokenDB) getLedgerToken(ids []*token.ID) ([][]byte, error) {
	logger.Debugf("retrieve ledger tokens for [%s]", ids)
	tokens := make([][]byte, len(ids))
	if len(ids) == 0 {
		return tokens, nil
	}
	args := make([]interface{}, 0)
	where := whereTokenIDs(&args, ids)

	query := fmt.Sprintf("SELECT tx_id, idx, ledger FROM %s WHERE %s", db.table.Ledger, where)
	logger.Debug(query, args)
	rows, err := db.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counter := 0
	for rows.Next() {
		var tok []byte
		var id token.ID
		if err := rows.Scan(&id.TxId, &id.Index, &tok); err != nil {
			return nil, err
		}
		logger.Debugf("found ledger token [%s:%d] [%v]", id.TxId, id.Index, tok)
		// the result is expected to be in order of the ids
		found := false
		for i := 0; i < len(ids); i++ {
			if ids[i].Equal(id) {
				tokens[i] = tok
				found = true
				break
			}
		}
		if !found {
			return nil, errors.Errorf("retrieved wrong token [%s]", id)
		}
		counter++
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	if counter == 0 {
		return nil, errors.Errorf("token not found for key [%s:%d]", ids[0].TxId, ids[0].Index)
	}
	if counter != len(ids) {
		for j, t := range tokens {
			if t == nil {
				return nil, errors.Errorf("token not found for key [%s:%d]", ids[j].TxId, ids[j].Index)
			}
		}
		panic("programming error: should not reach this point")
	}

	return tokens, nil
}

func (db *TokenDB) getLedgerTokenAndMeta(ids []*token.ID) ([][]byte, [][]byte, error) {
	tokens := make([][]byte, len(ids))
	metas := make([][]byte, len(ids))
	if len(ids) == 0 {
		return tokens, metas, nil
	}
	args := make([]interface{}, 0)
	where := whereTokenIDs(&args, ids)

	query := fmt.Sprintf("SELECT tx_id, idx, ledger, ledger_metadata FROM %s WHERE %s", db.table.Ledger, where)
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
		found := false
		for i := 0; i < len(ids); i++ {
			if ids[i].Equal(id) {
				tokens[i] = tok
				metas[i] = metadata
				found = true
				break
			}
		}
		if !found {
			return nil, nil, errors.Errorf("retrieved wrong token [%s]", id)
		}
	}
	if err = rows.Err(); err != nil {
		return tokens, metas, err
	}
	return tokens, metas, nil
}

// GetTokens returns the list of tokens with their respective vault keys
func (db *TokenDB) GetTokens(inputs ...*token.ID) ([]string, []*token.Token, error) {
	if len(inputs) == 0 {
		return []string{}, []*token.Token{}, nil
	}
	args := make([]interface{}, 0)
	where := whereTokenIDs(&args, inputs)

	query := fmt.Sprintf("SELECT tx_id, idx, owner_raw, token_type, quantity FROM %s WHERE %s AND is_deleted = false", db.table.Tokens, where)
	logger.Debug(query, args)
	rows, err := db.db.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	tokens := make([]*token.Token, len(inputs))
	ids := make([]string, len(inputs))
	counter := 0
	for rows.Next() {
		tokID := token.ID{}
		var typ, quantity string
		var ownerRaw []byte
		err := rows.Scan(
			&tokID.TxId,
			&tokID.Index,
			&ownerRaw,
			&typ,
			&quantity,
		)
		if err != nil {
			return nil, tokens, err
		}
		tok := &token.Token{
			Owner:    &token.Owner{Raw: ownerRaw},
			Type:     typ,
			Quantity: quantity,
		}

		// The token keys are used to refer to tokens as stored in the world state by the tokenchaincode
		// so that they can be validated as inputs for the transaction
		id, err := keys.CreateTokenKey(tokID.TxId, tokID.Index)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed generating id key [%v]", tokID)
		}
		logger.Debugf("input [%s]-[%s]-[%s:%s]", inputs[counter], tokID, tok.Type, tok.Quantity)

		// put in the right position
		found := false
		for j := 0; j < len(inputs); j++ {
			if inputs[j].Equal(tokID) {
				ids[j] = id
				tokens[j] = tok
				logger.Debugf("set token at location [%s:%s]-[%d]", tok.Type, tok.Quantity, j)
				found = true
				break
			}
		}
		if !found {
			return nil, nil, errors.Errorf("retrieved wrong token [%s]", id)
		}

		counter++
	}
	logger.Debugf("found [%d] tokens, expected [%d]", counter, len(inputs))
	if err = rows.Err(); err != nil {
		return nil, tokens, err
	}
	if counter == 0 {
		return nil, nil, errors.Errorf("token not found for key [%s:%d]", inputs[0].TxId, inputs[0].Index)
	}
	if counter != len(inputs) {
		for j, t := range tokens {
			if t == nil {
				return nil, nil, errors.Errorf("token not found for key [%s:%d]", inputs[j].TxId, inputs[j].Index)
			}
		}
		panic("programming error: should not reach this point")
	}
	return ids, tokens, nil
}

// WhoDeletedTokens returns information about which transaction deleted the passed tokens.
// The bool array is an indicator used to tell if the token at a given position has been deleted or not
func (db *TokenDB) WhoDeletedTokens(inputs ...*token.ID) ([]string, []bool, error) {
	logger.Debugf("search first over token table [%s]...", inputs)
	who, deleted, err := db.whoDeleteTokens(db.table.Tokens, inputs...)
	if err != nil || len(who) != len(inputs) {
		logger.Debugf("search then over auditor token table [%s]...", inputs)
		return db.whoDeleteTokens(db.table.AuditTokens, inputs...)
	}
	return who, deleted, err
}

func (db *TokenDB) whoDeleteTokens(table string, inputs ...*token.ID) ([]string, []bool, error) {
	if len(inputs) == 0 {
		return []string{}, []bool{}, nil
	}
	args := make([]interface{}, 0)
	where := whereTokenIDs(&args, inputs)

	query := fmt.Sprintf("SELECT tx_id, idx, spent_by, is_deleted FROM %s WHERE %s", table, where)
	logger.Debug(query, args)
	rows, err := db.db.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	spentBy := make([]string, len(inputs))
	isSpent := make([]bool, len(inputs))
	found := make([]bool, len(inputs))

	counter := 0
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
				found[i] = true
				break // stop searching for this id but continue looping over rows
			}
		}
		counter++
	}
	logger.Debugf("found [%d] records, expected [%d]", counter, len(inputs))
	if err = rows.Err(); err != nil {
		return nil, isSpent, err
	}
	if counter == 0 {
		return nil, nil, errors.Errorf("token not found for key [%s:%d]", inputs[0].TxId, inputs[0].Index)
	}
	if counter != len(inputs) {
		for j, f := range found {
			if !f {
				return nil, nil, errors.Errorf("token not found for key [%s:%d]", inputs[j].TxId, inputs[j].Index)
			}
		}
		panic("programming error: should not reach this point")
	}
	return spentBy, isSpent, nil

}

func (db *TokenDB) StorePublicParams(raw []byte) error {
	now := time.Now().UTC()
	query := fmt.Sprintf("INSERT INTO %s (raw, stored_at) VALUES ($1, $2)", db.table.PublicParams)
	logger.Debug(query, fmt.Sprintf("store public parameters (%d bytes), %v", len(raw), now))

	_, err := db.db.Exec(query, raw, now)
	return err
}

func (db *TokenDB) PublicParams() ([]byte, error) {
	var params []byte
	query := fmt.Sprintf("SELECT raw FROM %s ORDER BY stored_at DESC LIMIT 1;", db.table.PublicParams)
	logger.Debug(query)

	row := db.db.QueryRow(query)
	err := row.Scan(&params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "error querying db")
	}
	return params, nil
}

func (db *TokenDB) StoreCertifications(certifications map[*token.ID][]byte) error {
	now := time.Now().UTC()
	query := fmt.Sprintf("INSERT INTO %s (token_id, tx_id, tx_index, certification, stored_at) VALUES ($1, $2, $3, $4, $5)", db.table.Certifications)

	tx, err := db.db.Begin()
	if err != nil {
		return errors.New("failed starting a transaction")
	}
	defer tx.Rollback()
	for tokenID, certification := range certifications {
		if tokenID == nil {
			return errors.Errorf("invalid token-id, cannot be nil")
		}
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

func (db *TokenDB) ExistsCertification(tokenID *token.ID) bool {
	if tokenID == nil {
		return false
	}
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

func (db *TokenDB) GetCertifications(ids []*token.ID, callback func(*token.ID, []byte) error) error {
	if len(ids) == 0 {
		// nothing to do here
		return nil
	}

	// build query
	conditions, tokenIDs, err := certificationsQuerySql(ids)
	if err != nil {
		return err
	}
	query := fmt.Sprintf("SELECT tx_id, tx_index, certification FROM %s WHERE ", db.table.Certifications) + conditions

	rows, err := db.db.Query(query, tokenIDs...)
	if err != nil {
		return errors.Wrapf(err, "failed to query")
	}
	defer rows.Close()

	certifications := make([][]byte, len(ids))
	counter := 0
	for rows.Next() {
		var certification []byte
		var id token.ID
		if err := rows.Scan(&id.TxId, &id.Index, &certification); err != nil {
			return err
		}
		// the callback is expected to be called in order of the ids
		if len(certification) == 0 {
			return errors.Errorf("empty certification for [%s]", id.String())
		}
		for i := 0; i < len(ids); i++ {
			if *ids[i] == id {
				certifications[i] = certification
				break
			}
		}
		counter++
	}

	if err = rows.Err(); err != nil {
		return err
	}
	if counter != len(ids) {
		return errors.Errorf("not all tokens are certified")
	}

	for i, certification := range certifications {
		if err := callback(ids[i], certification); err != nil {
			return errors.WithMessagef(err, "failed callback for [%s]", ids[i])
		}
	}

	return nil
}

func (db *TokenDB) GetSchema() string {
	// owner_raw is as1 encoded Type(string), Identity([]byte) (see token/core/identity/owner.go).
	// If Type is "htlc", Identity is a json encoded Script.
	return fmt.Sprintf(`
		-- Tokens
		CREATE TABLE IF NOT EXISTS %s (
			tx_id TEXT NOT NULL,
			idx INT NOT NULL,
			amount BIGINT NOT NULL,
			token_type TEXT NOT NULL,
			quantity TEXT NOT NULL,
			issuer_raw BYTEA,
			owner_raw BYTEA NOT NULL,
			ledger BYTEA NOT NULL,
			ledger_metadata BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL,
			is_deleted BOOL NOT NULL DEFAULT false,
			spent_by TEXT NOT NULL DEFAULT '',
			spent_at TIMESTAMP,
			PRIMARY KEY (tx_id, idx)
		);
		CREATE INDEX IF NOT EXISTS idx_spent_%s ON %s ( is_deleted );

		-- Audit tokens
		CREATE TABLE IF NOT EXISTS %s (
			tx_id TEXT NOT NULL,
			idx INT NOT NULL,
			amount BIGINT NOT NULL,
			token_type TEXT NOT NULL,
			quantity TEXT NOT NULL,
			issuer_raw BYTEA,
			owner_raw BYTEA NOT NULL,
			ledger BYTEA NOT NULL,
			ledger_metadata BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL,
			is_deleted BOOL NOT NULL DEFAULT false,
			spent_by TEXT NOT NULL DEFAULT '',
			spent_at TIMESTAMP,
			PRIMARY KEY (tx_id, idx)
		);

		-- Issued tokens
		CREATE TABLE IF NOT EXISTS %s (
			tx_id TEXT NOT NULL,
			idx INT NOT NULL,
			amount BIGINT NOT NULL,
			token_type TEXT NOT NULL,
			quantity TEXT NOT NULL,
			owner_raw BYTEA NOT NULL,
			issuer_raw BYTEA NOT NULL,
			ledger BYTEA NOT NULL,
			ledger_metadata BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL,
			PRIMARY KEY (tx_id, idx)
		);

		-- Ownership
		CREATE TABLE IF NOT EXISTS %s (
			tx_id TEXT NOT NULL,
			idx INT NOT NULL,
			enrollment_id TEXT NOT NULL,
			PRIMARY KEY (tx_id, idx, enrollment_id)
		);

		-- Public Parameters
		CREATE TABLE IF NOT EXISTS %s (
			raw BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL,
			PRIMARY KEY (raw)
		);

		-- Ledger
		CREATE TABLE IF NOT EXISTS %s (
			tx_id TEXT NOT NULL,
			idx INT NOT NULL,
			ledger BYTEA NOT NULL,
			ledger_metadata BYTEA NOT NULL,
			PRIMARY KEY (tx_id, idx)
		);

		CREATE TABLE IF NOT EXISTS %s (
			token_id TEXT NOT NULL PRIMARY KEY,
			tx_id TEXT NOT NULL,
			tx_index INT NOT NULL,
			certification BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL
		);
		`,
		db.table.Tokens,
		db.table.Tokens,
		db.table.Tokens,
		db.table.AuditTokens,
		db.table.IssuedTokens,
		db.table.Ownership,
		db.table.PublicParams,
		db.table.Ledger,
		db.table.Certifications,
	)
}

func (db *TokenDB) Close() {
	db.db.Close()
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
