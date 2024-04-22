/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"database/sql"
	"fmt"
	"runtime/debug"
	"strings"
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
	PublicParams   string
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

func NewTokenDB(db *sql.DB, tablePrefix string, createSchema bool) (*TokenDB, error) {
	tables, err := getTableNames(tablePrefix)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table names")
	}

	tokenDB := newTokenDB(db, tokenTables{
		Tokens:         tables.Tokens,
		Ownership:      tables.Ownership,
		PublicParams:   tables.PublicParams,
		Certifications: tables.Certifications,
	})
	if createSchema {
		if err = initSchema(db, tokenDB.GetSchema()); err != nil {
			return nil, err
		}
	}
	return tokenDB, nil
}

func (db *TokenDB) StoreToken(tr driver.TokenRecord, owners []string) (err error) {
	tx, err := db.NewTokenDBTransaction()
	if err != nil {
		return
	}
	if err = tx.StoreToken(tr, owners); err != nil {
		if err1 := tx.Rollback(); err1 != nil {
			logger.Errorf("error rolling back: %s", err1.Error())
		}
		return
	}
	if err = tx.Commit(); err != nil {
		return
	}
	return nil
}

// DeleteTokens deletes multiple tokens at the same time (when spent, invalid or expired)
func (db *TokenDB) DeleteTokens(deletedBy string, ids ...*token.ID) error {
	logger.Debugf("delete tokens [%s][%v]", deletedBy, ids)
	if len(ids) == 0 {
		return nil
	}
	args := []interface{}{deletedBy, time.Now().UTC()}
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
	query := fmt.Sprintf("SELECT tx_id FROM %s WHERE tx_id = $1 AND idx = $2 AND is_deleted = false AND owner = true LIMIT 1;", db.table.Tokens)
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
	return db.UnspentTokensIteratorBy("", "")
}

// UnspentTokensIteratorBy returns an iterator of unspent tokens owned by the passed id and whose type is the passed on.
// The token type can be empty. In that case, tokens of any type are returned.
func (db *TokenDB) UnspentTokensIteratorBy(ownerEID, typ string) (tdriver.UnspentTokensIterator, error) {
	where, join, args := tokenQuerySql(ownerEID, driver.QueryTokenDetailsParams{TokenType: typ}, db.table.Tokens, db.table.Ownership)
	query := fmt.Sprintf("SELECT %s.tx_id, %s.idx, owner_raw, token_type, quantity FROM %s %s %s",
		db.table.Tokens, db.table.Tokens, db.table.Tokens, join, where)

	logger.Debug(query, args)
	rows, err := db.db.Query(query, args...)

	return &UnspentTokensIterator{txs: rows}, err
}

// ListUnspentTokensBy returns the list of unspent tokens, filtered by owner and token type
func (db *TokenDB) ListUnspentTokensBy(ownerEID, typ string) (*token.UnspentTokens, error) {
	logger.Debugf("List unspent token by...")
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

	query := fmt.Sprintf("SELECT tx_id, idx, owner_raw, token_type, quantity FROM %s WHERE %s AND auditor = true", db.table.Tokens, where)
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
	query := fmt.Sprintf("SELECT tx_id, idx, owner_raw, token_type, quantity, issuer_raw FROM %s WHERE issuer = true", db.table.Tokens)
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
func (db *TokenDB) GetTokenInfos(ids []*token.ID) ([][]byte, error) {
	return db.GetAllTokenInfos(ids)
}

// GetTokenInfoAndOutputs retrieves both the token output and information for the passed ids.
func (db *TokenDB) GetTokenInfoAndOutputs(ids []*token.ID) ([]string, [][]byte, [][]byte, error) {
	tokens, metas, err := db.getLedgerTokenAndMeta(ids)
	if err != nil {
		return nil, nil, nil, err
	}
	outputIDs := make([]string, len(ids))
	for i := 0; i < len(ids); i++ {
		outputID, err := keys.CreateTokenKey(ids[i].TxId, ids[i].Index)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "error creating output ID: %v", ids[i])
		}
		outputIDs[i] = outputID
	}
	return outputIDs, tokens, metas, nil
}

// GetAllTokenInfos retrieves the token information for the passed ids.
func (db *TokenDB) GetAllTokenInfos(ids []*token.ID) ([][]byte, error) {
	if len(ids) == 0 {
		return [][]byte{}, nil
	}
	_, metas, err := db.getLedgerTokenAndMeta(ids)
	return metas, err
}

func (db *TokenDB) getLedgerToken(ids []*token.ID) ([][]byte, error) {
	logger.Debugf("retrieve ledger tokens for [%s]", ids)
	if len(ids) == 0 {
		return [][]byte{}, nil
	}
	args := make([]interface{}, 0)
	where := whereTokenIDs(&args, ids)

	query := fmt.Sprintf("SELECT tx_id, idx, ledger FROM %s WHERE %s", db.table.Tokens, where)
	logger.Debug(query, args)
	rows, err := db.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokenMap := make(map[string][]byte, len(ids))
	for rows.Next() {
		var tok []byte
		var id token.ID
		if err := rows.Scan(&id.TxId, &id.Index, &tok); err != nil {
			return nil, err
		}
		tokenMap[id.String()] = tok
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	tokens := make([][]byte, len(ids))
	for i, id := range ids {
		if tok, ok := tokenMap[id.String()]; !ok || tok == nil {
			return nil, errors.Errorf("token not found for key [%s]", id)
		} else if len(tok) == 0 {
			return nil, errors.Errorf("empty token found for key [%s]", id)
		} else {
			tokens[i] = tok
		}
	}
	return tokens, nil
}

func (db *TokenDB) getLedgerTokenAndMeta(ids []*token.ID) ([][]byte, [][]byte, error) {
	if len(ids) == 0 {
		return [][]byte{}, [][]byte{}, nil
	}
	args := make([]interface{}, 0)
	where := whereTokenIDs(&args, ids)

	query := fmt.Sprintf("SELECT tx_id, idx, ledger, ledger_metadata FROM %s WHERE %s", db.table.Tokens, where)
	logger.Debug(query, args)
	rows, err := db.db.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	infoMap := make(map[string][2][]byte, len(ids))
	for rows.Next() {
		var tok []byte
		var metadata []byte
		var id token.ID
		if err := rows.Scan(&id.TxId, &id.Index, &tok, &metadata); err != nil {
			return nil, nil, err
		}
		infoMap[id.String()] = [2][]byte{tok, metadata}
	}
	if err = rows.Err(); err != nil {
		return nil, nil, err
	}
	tokens := make([][]byte, len(ids))
	metas := make([][]byte, len(ids))
	for i, id := range ids {
		if info, ok := infoMap[id.String()]; !ok {
			return nil, nil, errors.Errorf("token/metadata not found for [%s]", id)
		} else {
			tokens[i] = info[0]
			metas[i] = info[1]
		}
	}
	return tokens, metas, nil
}

// GetTokens returns the owned tokens and their identifier keys for the passed ids.
func (db *TokenDB) GetTokens(inputs ...*token.ID) ([]string, []*token.Token, error) {
	if len(inputs) == 0 {
		return []string{}, []*token.Token{}, nil
	}
	args := make([]interface{}, 0)
	where := whereTokenIDs(&args, inputs)

	query := fmt.Sprintf("SELECT tx_id, idx, owner_raw, token_type, quantity FROM %s WHERE %s AND is_deleted = false AND owner = true", db.table.Tokens, where)
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

// QueryTokensDetails returns details about owner tokens, regardless if they have been spent or not.
// Filters work cumulatively and may be left empty. If a token is owned by two enrollmentIDs and there
// is no filter on enrollmentID, the token will be returned twice (once for each owner).
func (db *TokenDB) QueryTokenDetails(ownerEID string, params driver.QueryTokenDetailsParams) ([]driver.TokenDetails, error) {
	where, join, args := tokenQuerySql(ownerEID, params, db.table.Tokens, db.table.Ownership)

	query := fmt.Sprintf("SELECT %s.tx_id, %s.idx, owner_raw, owner_type, enrollment_id, token_type, amount, is_deleted, spent_by, stored_at FROM %s %s %s",
		db.table.Tokens, db.table.Tokens, db.table.Tokens, join, where)
	logger.Debug(query, args)
	rows, err := db.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	deets := []driver.TokenDetails{}
	for rows.Next() {
		td := driver.TokenDetails{}
		if err := rows.Scan(
			&td.TxID,
			&td.Index,
			&td.OwnerRaw,
			&td.OwnerType,
			&td.OwnerEnrollment,
			&td.Type,
			&td.Amount,
			&td.IsSpent,
			&td.SpentBy,
			&td.StoredAt,
		); err != nil {
			return deets, err
		}
		deets = append(deets, td)
	}
	logger.Debugf("found [%d] tokens", len(deets))
	if err = rows.Err(); err != nil {
		return deets, err
	}
	return deets, nil
}

// WhoDeletedTokens returns information about which transaction deleted the passed tokens.
// The bool array is an indicator used to tell if the token at a given position has been deleted or not
func (db *TokenDB) WhoDeletedTokens(inputs ...*token.ID) ([]string, []bool, error) {
	if len(inputs) == 0 {
		return []string{}, []bool{}, nil
	}
	args := []any{}
	where := whereTokenIDs(&args, inputs)

	query := fmt.Sprintf("SELECT tx_id, idx, spent_by, is_deleted FROM %s WHERE %s", db.table.Tokens, where)
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

func (db *TokenDB) StoreCertifications(certifications map[*token.ID][]byte) (err error) {
	now := time.Now().UTC()
	query := fmt.Sprintf("INSERT INTO %s (tx_id, idx, certification, stored_at) VALUES ($1, $2, $3, $4)", db.table.Certifications)

	tx, err := db.db.Begin()
	if err != nil {
		return errors.New("failed starting a transaction")
	}
	defer tx.Rollback()

	for tokenID, certification := range certifications {
		if tokenID == nil {
			return errors.Errorf("invalid token-id, cannot be nil")
		}
		logger.Debug(query, fmt.Sprintf("(%d bytes)", len(certification)), now)
		if _, err = tx.Exec(query, tokenID.TxId, tokenID.Index, certification, now); err != nil {
			return tokenDBError(err)
		}
	}
	if err = tx.Commit(); err != nil {
		return errors.Wrap(err, "failed committing certifications")
	}
	return
}

func (db *TokenDB) ExistsCertification(tokenID *token.ID) bool {
	if tokenID == nil {
		return false
	}
	args := []any{}
	where := whereTokenIDs(&args, []*token.ID{tokenID})

	query := fmt.Sprintf("SELECT certification FROM %s WHERE %s", db.table.Certifications, where)
	logger.Debug(query, args)
	row := db.db.QueryRow(query, args...)

	var certification []byte
	if err := row.Scan(&certification); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false
		}
		logger.Warnf("tried to check certification existence for token id %s, err %s", tokenID, err)
		return false
	}
	result := len(certification) != 0
	if !result {
		logger.Warnf("tried to check certification existence for token id %s, got an empty certification", tokenID)
	}
	return result
}

func (db *TokenDB) GetCertifications(ids []*token.ID) ([][]byte, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	args := []any{}
	where := whereTokenIDs(&args, ids)
	query := fmt.Sprintf("SELECT tx_id, idx, certification FROM %s WHERE %s ", db.table.Certifications, where)

	rows, err := db.db.Query(query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query")
	}
	defer rows.Close()

	certificationMap := make(map[string][]byte, len(ids))
	for rows.Next() {
		var certification []byte
		var id token.ID
		if err := rows.Scan(&id.TxId, &id.Index, &certification); err != nil {
			return nil, err
		}
		certificationMap[id.String()] = certification
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	certifications := make([][]byte, len(ids))
	for i, id := range ids {
		if cert, ok := certificationMap[id.String()]; !ok {
			return nil, errors.Errorf("token %s was not certified", id)
		} else if len(cert) == 0 {
			return nil, errors.Errorf("empty certification for [%s]", id)
		} else {
			certifications[i] = cert
		}
	}
	return certifications, nil
}

func (db *TokenDB) GetSchema() string {
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
			owner_type TEXT NOT NULL,
			ledger BYTEA NOT NULL,
			ledger_metadata BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL,
			is_deleted BOOL NOT NULL DEFAULT false,
			spent_by TEXT NOT NULL DEFAULT '',
			spent_at TIMESTAMP,
			owner BOOL NOT NULL DEFAULT false,
			auditor BOOL NOT NULL DEFAULT false,
			issuer BOOL NOT NULL DEFAULT false,
			PRIMARY KEY (tx_id, idx)
		);
		CREATE INDEX IF NOT EXISTS idx_spent_%s ON %s ( is_deleted, owner );
		CREATE INDEX IF NOT EXISTS idx_tx_id_%s ON %s ( tx_id );

		-- Ownership
		CREATE TABLE IF NOT EXISTS %s (
			tx_id TEXT NOT NULL,
			idx INT NOT NULL,
			enrollment_id TEXT NOT NULL,
			PRIMARY KEY (tx_id, idx, enrollment_id)
			FOREIGN KEY (tx_id, idx) REFERENCES %s
		);

		-- Public Parameters
		CREATE TABLE IF NOT EXISTS %s (
			raw BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL PRIMARY KEY
		);

		-- Certifications
		CREATE TABLE IF NOT EXISTS %s (
			tx_id TEXT NOT NULL,
			idx INT NOT NULL,
			certification BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL,
			PRIMARY KEY (tx_id, idx)
			FOREIGN KEY (tx_id, idx) REFERENCES %s
		);
		`,
		db.table.Tokens,
		db.table.Tokens, db.table.Tokens,
		db.table.Tokens, db.table.Tokens,
		db.table.Ownership, db.table.Tokens,
		db.table.PublicParams,
		db.table.Certifications, db.table.Tokens,
	)
}

func (db *TokenDB) Close() {
	db.db.Close()
}

func (db *TokenDB) NewTokenDBTransaction() (driver.TokenDBTransaction, error) {
	tx, err := db.db.Begin()
	if err != nil {
		return nil, errors.New("failed starting a db transaction")
	}
	return &TokenTransaction{db: db, tx: tx}, nil
}

type TokenTransaction struct {
	db *TokenDB
	tx *sql.Tx
}

func (t *TokenTransaction) TransactionExists(id string) (bool, error) {
	query := fmt.Sprintf("SELECT tx_id FROM %s WHERE tx_id=$1 LIMIT 1;", t.db.table.Tokens)
	logger.Debug(query, id)

	row := t.tx.QueryRow(query, id)
	var certification []byte
	if err := row.Scan(&certification); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		logger.Warnf("tried to check transaction existence for id %s, err %s", id, err)
		return false, err
	}
	result := len(certification) != 0
	if !result {
		logger.Warnf("tried to check transaction existence for id %s, got nothing", id)
	}
	return result, nil

}

func (t *TokenTransaction) GetToken(txID string, index uint64, includeDeleted bool) (*token.Token, []string, error) {
	where, join, args := tokenQuerySql("", driver.QueryTokenDetailsParams{
		IDs:            []*token.ID{{TxId: txID, Index: index}},
		IncludeDeleted: includeDeleted,
	}, t.db.table.Tokens, t.db.table.Ownership)
	query := fmt.Sprintf("SELECT owner_raw, token_type, quantity, enrollment_id FROM %s %s %s", t.db.table.Tokens, join, where)
	logger.Debug(query, args)
	rows, err := t.tx.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var raw []byte
	var tokenType string
	var quantity string
	owners := []string{}
	for rows.Next() {
		var owner string
		if err := rows.Scan(&raw, &tokenType, &quantity, &owner); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, owners, nil
			}
			return nil, owners, err
		}
		if len(owner) > 0 {
			owners = append(owners, owner)
		}
	}
	if rows.Err() != nil {
		return nil, nil, rows.Err()
	}
	if len(raw) == 0 {
		return nil, owners, nil
	}
	return &token.Token{
		Owner: &token.Owner{
			Raw: raw,
		},
		Type:     tokenType,
		Quantity: quantity,
	}, owners, nil
}

func (t *TokenTransaction) OwnersOf(txID string, index uint64) ([]string, error) {
	args := make([]interface{}, 0)
	tokenIDs := []*token.ID{{TxId: txID, Index: index}}
	where := whereTokenIDs(&args, tokenIDs)
	query := fmt.Sprintf("SELECT enrollment_id FROM %s WHERE %s", t.db.table.Ownership, where)
	logger.Debug(query, args)
	rows, err := t.tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var owners []string
	for rows.Next() {
		var owner string
		if err := rows.Scan(&owner); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, err
		}
		owners = append(owners, owner)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return owners, nil
}

func (t *TokenTransaction) Delete(txID string, index uint64, deletedBy string) error {
	logger.Debugf("delete token [%s:%d:%s]", txID, index, deletedBy)
	// We don't delete audit tokens, and we keep the 'ownership' relation.
	now := time.Now().UTC()
	query := fmt.Sprintf("UPDATE %s SET is_deleted = true, spent_by = $1, spent_at = $2 WHERE tx_id = $3 AND idx = $4;", t.db.table.Tokens)
	logger.Debug(query, deletedBy, now, txID, index)
	if _, err := t.tx.Exec(query, deletedBy, now, txID, index); err != nil {
		return errors.Wrapf(err, "error setting token to deleted [%s]", txID)
	}
	return nil
}

func (t *TokenTransaction) StoreToken(tr driver.TokenRecord, owners []string) error {
	logger.Debugf("store record [%s:%d,%v] in table [%s]", tr.TxID, tr.Index, owners, t.db.table.Tokens)

	// Store token
	now := time.Now().UTC()
	query := fmt.Sprintf("INSERT INTO %s (tx_id, idx, issuer_raw, owner_raw, owner_type, ledger, ledger_metadata, token_type, quantity, amount, stored_at, owner, auditor, issuer) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)", t.db.table.Tokens)
	logger.Debug(query, tr.TxID, tr.Index, len(tr.IssuerRaw), len(tr.OwnerRaw), tr.OwnerType, len(tr.Ledger), len(tr.LedgerMetadata), tr.Type, tr.Quantity, tr.Amount, now, tr.Owner, tr.Auditor, tr.Issuer)
	if _, err := t.tx.Exec(query, tr.TxID, tr.Index, tr.IssuerRaw, tr.OwnerRaw, tr.OwnerType, tr.Ledger, tr.LedgerMetadata, tr.Type, tr.Quantity, tr.Amount, now, tr.Owner, tr.Auditor, tr.Issuer); err != nil {
		logger.Errorf("error storing token [%s] in table [%s]: [%s][%s]", tr.TxID, t.db.table.Tokens, err, string(debug.Stack()))
		return errors.Wrapf(err, "error storing token [%s] in table [%s]", tr.TxID, t.db.table.Tokens)
	}

	// Store ownership
	for _, eid := range owners {
		query = fmt.Sprintf("INSERT INTO %s (tx_id, idx, enrollment_id) VALUES ($1, $2, $3)", t.db.table.Ownership)
		logger.Debug(query, tr.TxID, tr.Index, eid)
		if _, err := t.tx.Exec(query, tr.TxID, tr.Index, eid); err != nil {
			return errors.Wrapf(err, "error storing token ownership [%s]", tr.TxID)
		}
	}

	return nil
}

func (t *TokenTransaction) Commit() error {
	return t.tx.Commit()
}

func (t *TokenTransaction) Rollback() error {
	return t.tx.Rollback()
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

func tokenDBError(err error) error {
	if err == nil {
		return nil
	}
	logger.Error(err)
	e := strings.ToLower(err.Error())
	if strings.Contains(e, "foreign key constraint") {

		return driver.ErrTokenDoesNotExist
	}
	return err
}
