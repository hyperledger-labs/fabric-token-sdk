/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"database/sql"
	"encoding/base64"
	errors2 "errors"
	"fmt"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	tdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"go.opentelemetry.io/otel/trace"
)

type tokenTables struct {
	Tokens         string
	Ownership      string
	PublicParams   string
	Certifications string
}

func NewTokenDB(readDB, writeDB *sql.DB, opts NewDBOpts, ci TokenInterpreter) (driver.TokenDB, error) {
	tables, err := GetTableNames(opts.TablePrefix)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table names")
	}

	tokenDB := newTokenDB(readDB, writeDB, tokenTables{
		Tokens:         tables.Tokens,
		Ownership:      tables.Ownership,
		PublicParams:   tables.PublicParams,
		Certifications: tables.Certifications,
	}, ci)
	if opts.CreateSchema {
		if err = common.InitSchema(writeDB, tokenDB.GetSchema()); err != nil {
			return nil, err
		}
	}
	return tokenDB, nil
}

type TokenDB struct {
	readDB  *sql.DB
	writeDB *sql.DB
	table   tokenTables
	ci      TokenInterpreter

	sttMutex              sync.RWMutex
	supportedTokenFormats []token.Format
}

func newTokenDB(readDB, writeDB *sql.DB, tables tokenTables, ci TokenInterpreter) *TokenDB {
	return &TokenDB{
		readDB:  readDB,
		writeDB: writeDB,
		table:   tables,
		ci:      ci,
	}
}

func (db *TokenDB) StoreToken(tr driver.TokenRecord, owners []string) (err error) {
	tx, err := db.NewTokenDBTransaction()
	if err != nil {
		return
	}
	if err = tx.StoreToken(context.TODO(), tr, owners); err != nil {
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
	cond := db.ci.HasTokens("tx_id", "idx", ids...)
	args := append([]any{true, deletedBy, time.Now().UTC()}, cond.Params()...)
	offset := 4
	where := cond.ToString(&offset)

	query, err := NewUpdate(db.table.Tokens).Set("is_deleted, spent_by, spent_at").Where(where).Compile()
	if err != nil {
		return errors.Wrapf(err, "failed to update tokens")
	}
	// query := fmt.Sprintf("UPDATE %s SET is_deleted = true, spent_by = $1, spent_at = $2 WHERE %s", db.table.Tokens, where)
	logger.Debug(query, args)
	if _, err := db.writeDB.Exec(query, args...); err != nil {
		return errors.Wrapf(err, "error setting tokens to deleted [%v]", ids)
	}
	return nil
}

// IsMine just checks if the token is in the local storage and not deleted
func (db *TokenDB) IsMine(txID string, index uint64) (bool, error) {
	id := ""
	query, err := NewSelect("tx_id").
		From(db.table.Tokens).
		Where("tx_id = $1 AND idx = $2 AND is_deleted = false AND owner = true LIMIT 1").
		Compile()
	if err != nil {
		return false, errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query, txID, index)

	row := db.readDB.QueryRow(query, txID, index)
	if err := row.Scan(&id); err != nil {
		if errors2.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, errors.Wrapf(err, "error querying db")
	}
	return id == txID, nil
}

// UnspentTokensIterator returns an iterator over all unspent tokens
func (db *TokenDB) UnspentTokensIterator() (tdriver.UnspentTokensIterator, error) {
	return db.UnspentTokensIteratorBy(context.TODO(), "", "")
}

// UnspentTokensIteratorBy returns an iterator of unspent tokens owned by the passed id and whose type is the passed on.
// The token type can be empty. In that case, tokens of any type are returned.
func (db *TokenDB) UnspentTokensIteratorBy(ctx context.Context, walletID string, tokenType token.Type) (tdriver.UnspentTokensIterator, error) {
	span := trace.SpanFromContext(ctx)
	where, args := common.Where(db.ci.HasTokenDetails(driver.QueryTokenDetailsParams{
		WalletID:  walletID,
		TokenType: tokenType,
	}, db.table.Tokens))
	join := joinOnTokenID(db.table.Tokens, db.table.Ownership)

	query, err := NewSelect(
		fmt.Sprintf("%s.tx_id, %s.idx, owner_raw, token_type, quantity", db.table.Tokens, db.table.Tokens),
	).From(db.table.Tokens, join).Where(where).Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query, args)
	span.AddEvent("start_query", tracing.WithAttributes(tracing.String(QueryLabel, query)))
	rows, err := db.readDB.Query(query, args...)
	span.AddEvent("end_query")

	return &UnspentTokensIterator{txs: rows}, err
}

// SpendableTokensIteratorBy returns the minimum information about the tokens needed for the selector
func (db *TokenDB) SpendableTokensIteratorBy(ctx context.Context, walletID string, typ token.Type) (tdriver.SpendableTokensIterator, error) {
	span := trace.SpanFromContext(ctx)
	where, args := common.Where(db.ci.HasTokenDetails(driver.QueryTokenDetailsParams{
		WalletID:           walletID,
		TokenType:          typ,
		Spendable:          driver.SpendableOnly,
		LedgerTokenFormats: db.getSupportedTokenFormats(),
	}, ""))

	query, err := NewSelect("tx_id, idx, token_type, quantity, owner_wallet_id").From(db.table.Tokens).Where(where).Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile query")
	}
	logger.Warn(query, args)
	span.AddEvent("start_query", tracing.WithAttributes(tracing.String(QueryLabel, query)))
	rows, err := db.readDB.Query(query, args...)
	span.AddEvent("end_query")
	if err != nil {
		return nil, errors.Wrapf(err, "error querying db")
	}
	return &UnspentTokensInWalletIterator{txs: rows}, nil
}

// UnsupportedTokensIteratorBy returns the minimum information for upgrade about the tokens that are not supported
func (db *TokenDB) UnsupportedTokensIteratorBy(ctx context.Context, walletID string, tokenType token.Type) (tdriver.UnsupportedTokensIterator, error) {
	// first select all the distinct ledger types
	includeFormats, err := db.unspendableTokenFormats(ctx, walletID, tokenType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get unspendable token formats")
	}
	logger.Debugf("after filtering we have [%v]", includeFormats)

	span := trace.SpanFromContext(ctx)
	// now, select the tokens with the list of ledger tokens
	where, args := common.Where(db.ci.HasTokenDetails(driver.QueryTokenDetailsParams{
		WalletID:           walletID,
		TokenType:          tokenType,
		Spendable:          driver.Any,
		LedgerTokenFormats: includeFormats,
	}, ""))
	query, err := NewSelect("tx_id, idx, ledger, ledger_metadata, ledger_type").From(db.table.Tokens).Where(where).Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query, args)
	span.AddEvent("start_query", tracing.WithAttributes(tracing.String(QueryLabel, query)))
	rows, err := db.readDB.Query(query, args...)
	span.AddEvent("end_query")
	if err != nil {
		return nil, errors.Wrapf(err, "error querying db")
	}
	return &UnspendableTokensInWalletIterator{txs: rows}, nil
}

// Balance returns the sun of the amounts, with 64 bits of precision, of the tokens with type and EID equal to those passed as arguments.
func (db *TokenDB) Balance(walletID string, typ token.Type) (uint64, error) {
	return db.balance(driver.QueryTokenDetailsParams{
		WalletID:  walletID,
		TokenType: typ,
	})
}

func (db *TokenDB) balance(opts driver.QueryTokenDetailsParams) (uint64, error) {
	where, args := common.Where(db.ci.HasTokenDetails(opts, db.table.Tokens))
	join := joinOnTokenID(db.table.Tokens, db.table.Ownership)
	query, err := NewSelect("SUM(amount)").From(db.table.Tokens, join).Where(where).Compile()
	if err != nil {
		return 0, errors.Wrapf(err, "failed to compile query")
	}

	logger.Debug(query, args)
	row := db.readDB.QueryRow(query, args...)
	var sum *uint64
	if err := row.Scan(&sum); err != nil {
		if errors.HasCause(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, errors.Wrapf(err, "error querying db")
	}
	if sum == nil {
		return 0, nil
	}
	return *sum, nil
}

// ListUnspentTokensBy returns the list of unspent tokens, filtered by owner and token type
func (db *TokenDB) ListUnspentTokensBy(walletID string, typ token.Type) (*token.UnspentTokens, error) {
	logger.Debugf("list unspent token by [%s,%s]", walletID, typ)
	it, err := db.UnspentTokensIteratorBy(context.TODO(), walletID, typ)
	if err != nil {
		return nil, err
	}
	tokens, err := collections.ToSlice(it)
	if err != nil {
		return nil, err
	}
	return &token.UnspentTokens{Tokens: tokens}, nil
}

// ListUnspentTokens returns the list of unspent tokens
func (db *TokenDB) ListUnspentTokens() (*token.UnspentTokens, error) {
	logger.Debugf("list unspent tokens...")
	it, err := db.UnspentTokensIterator()
	if err != nil {
		return nil, err
	}
	tokens, err := collections.ToSlice(it)
	if err != nil {
		return nil, err
	}
	return &token.UnspentTokens{Tokens: tokens}, nil
}

// ListAuditTokens returns the audited tokens associated to the passed ids
func (db *TokenDB) ListAuditTokens(ids ...*token.ID) ([]*token.Token, error) {
	return db.listTokens(ids, common.ConstCondition("auditor = true"))
}

// GetTokens returns the owned tokens and their identifier keys for the passed ids.
func (db *TokenDB) GetTokens(inputs ...*token.ID) ([]*token.Token, error) {
	return db.listTokens(inputs, common.ConstCondition("is_deleted = false"), common.ConstCondition("owner = true"))
}

func (db *TokenDB) listTokens(inputs []*token.ID, conds ...common.Condition) ([]*token.Token, error) {
	if len(inputs) == 0 {
		return []*token.Token{}, nil
	}
	where, args := common.Where(db.ci.And(append(conds, db.ci.HasTokens("tx_id", "idx", inputs...))...))
	query, err := NewSelect("tx_id, idx, owner_raw, token_type, quantity").From(db.table.Tokens).Where(where).Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query, args)
	rows, err := db.readDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer Close(rows)

	tokens := make([]*token.Token, len(inputs))
	counter := 0
	for rows.Next() {
		tokID := token.ID{}
		var typ token.Type
		var quantity string
		var ownerRaw []byte
		err := rows.Scan(
			&tokID.TxId,
			&tokID.Index,
			&ownerRaw,
			&typ,
			&quantity,
		)
		if err != nil {
			return tokens, err
		}
		tok := &token.Token{
			Owner:    ownerRaw,
			Type:     typ,
			Quantity: quantity,
		}

		// put in the right position
		found := false
		for j := 0; j < len(inputs); j++ {
			if inputs[j].Equal(tokID) {
				tokens[j] = tok
				logger.Debugf("set token at location [%s:%s]-[%d]", tok.Type, tok.Quantity, j)
				found = true
				break
			}
		}
		if !found {
			return nil, errors.Errorf("retrieved wrong token [%v]", tokID)
		}

		counter++
	}
	logger.Debugf("found [%d] tokens, expected [%d]", counter, len(inputs))
	if err = rows.Err(); err != nil {
		return tokens, err
	}
	if counter == 0 {
		return nil, errors.Errorf("token not found for key [%s:%d]", inputs[0].TxId, inputs[0].Index)
	}
	if counter != len(inputs) {
		for j, t := range tokens {
			if t == nil {
				return nil, errors.Errorf("token not found for key [%s:%d]", inputs[j].TxId, inputs[j].Index)
			}
		}
		panic("programming error: should not reach this point")
	}
	return tokens, nil
}

// ListHistoryIssuedTokens returns the list of issued tokens
func (db *TokenDB) ListHistoryIssuedTokens() (*token.IssuedTokens, error) {
	query, err := NewSelect("tx_id, idx, owner_raw, token_type, quantity, issuer_raw").From(db.table.Tokens).Where("issuer = true").Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query)
	rows, err := db.readDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer Close(rows)

	var tokens []*token.IssuedToken
	for rows.Next() {
		tok := token.IssuedToken{
			Id: &token.ID{
				TxId:  "",
				Index: 0,
			},
			Owner:    []byte{},
			Type:     "",
			Quantity: "",
			Issuer:   []byte{},
		}
		if err := rows.Scan(&tok.Id.TxId, &tok.Id.Index, &tok.Owner, &tok.Type, &tok.Quantity, &tok.Issuer); err != nil {
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

// GetTokenMetadata retrieves the token metadata for the passed ids.
// For each id, the callback is invoked to unmarshal the token metadata
func (db *TokenDB) GetTokenMetadata(ids []*token.ID) ([][]byte, error) {
	return db.GetAllTokenInfos(ids)
}

// GetTokenOutputsAndMeta retrieves both the token output, metadata, and type for the passed ids.
func (db *TokenDB) GetTokenOutputsAndMeta(ctx context.Context, ids []*token.ID) ([][]byte, [][]byte, []token.Format, error) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent("get_ledger_token_meta")
	tokens, metas, types, err := db.getLedgerTokenAndMeta(ctx, ids)
	if err != nil {
		return nil, nil, nil, err
	}
	span.AddEvent("create_outputs")
	return tokens, metas, types, nil
}

// GetAllTokenInfos retrieves the token information for the passed ids.
func (db *TokenDB) GetAllTokenInfos(ids []*token.ID) ([][]byte, error) {
	if len(ids) == 0 {
		return [][]byte{}, nil
	}
	_, metas, _, err := db.getLedgerTokenAndMeta(context.TODO(), ids)
	return metas, err
}

func (db *TokenDB) getLedgerToken(ids []*token.ID) ([][]byte, error) {
	logger.Debugf("retrieve ledger tokens for [%s]", ids)
	if len(ids) == 0 {
		return [][]byte{}, nil
	}
	where, args := common.Where(db.ci.HasTokens("tx_id", "idx", ids...))

	query, err := NewSelect("tx_id, idx, ledger").From(db.table.Tokens).Where(where).Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query, args)
	rows, err := db.readDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer Close(rows)

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
	logger.Debugf("retrieve ledger tokens for [%s], retrieved [%d] tokens", ids, len(tokenMap))

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

func (db *TokenDB) getLedgerTokenAndMeta(ctx context.Context, ids []*token.ID) ([][]byte, [][]byte, []token.Format, error) {
	span := trace.SpanFromContext(ctx)
	if len(ids) == 0 {
		return nil, nil, nil, nil
	}
	where, args := common.Where(db.ci.HasTokens("tx_id", "idx", ids...))

	query, err := NewSelect("tx_id, idx, ledger, ledger_type, ledger_metadata  ").From(db.table.Tokens).Where(where).Compile()
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to compile query")
	}
	span.AddEvent("query", tracing.WithAttributes(tracing.String(QueryLabel, query)))
	logger.Debug(query, args)
	rows, err := db.readDB.Query(query, args...)
	if err != nil {
		return nil, nil, nil, err
	}
	defer Close(rows)

	span.AddEvent("start_scan_rows")
	infoMap := make(map[string][3][]byte, len(ids))
	for rows.Next() {
		var tok []byte
		var tokType string
		var metadata []byte
		var id token.ID
		if err := rows.Scan(&id.TxId, &id.Index, &tok, &tokType, &metadata); err != nil {
			return nil, nil, nil, err
		}
		infoMap[id.String()] = [3][]byte{tok, metadata, []byte(tokType)}
	}
	if err = rows.Err(); err != nil {
		return nil, nil, nil, err
	}
	span.AddEvent("end_scan_rows", tracing.WithAttributes(tracing.Int(ResultRowsLabel, len(ids))))

	span.AddEvent("combine_results")
	tokens := make([][]byte, len(ids))
	metas := make([][]byte, len(ids))
	types := make([]token.Format, len(ids))
	for i, id := range ids {
		if info, ok := infoMap[id.String()]; !ok {
			return nil, nil, nil, errors.Errorf("token/metadata not found for [%s]", id)
		} else {
			tokens[i] = info[0]
			metas[i] = info[1]
			types[i] = token.Format(info[2])
		}
	}
	return tokens, metas, types, nil
}

// QueryTokenDetails returns details about owned tokens, regardless if they have been spent or not.
// Filters work cumulatively and may be left empty. If a token is owned by two enrollmentIDs and there
// is no filter on enrollmentID, the token will be returned twice (once for each owner).
func (db *TokenDB) QueryTokenDetails(params driver.QueryTokenDetailsParams) ([]driver.TokenDetails, error) {
	where, args := common.Where(db.ci.HasTokenDetails(params, db.table.Tokens))
	join := joinOnTokenID(db.table.Tokens, db.table.Ownership)

	query, err := NewSelect(fmt.Sprintf("%s.tx_id, %s.idx, owner_identity, owner_type, wallet_id, token_type, amount, is_deleted, spent_by, stored_at", db.table.Tokens, db.table.Tokens)).
		From(db.table.Tokens, join).Where(where).Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query, args)
	rows, err := db.readDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer Close(rows)

	var tokenDetails []driver.TokenDetails
	for rows.Next() {
		td := driver.TokenDetails{}
		if err := rows.Scan(
			&td.TxID,
			&td.Index,
			&td.OwnerIdentity,
			&td.OwnerType,
			&td.OwnerEnrollment,
			&td.Type,
			&td.Amount,
			&td.IsSpent,
			&td.SpentBy,
			&td.StoredAt,
		); err != nil {
			return tokenDetails, err
		}
		tokenDetails = append(tokenDetails, td)
	}
	logger.Debugf("found [%d] tokens", len(tokenDetails))
	if err = rows.Err(); err != nil {
		return tokenDetails, err
	}
	return tokenDetails, nil
}

// WhoDeletedTokens returns information about which transaction deleted the passed tokens.
// The bool array is an indicator used to tell if the token at a given position has been deleted or not
func (db *TokenDB) WhoDeletedTokens(inputs ...*token.ID) ([]string, []bool, error) {
	if len(inputs) == 0 {
		return []string{}, []bool{}, nil
	}
	where, args := common.Where(db.ci.HasTokens("tx_id", "idx", inputs...))

	query, err := NewSelect("tx_id, idx, spent_by, is_deleted").From(db.table.Tokens).Where(where).Compile()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query, args)
	rows, err := db.readDB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer Close(rows)

	spentBy := make([]string, len(inputs))
	isSpent := make([]bool, len(inputs))
	found := make([]bool, len(inputs))

	counter := 0
	for rows.Next() {
		var txID string
		var idx uint64
		var spBy string
		var isSp bool
		if err := rows.Scan(&txID, &idx, &spBy, &isSp); err != nil {
			return spentBy, isSpent, err
		}
		// order is not necessarily the same, so we have to set it in a loop
		for i, inp := range inputs {
			if inp.TxId == txID && inp.Index == idx {
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

func (db *TokenDB) TransactionExists(ctx context.Context, id string) (bool, error) {
	span := trace.SpanFromContext(ctx)
	query, err := NewSelect("tx_id").From(db.table.Tokens).Where("tx_id=$1 LIMIT 1").Compile()
	if err != nil {
		return false, errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query, id)

	span.AddEvent("query", trace.WithAttributes(tracing.String(QueryLabel, query)))
	row := db.readDB.QueryRow(query, id)
	var found string
	span.AddEvent("scan_rows")
	if err := row.Scan(&found); err != nil {
		if errors2.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		logger.Warnf("tried to check transaction existence for id %s, err %s", id, err)
		return false, err
	}
	return true, nil
}

func (db *TokenDB) StorePublicParams(raw []byte) error {
	rawHash := hash.Hashable(raw).Raw()
	_, err := db.PublicParamsByHash(rawHash)
	if err == nil {
		logger.Debugf("public params [%s] already in the database", base64.StdEncoding.EncodeToString(rawHash))
		// no need to update the public parameters
		return nil
	}

	now := time.Now().UTC()
	query, err := NewInsertInto(db.table.PublicParams).Rows("raw, raw_hash, stored_at").Compile()
	if err != nil {
		return errors.Wrapf(err, "failed to compile query")
	}
	logger.Debugf(query, fmt.Sprintf("store public parameters (%d bytes) [%v], hash [%s]", len(raw), now, base64.StdEncoding.EncodeToString(rawHash)))
	_, err = db.writeDB.Exec(query, raw, rawHash, now)
	return err
}

func (db *TokenDB) PublicParams() ([]byte, error) {
	var params []byte
	query, err := NewSelect("raw").From(db.table.PublicParams).OrderBy("stored_at DESC LIMIT 1").Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query)

	row := db.readDB.QueryRow(query)
	err = row.Scan(&params)
	if err != nil {
		if errors.HasCause(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "error querying db")
	}
	return params, nil
}

func (db *TokenDB) PublicParamsByHash(rawHash tdriver.PPHash) ([]byte, error) {
	var params []byte
	query, err := NewSelect("raw").From(db.table.PublicParams).Where("raw_hash = $1").Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query)

	row := db.readDB.QueryRow(query, rawHash)
	err = row.Scan(&params)
	if err != nil {
		return nil, errors.Wrapf(err, "error querying db")
	}
	return params, nil
}

// TODO Convert to multi-row query
func (db *TokenDB) StoreCertifications(certifications map[*token.ID][]byte) (err error) {
	now := time.Now().UTC()
	query, err := NewInsertInto(db.table.Certifications).Rows("tx_id, idx, certification, stored_at").Compile()
	if err != nil {
		return errors.Wrapf(err, "failed to compile query")
	}

	tx, err := db.writeDB.Begin()
	if err != nil {
		return errors.Errorf("failed starting a transaction")
	}
	defer func() {
		if err != nil && tx != nil {
			if err := tx.Rollback(); err != nil {
				logger.Errorf("failed to rollback [%s][%s]", err, debug.Stack())
			}
		}
	}()

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
	where, args := common.Where(db.ci.HasTokens("tx_id", "idx", tokenID))

	query, err := NewSelect("certification").From(db.table.Certifications).Where(where).Compile()
	if err != nil {
		return false
	}
	logger.Debug(query, args)
	row := db.readDB.QueryRow(query, args...)

	var certification []byte
	if err := row.Scan(&certification); err != nil {
		if errors.HasCause(err, sql.ErrNoRows) {
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
	where, args := common.Where(db.ci.HasTokens("tx_id", "idx", ids...))

	query, err := NewSelect("tx_id, idx, certification").From(db.table.Certifications).Where(where).Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile query")
	}

	rows, err := db.readDB.Query(query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query")
	}
	defer Close(rows)

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
			owner_identity BYTEA NOT NULL,
			owner_wallet_id TEXT, 
			ledger BYTEA NOT NULL,
            ledger_type TEXT DEFAULT '',
			ledger_metadata BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL,
			is_deleted BOOL NOT NULL DEFAULT false,
			spent_by TEXT NOT NULL DEFAULT '',
			spent_at TIMESTAMP,
			owner BOOL NOT NULL DEFAULT false,
			auditor BOOL NOT NULL DEFAULT false,
			issuer BOOL NOT NULL DEFAULT false,
			spendable BOOL NOT NULL DEFAULT true,
			PRIMARY KEY (tx_id, idx)
		);
		CREATE INDEX IF NOT EXISTS idx_spent_%s ON %s ( is_deleted, owner );
		CREATE INDEX IF NOT EXISTS idx_tx_id_%s ON %s ( tx_id );

		-- Ownership
		CREATE TABLE IF NOT EXISTS %s (
			tx_id TEXT NOT NULL,
			idx INT NOT NULL,
			wallet_id TEXT NOT NULL,
			PRIMARY KEY (tx_id, idx, wallet_id),
			FOREIGN KEY (tx_id, idx) REFERENCES %s
		);

		-- Public Parameters
		CREATE TABLE IF NOT EXISTS %s (
			raw_hash BYTEA PRIMARY KEY,
			raw BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL 
		);
		CREATE INDEX IF NOT EXISTS stored_at_%s ON %s ( stored_at );

		-- Certifications
		CREATE TABLE IF NOT EXISTS %s (
			tx_id TEXT NOT NULL,
			idx INT NOT NULL,
			certification BYTEA NOT NULL,
			stored_at TIMESTAMP NOT NULL,
			PRIMARY KEY (tx_id, idx),
			FOREIGN KEY (tx_id, idx) REFERENCES %s
		);
		`,
		db.table.Tokens,
		db.table.Tokens, db.table.Tokens,
		db.table.Tokens, db.table.Tokens,
		db.table.Ownership, db.table.Tokens,
		db.table.PublicParams, db.table.PublicParams, db.table.PublicParams,
		db.table.Certifications, db.table.Tokens,
	)
}

func (db *TokenDB) Close() {
	if db.readDB != db.writeDB {
		Close(db.writeDB)
	}
	Close(db.readDB)
}

func (db *TokenDB) NewTokenDBTransaction() (driver.TokenDBTransaction, error) {
	tx, err := db.writeDB.Begin()
	if err != nil {
		return nil, errors.Errorf("failed starting a db transaction")
	}
	return &TokenTransaction{ci: db.ci, table: &db.table, tx: tx}, nil
}

func (db *TokenDB) SetSupportedTokenFormats(formats []token.Format) error {
	db.sttMutex.Lock()
	db.supportedTokenFormats = formats
	db.sttMutex.Unlock()
	return nil
}

func (db *TokenDB) getSupportedTokenFormats() []token.Format {
	db.sttMutex.RLock()
	supportedTokenTypes := db.supportedTokenFormats
	db.sttMutex.RUnlock()
	return supportedTokenTypes
}

func (db *TokenDB) unspendableTokenFormats(ctx context.Context, walletID string, tokenType token.Type) ([]token.Format, error) {
	span := trace.SpanFromContext(ctx)
	where, args := common.Where(db.ci.HasTokenDetails(driver.QueryTokenDetailsParams{
		WalletID:  walletID,
		TokenType: tokenType,
		Spendable: driver.Any,
	}, ""))
	query, err := NewSelectDistinct("ledger_type").From(db.table.Tokens).Where(where).Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile query")
	}
	logger.Debug(query, args)
	span.AddEvent("start_query", tracing.WithAttributes(tracing.String(QueryLabel, query)))
	rows, err := db.readDB.Query(query, args...)
	span.AddEvent("end_query")
	if err != nil {
		return nil, errors.Wrapf(err, "error querying db")
	}
	defer Close(rows)
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "error querying db")
	}
	// read the types from the query result and remove discard those in db.getSupportedTokenFormats()
	supportedFormats := db.getSupportedTokenFormats()
	logger.Debugf("supported token formats are [%v]", supportedFormats)
	includeFormats := make([]token.Format, 0)
	for rows.Next() {
		var tmp string
		if err := rows.Scan(&tmp); err != nil {
			return nil, errors.Wrapf(err, "failed to scan row")
		}
		format := token.Format(tmp)

		logger.Debugf("format from db [%s]", format)

		// include format only if it is not in supportedFormats
		if !slices.Contains(supportedFormats, format) {
			includeFormats = append(includeFormats, format)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "error querying db")
	}
	logger.Debugf("after filtering we have [%v]", includeFormats)

	return includeFormats, nil
}

type TokenTransaction struct {
	table *tokenTables
	ci    TokenInterpreter
	tx    *sql.Tx
}

func (t *TokenTransaction) GetToken(ctx context.Context, tokenID token.ID, includeDeleted bool) (*token.Token, []string, error) {
	span := trace.SpanFromContext(ctx)
	where, args := common.Where(t.ci.HasTokenDetails(driver.QueryTokenDetailsParams{
		IDs:            []*token.ID{&tokenID},
		IncludeDeleted: includeDeleted,
	}, t.table.Tokens))
	join := joinOnTokenID(t.table.Tokens, t.table.Ownership)

	query, err := NewSelect(
		fmt.Sprintf("owner_raw, token_type, quantity, %s.wallet_id, owner_wallet_id", t.table.Ownership),
	).From(t.table.Tokens, join).Where(where).Compile()
	if err != nil {
		return nil, nil, errors.Errorf("failed building query")
	}
	span.AddEvent("query", tracing.WithAttributes(tracing.String(QueryLabel, query)))
	logger.Debug(query, args)
	rows, err := t.tx.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer Close(rows)

	span.AddEvent("start_scan_rows")
	var raw []byte
	var tokenType token.Type
	var quantity string
	var owners []string
	var walletID *string
	for rows.Next() {
		var tempOwner *string
		if err := rows.Scan(&raw, &tokenType, &quantity, &tempOwner, &walletID); err != nil {
			return nil, owners, err
		}
		var owner string
		if tempOwner != nil {
			owner = *tempOwner
		}
		if len(owner) > 0 {
			owners = append(owners, owner)
		}
	}
	if rows.Err() != nil {
		return nil, nil, rows.Err()
	}
	if walletID != nil && len(*walletID) != 0 {
		owners = append(owners, *walletID)
	}
	span.AddEvent("end_scan_rows", tracing.WithAttributes(tracing.Int(ResultRowsLabel, len(owners))))
	if len(raw) == 0 {
		return nil, owners, nil
	}
	return &token.Token{
		Owner:    raw,
		Type:     tokenType,
		Quantity: quantity,
	}, owners, nil
}

func (t *TokenTransaction) Delete(ctx context.Context, tokenID token.ID, deletedBy string) error {
	span := trace.SpanFromContext(ctx)
	// logger.Debugf("delete token [%s:%d:%s]", txID, index, deletedBy)
	// We don't delete audit tokens, and we keep the 'ownership' relation.
	now := time.Now().UTC()
	query, err := NewUpdate(t.table.Tokens).Set("is_deleted, spent_by, spent_at").Where("tx_id, idx").Compile()
	if err != nil {
		return errors.Wrapf(err, "failed building query")
	}
	logger.Debugf(query, true, deletedBy, now, tokenID.TxId, tokenID.Index)
	span.AddEvent("query", tracing.WithAttributes(tracing.String(QueryLabel, query)))
	if _, err := t.tx.Exec(query, true, deletedBy, now, tokenID.TxId, tokenID.Index); err != nil {
		span.RecordError(err)
		return errors.Wrapf(err, "error setting token to deleted [%s]", tokenID.TxId)
	}
	span.AddEvent("end_query")
	return nil
}

func (t *TokenTransaction) StoreToken(ctx context.Context, tr driver.TokenRecord, owners []string) error {
	if len(tr.OwnerWalletID) == 0 && len(owners) == 0 && tr.Owner {
		return errors.Errorf("no owners specified [%s]", string(debug.Stack()))
	}

	span := trace.SpanFromContext(ctx)
	// logger.Debugf("store record [%s:%d,%v] in table [%s]", tr.TxID, tr.Index, owners, t.db.table.Tokens)

	// Store token
	now := time.Now().UTC()
	query, err := NewInsertInto(t.table.Tokens).Rows(
		"tx_id, idx, issuer_raw, owner_raw, owner_type, owner_identity, owner_wallet_id, ledger, ledger_type, ledger_metadata, token_type, quantity, amount, stored_at, owner, auditor, issuer").Compile()
	if err != nil {
		return errors.Wrapf(err, "failed building insert")
	}
	logger.Debug(query,
		tr.TxID,
		tr.Index,
		len(tr.IssuerRaw),
		len(tr.OwnerRaw),
		tr.OwnerType,
		len(tr.OwnerIdentity),
		tr.OwnerWalletID,
		len(tr.Ledger),
		tr.LedgerFormat,
		len(tr.LedgerMetadata),
		tr.Type,
		tr.Quantity,
		tr.Amount,
		now,
		tr.Owner,
		tr.Auditor,
		tr.Issuer)
	span.AddEvent("query", tracing.WithAttributes(tracing.String(QueryLabel, query)))
	if _, err := t.tx.Exec(query,
		tr.TxID,
		tr.Index,
		tr.IssuerRaw,
		tr.OwnerRaw,
		tr.OwnerType,
		tr.OwnerIdentity,
		tr.OwnerWalletID,
		tr.Ledger,
		tr.LedgerFormat,
		tr.LedgerMetadata,
		tr.Type,
		tr.Quantity,
		tr.Amount,
		now,
		tr.Owner,
		tr.Auditor,
		tr.Issuer); err != nil {
		logger.Errorf("error storing token [%s] in table [%s]: [%s][%s]", tr.TxID, t.table.Tokens, err, string(debug.Stack()))
		return errors.Wrapf(err, "error storing token [%s] in table [%s]", tr.TxID, t.table.Tokens)
	}

	// Store ownership
	span.AddEvent("store_ownerships")
	for _, eid := range owners {
		query, err := NewInsertInto(t.table.Ownership).Rows("tx_id, idx, wallet_id").Compile()
		if err != nil {
			return errors.Wrapf(err, "failed building insert")
		}
		logger.Debug(query, tr.TxID, tr.Index, eid)
		span.AddEvent("query", tracing.WithAttributes(tracing.String(QueryLabel, query)))
		if _, err := t.tx.Exec(query, tr.TxID, tr.Index, eid); err != nil {
			return errors.Wrapf(err, "error storing token ownership [%s]", tr.TxID)
		}
	}

	return nil
}

func (t *TokenTransaction) SetSpendable(ctx context.Context, tokenID token.ID, spendable bool) error {
	span := trace.SpanFromContext(ctx)
	query := fmt.Sprintf("UPDATE %s SET spendable = $1 WHERE tx_id = $2 AND idx = $3;", t.table.Tokens)
	logger.Infof(query, spendable, tokenID.TxId, tokenID.Index)
	span.AddEvent("query", tracing.WithAttributes(tracing.String(QueryLabel, query)))
	if _, err := t.tx.Exec(query, spendable, tokenID.TxId, tokenID.Index); err != nil {
		span.RecordError(err)
		return errors.Wrapf(err, "error setting spendable flag to [%v] for [%s]", spendable, tokenID.TxId)
	}
	span.AddEvent("end_query")
	return nil

}

func (t *TokenTransaction) SetSpendableBySupportedTokenFormats(ctx context.Context, formats []token.Format) error {
	span := trace.SpanFromContext(ctx)

	// first set all spendable flags to false
	query := fmt.Sprintf("UPDATE %s SET spendable = $1;", t.table.Tokens)
	logger.Infof(query, false)
	span.AddEvent("query", tracing.WithAttributes(tracing.String(QueryLabel, query)))
	if _, err := t.tx.Exec(query, false); err != nil {
		span.RecordError(err)
		return errors.Wrapf(err, "error setting spendable flag to false for all tokens")
	}
	span.AddEvent("end_query")

	// then set the spendable flags to true only for the supported token types

	cond := t.ci.HasTokenFormats("ledger_type", formats...)
	args := append([]any{true}, cond.Params()...)
	offset := 2
	where := cond.ToString(&offset)

	query = fmt.Sprintf("UPDATE %s SET spendable = $1 WHERE %s;", t.table.Tokens, where)
	logger.Infof(query, args)
	span.AddEvent("query", tracing.WithAttributes(tracing.String(QueryLabel, query)))
	res, err := t.tx.Exec(query, args...)
	if err != nil {
		span.RecordError(err)
		return errors.Wrapf(err, "error setting spendable flag to true for token types [%v]", formats)
	} else {
		rows, _ := res.RowsAffected()
		logger.Infof("row affected [%d]", rows)
	}
	span.AddEvent("end_query")

	return nil
}

func (t *TokenTransaction) Commit() error {
	return t.tx.Commit()
}

func (t *TokenTransaction) Rollback() error {
	return t.tx.Rollback()
}

type UnspentTokensInWalletIterator struct {
	txs *sql.Rows
}

func (u *UnspentTokensInWalletIterator) Close() {
	Close(u.txs)
}

func (u *UnspentTokensInWalletIterator) Next() (*token.UnspentTokenInWallet, error) {
	if !u.txs.Next() {
		return nil, nil
	}

	tok := &token.UnspentTokenInWallet{
		Id:       &token.ID{},
		WalletID: "",
		Type:     "",
		Quantity: "",
	}
	if err := u.txs.Scan(&tok.Id.TxId, &tok.Id.Index, &tok.Type, &tok.Quantity, &tok.WalletID); err != nil {
		return nil, err
	}
	return tok, nil
}

type UnspentTokensIterator struct {
	txs *sql.Rows
}

func (u *UnspentTokensIterator) Close() {
	Close(u.txs)
}

func (u *UnspentTokensIterator) Next() (*token.UnspentToken, error) {
	if !u.txs.Next() {
		return nil, nil
	}

	var typ token.Type
	var quantity string
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
		Id:       &id,
		Owner:    owner,
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

type UnspendableTokensInWalletIterator struct {
	txs *sql.Rows
}

func (u *UnspendableTokensInWalletIterator) Close() {
	Close(u.txs)
}

func (u *UnspendableTokensInWalletIterator) Next() (*token.LedgerToken, error) {
	if !u.txs.Next() {
		return nil, nil
	}

	tok := &token.LedgerToken{}
	if err := u.txs.Scan(&tok.ID.TxId, &tok.ID.Index, &tok.Token, &tok.TokenMetadata, &tok.Format); err != nil {
		return nil, err
	}
	return tok, nil
}
