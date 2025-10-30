/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"database/sql"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/common"
	q "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query"
	common3 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/cond"
	tdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type tokenTables struct {
	Tokens         string
	Ownership      string
	PublicParams   string
	Certifications string
}

func NewTokenStore(readDB, writeDB *sql.DB, tables TableNames, ci common3.CondInterpreter) (*TokenStore, error) {
	return newTokenStore(readDB, writeDB, tokenTables{
		Tokens:         tables.Tokens,
		Ownership:      tables.Ownership,
		PublicParams:   tables.PublicParams,
		Certifications: tables.Certifications,
	}, ci), nil
}

func (db *TokenStore) CreateSchema() error {
	return common.InitSchema(db.writeDB, db.GetSchema())
}

type TokenStore struct {
	readDB  *sql.DB
	writeDB *sql.DB
	table   tokenTables
	ci      common3.CondInterpreter

	sttMutex              sync.RWMutex
	supportedTokenFormats []token.Format
}

func newTokenStore(readDB, writeDB *sql.DB, tables tokenTables, ci common3.CondInterpreter) *TokenStore {
	return &TokenStore{
		readDB:  readDB,
		writeDB: writeDB,
		table:   tables,
		ci:      ci,
	}
}

func (db *TokenStore) StoreToken(ctx context.Context, tr driver.TokenRecord, owners []string) (err error) {
	tx, err := db.NewTokenDBTransaction()
	if err != nil {
		return
	}
	if err = tx.StoreToken(ctx, tr, owners); err != nil {
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
func (db *TokenStore) DeleteTokens(ctx context.Context, deletedBy string, ids ...*token.ID) error {
	logger.DebugfContext(ctx, "delete tokens [%s][%v]", deletedBy, ids)
	if len(ids) == 0 {
		return nil
	}

	query, args := q.Update(db.table.Tokens).
		Set("is_deleted", true).
		Set("spent_by", deletedBy).
		Set("spent_at", time.Now().UTC()).
		Where(HasTokens("tx_id", "idx", ids...)).
		Format(db.ci)
	logger.Debug(query, args)
	if _, err := db.writeDB.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "error setting tokens to deleted [%v]", ids)
	}
	return nil
}

// IsMine just checks if the token is in the local storage and not deleted
func (db *TokenStore) IsMine(ctx context.Context, txID string, index uint64) (bool, error) {
	query, args := q.Select().
		FieldsByName("tx_id").
		From(q.Table(db.table.Tokens)).
		Where(cond.And(cond.Eq("tx_id", txID), cond.Eq("idx", index), cond.Eq("is_deleted", false), cond.Eq("owner", true))).
		Limit(1).
		Format(db.ci)

	id, err := common.QueryUnique[string](db.readDB, query, args...)

	logger.DebugfContext(ctx, "token [%s:%d] is mine [%s]", txID, index, id)
	return id == txID, err
}

// UnspentTokensIterator returns an iterator over all unspent tokens
func (db *TokenStore) UnspentTokensIterator(ctx context.Context) (tdriver.UnspentTokensIterator, error) {
	return db.UnspentTokensIteratorBy(ctx, "", "")
}

// UnspentTokensIteratorBy returns an iterator of unspent tokens owned by the passed id and whose type is the passed on.
// The token type can be empty. In that case, tokens of any type are returned.
func (db *TokenStore) UnspentTokensIteratorBy(ctx context.Context, walletID string, tokenType token.Type) (tdriver.UnspentTokensIterator, error) {
	tokenTable, ownershipTable := q.Table(db.table.Tokens), q.Table(db.table.Ownership)
	query, args := q.Select().
		Fields(
			tokenTable.Field("tx_id"), tokenTable.Field("idx"), common3.FieldName("owner_raw"),
			common3.FieldName("token_type"), common3.FieldName("quantity"),
		).
		From(tokenTable.Join(ownershipTable, cond.And(
			cond.Cmp(tokenTable.Field("tx_id"), "=", ownershipTable.Field("tx_id")),
			cond.Cmp(tokenTable.Field("idx"), "=", ownershipTable.Field("idx"))),
		)).
		Where(HasTokenDetails(driver.QueryTokenDetailsParams{
			WalletID:  walletID,
			TokenType: tokenType,
		}, tokenTable)).
		Format(db.ci)

	logger.Debug(query, args)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	return common.NewIterator(rows, func(r *token.UnspentToken) error {
		return rows.Scan(&r.Id.TxId, &r.Id.Index, &r.Owner, &r.Type, &r.Quantity)
	}), nil
}

// SpendableTokensIteratorBy returns the minimum information about the tokens needed for the selector
func (db *TokenStore) SpendableTokensIteratorBy(ctx context.Context, walletID string, typ token.Type) (tdriver.SpendableTokensIterator, error) {
	query, args := q.Select().
		FieldsByName("tx_id", "idx", "token_type", "quantity", "owner_wallet_id").
		From(q.Table(db.table.Tokens)).
		Where(HasTokenDetails(driver.QueryTokenDetailsParams{
			WalletID:           walletID,
			TokenType:          typ,
			Spendable:          driver.SpendableOnly,
			LedgerTokenFormats: db.getSupportedTokenFormats(),
		}, nil)).
		Format(db.ci)

	logger.Debug(query, args)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "error querying db")
	}

	return common.NewIterator(rows, func(r *token.UnspentTokenInWallet) error {
		return rows.Scan(&r.Id.TxId, &r.Id.Index, &r.Type, &r.Quantity, &r.WalletID)
	}), nil
}

// UnspentLedgerTokensIteratorBy returns an iterator over all unspent ledger tokens
func (db *TokenStore) UnspentLedgerTokensIteratorBy(ctx context.Context) (tdriver.LedgerTokensIterator, error) {
	return db.queryLedgerTokens(ctx, driver.QueryTokenDetailsParams{Spendable: driver.Any})
}

// UnsupportedTokensIteratorBy returns the minimum information for upgrade about the tokens that are not supported
func (db *TokenStore) UnsupportedTokensIteratorBy(ctx context.Context, walletID string, tokenType token.Type) (tdriver.UnsupportedTokensIterator, error) {
	// first select all the distinct ledger types
	includeFormats, err := db.unspendableTokenFormats(ctx, walletID, tokenType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get unspendable token formats")
	}
	logger.DebugfContext(ctx, "after filtering we have [%v]", includeFormats)

	// now, select the tokens with the list of ledger tokens
	return db.queryLedgerTokens(ctx, driver.QueryTokenDetailsParams{
		WalletID:           walletID,
		TokenType:          tokenType,
		Spendable:          driver.Any,
		LedgerTokenFormats: includeFormats,
	})
}

func (db *TokenStore) queryLedgerTokens(ctx context.Context, details driver.QueryTokenDetailsParams) (tdriver.UnsupportedTokensIterator, error) {
	query, args := q.Select().FieldsByName("tx_id", "idx", "ledger", "ledger_metadata", "ledger_type").
		From(q.Table(db.table.Tokens)).
		Where(HasTokenDetails(details, nil)).
		Format(db.ci)

	logger.Debug(query, args)

	rows, err := db.readDB.QueryContext(ctx, query, args...)

	if err != nil {
		return nil, errors.Wrapf(err, "error querying db")
	}
	return common.NewIterator(rows, func(tok *token.LedgerToken) error {
		return rows.Scan(&tok.ID.TxId, &tok.ID.Index, &tok.Token, &tok.TokenMetadata, &tok.Format)
	}), nil
}

// Balance returns the sun of the amounts, with 64 bits of precision, of the tokens with type and EID equal to those passed as arguments.
func (db *TokenStore) Balance(ctx context.Context, walletID string, typ token.Type) (uint64, error) {
	return db.balance(ctx, driver.QueryTokenDetailsParams{
		WalletID:  walletID,
		TokenType: typ,
	})
}

func (db *TokenStore) balance(ctx context.Context, opts driver.QueryTokenDetailsParams) (uint64, error) {
	tokenTable, ownershipTable := q.Table(db.table.Tokens), q.Table(db.table.Ownership)
	query, args := q.Select().FieldsByName("SUM(amount)").
		From(tokenTable.Join(ownershipTable, cond.And(
			cond.Cmp(tokenTable.Field("tx_id"), "=", ownershipTable.Field("tx_id")),
			cond.Cmp(tokenTable.Field("idx"), "=", ownershipTable.Field("idx"))),
		)).
		Where(HasTokenDetails(opts, tokenTable)).
		Format(db.ci)

	sum, err := common.QueryUnique[*uint64](db.readDB, query, args...)
	if err != nil || sum == nil {
		return 0, err
	}
	return *sum, nil
}

// ListUnspentTokensBy returns the list of unspent tokens, filtered by owner and token type
func (db *TokenStore) ListUnspentTokensBy(ctx context.Context, walletID string, typ token.Type) (*token.UnspentTokens, error) {
	logger.DebugfContext(ctx, "list unspent token by [%s,%s]", walletID, typ)
	it, err := db.UnspentTokensIteratorBy(ctx, walletID, typ)
	if err != nil {
		return nil, err
	}
	tokens, err := iterators.ReadAllPointers(it)
	if err != nil {
		return nil, err
	}
	return &token.UnspentTokens{Tokens: tokens}, nil
}

// ListUnspentTokens returns the list of unspent tokens
func (db *TokenStore) ListUnspentTokens(ctx context.Context) (*token.UnspentTokens, error) {
	logger.DebugfContext(ctx, "list unspent tokens...")
	it, err := db.UnspentTokensIterator(ctx)
	if err != nil {
		return nil, err
	}
	tokens, err := iterators.ReadAllPointers(it)
	if err != nil {
		return nil, err
	}
	return &token.UnspentTokens{Tokens: tokens}, nil
}

// ListAuditTokens returns the audited tokens associated to the passed ids
func (db *TokenStore) ListAuditTokens(ctx context.Context, ids ...*token.ID) ([]*token.Token, error) {
	if len(ids) == 0 {
		return []*token.Token{}, nil
	}

	query, args := q.Select().
		FieldsByName("tx_id", "idx", "owner_raw", "token_type", "quantity").
		From(q.Table(db.table.Tokens)).
		Where(cond.And(
			HasTokens("tx_id", "idx", ids...),
			cond.Eq("auditor", true)),
		).
		Format(db.ci)

	logger.Debug(query, args)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer Close(rows)

	tokens := make([]*token.Token, len(ids))
	counter := 0
	for rows.Next() {
		id := token.ID{}
		tok := token.Token{
			Owner:    []byte{},
			Type:     "",
			Quantity: "",
		}
		if err := rows.Scan(&id.TxId, &id.Index, &tok.Owner, &tok.Type, &tok.Quantity); err != nil {
			return tokens, err
		}

		// the result is expected to be in order of the ids
		found := false
		for i := range ids {
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
func (db *TokenStore) ListHistoryIssuedTokens(ctx context.Context) (*token.IssuedTokens, error) {
	query, args := q.Select().
		FieldsByName("tx_id", "idx", "owner_raw", "token_type", "quantity", "issuer_raw").
		From(q.Table(db.table.Tokens)).
		Where(cond.Eq("issuer", true)).
		Format(db.ci)

	logger.Debug(query)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	it := common.NewIterator(rows, func(tok *token.IssuedToken) error {
		return rows.Scan(&tok.Id.TxId, &tok.Id.Index, &tok.Owner, &tok.Type, &tok.Quantity, &tok.Issuer)
	})
	tokens, err := iterators.ReadAllPointers(it)
	if err != nil {
		return nil, err
	}
	return &token.IssuedTokens{Tokens: tokens}, rows.Err()
}

func (db *TokenStore) GetTokenOutputs(ctx context.Context, ids []*token.ID, callback tdriver.QueryCallbackFunc) error {
	tokens, err := db.getLedgerToken(ctx, ids)
	if err != nil {
		return err
	}
	for i := range ids {
		if err := callback(ids[i], tokens[i]); err != nil {
			return err
		}
	}
	return nil
}

// GetTokenMetadata retrieves the token metadata for the passed ids.
// For each id, the callback is invoked to unmarshal the token metadata
func (db *TokenStore) GetTokenMetadata(ctx context.Context, ids []*token.ID) ([][]byte, error) {
	return db.GetAllTokenInfos(ctx, ids)
}

// GetTokenOutputsAndMeta retrieves both the token output, metadata, and type for the passed ids.
func (db *TokenStore) GetTokenOutputsAndMeta(ctx context.Context, ids []*token.ID) ([][]byte, [][]byte, []token.Format, error) {
	tokens, metas, types, err := db.getLedgerTokenAndMeta(ctx, ids)
	if err != nil {
		return nil, nil, nil, err
	}
	return tokens, metas, types, nil
}

// GetAllTokenInfos retrieves the token information for the passed ids.
func (db *TokenStore) GetAllTokenInfos(ctx context.Context, ids []*token.ID) ([][]byte, error) {
	if len(ids) == 0 {
		return [][]byte{}, nil
	}
	_, metas, _, err := db.getLedgerTokenAndMeta(ctx, ids)
	return metas, err
}

func (db *TokenStore) getLedgerToken(ctx context.Context, ids []*token.ID) ([][]byte, error) {
	logger.DebugfContext(ctx, "retrieve ledger tokens for [%s]", ids)
	if len(ids) == 0 {
		return [][]byte{}, nil
	}

	query, args := q.Select().
		FieldsByName("tx_id, idx, ledger").
		From(q.Table(db.table.Tokens)).
		Where(HasTokens("tx_id", "idx", ids...)).
		Format(db.ci)

	logger.Debug(query, args)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
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
	logger.DebugfContext(ctx, "retrieve ledger tokens for [%s], retrieved [%d] tokens", ids, len(tokenMap))

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

func (db *TokenStore) getLedgerTokenAndMeta(ctx context.Context, ids []*token.ID) ([][]byte, [][]byte, []token.Format, error) {
	if len(ids) == 0 {
		return nil, nil, nil, nil
	}

	query, args := q.Select().
		FieldsByName("tx_id", "idx", "ledger", "ledger_type", "ledger_metadata").
		From(q.Table(db.table.Tokens)).
		Where(HasTokens("tx_id", "idx", ids...)).
		Format(db.ci)

	logger.Debug(query, args)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, nil, err
	}
	defer Close(rows)

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

// GetTokens returns the owned tokens and their identifier keys for the passed ids.
func (db *TokenStore) GetTokens(ctx context.Context, inputs ...*token.ID) ([]*token.Token, error) {
	if len(inputs) == 0 {
		return []*token.Token{}, nil
	}

	query, args := q.Select().
		FieldsByName("tx_id", "idx", "owner_raw", "token_type", "quantity").
		From(q.Table(db.table.Tokens)).
		Where(cond.And(
			HasTokens("tx_id", "idx", inputs...),
			cond.Eq("is_deleted", false),
			cond.Eq("owner", true),
		)).
		Format(db.ci)

	logger.Debug(query, args)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	it := common.NewIterator(rows, func(tok *token.UnspentToken) error {
		return rows.Scan(&tok.Id.TxId, &tok.Id.Index, &tok.Owner, &tok.Type, &tok.Quantity)
	})
	counter := 0
	tokens := make([]*token.Token, len(inputs))
	err = iterators.ForEach(it, func(tok *token.UnspentToken) error {
		// put in the right position
		found := false
		for j := range inputs {
			if inputs[j].Equal(tok.Id) {
				tokens[j] = &token.Token{
					Owner:    tok.Owner,
					Type:     tok.Type,
					Quantity: tok.Quantity,
				}
				logger.DebugfContext(ctx, "set token at location [%s:%s]-[%d]", tok.Type, tok.Quantity, j)
				found = true
				break
			}
		}
		if !found {
			return errors.Errorf("retrieved wrong token [%v]", tok.Id)
		}
		counter++
		return nil
	})
	if err != nil {
		return nil, err
	}

	logger.DebugfContext(ctx, "found [%d] tokens, expected [%d]", counter, len(inputs))
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

// QueryTokenDetails returns details about owned tokens, regardless if they have been spent or not.
// Filters work cumulatively and may be left empty. If a token is owned by two enrollmentIDs and there
// is no filter on enrollmentID, the token will be returned twice (once for each owner).
func (db *TokenStore) QueryTokenDetails(ctx context.Context, params driver.QueryTokenDetailsParams) ([]driver.TokenDetails, error) {
	tokenTable, ownershipTable := q.Table(db.table.Tokens), q.Table(db.table.Ownership)
	query, args := q.Select().
		Fields(
			tokenTable.Field("tx_id"), tokenTable.Field("idx"), common3.FieldName("owner_identity"),
			common3.FieldName("owner_type"), common3.FieldName("wallet_id"), common3.FieldName("token_type"),
			common3.FieldName("amount"), common3.FieldName("is_deleted"), common3.FieldName("spent_by"),
			common3.FieldName("stored_at"),
		).
		From(tokenTable.Join(ownershipTable, cond.And(
			cond.Cmp(tokenTable.Field("tx_id"), "=", ownershipTable.Field("tx_id")),
			cond.Cmp(tokenTable.Field("idx"), "=", ownershipTable.Field("idx"))),
		)).
		Where(HasTokenDetails(params, tokenTable)).
		Format(db.ci)

	logger.Debug(query, args)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	it := common.NewIterator(rows, func(td *driver.TokenDetails) error {
		return rows.Scan(&td.TxID, &td.Index, &td.OwnerIdentity, &td.OwnerType, &td.OwnerEnrollment, &td.Type, &td.Amount, &td.IsSpent, &td.SpentBy, &td.StoredAt)
	})
	return iterators.ReadAllValues(it)
}

// WhoDeletedTokens returns information about which transaction deleted the passed tokens.
// The bool array is an indicator used to tell if the token at a given position has been deleted or not
func (db *TokenStore) WhoDeletedTokens(ctx context.Context, inputs ...*token.ID) ([]string, []bool, error) {
	if len(inputs) == 0 {
		return []string{}, []bool{}, nil
	}

	query, args := q.Select().
		FieldsByName("tx_id", "idx", "spent_by", "is_deleted").
		From(q.Table(db.table.Tokens)).
		Where(HasTokens("tx_id", "idx", inputs...)).
		Format(db.ci)

	logger.Debug(query, args)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
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
	logger.DebugfContext(ctx, "found [%d] records, expected [%d]", counter, len(inputs))
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

func (db *TokenStore) TransactionExists(ctx context.Context, id string) (bool, error) {
	query, args := q.Select().
		FieldsByName("tx_id").
		From(q.Table(db.table.Tokens)).
		Where(cond.Eq("tx_id", id)).
		Limit(1).
		Format(db.ci)

	txID, err := common.QueryUniqueContext[string](ctx, db.readDB, query, args...)

	return len(txID) > 0, err
}

func (db *TokenStore) StorePublicParams(ctx context.Context, raw []byte) error {
	rawHash := utils.Hashable(raw).Raw()

	if pps, err := db.PublicParamsByHash(ctx, rawHash); err == nil && len(pps) > 0 {
		logger.DebugfContext(ctx, "public params [%s] already in the database", logging.Base64(rawHash))
		// no need to update the public parameters
		return nil
	}

	query, args := q.InsertInto(db.table.PublicParams).
		Fields("raw", "raw_hash", "stored_at").
		Row(raw, rawHash, time.Now().UTC()).
		Format()
	logger.DebugfContext(ctx, query, fmt.Sprintf("store public parameters (%d bytes), hash [%s]", len(raw), logging.Base64(rawHash)))
	_, err := db.writeDB.ExecContext(ctx, query, args...)
	return err
}

func (db *TokenStore) PublicParams(ctx context.Context) ([]byte, error) {
	query, args := q.Select().
		FieldsByName("raw").
		From(q.Table(db.table.PublicParams)).
		OrderBy(q.Desc(common3.FieldName("stored_at"))).
		Limit(1).
		Format(db.ci)

	return common.QueryUnique[[]byte](db.readDB, query, args...)
}

func (db *TokenStore) PublicParamsByHash(ctx context.Context, rawHash tdriver.PPHash) ([]byte, error) {
	query, args := q.Select().
		FieldsByName("raw").
		From(q.Table(db.table.PublicParams)).
		Where(cond.Eq("raw_hash", rawHash)).
		Format(db.ci)

	return common.QueryUnique[[]byte](db.readDB, query, args...)
}

func (db *TokenStore) StoreCertifications(ctx context.Context, certifications map[*token.ID][]byte) error {
	if len(certifications) == 0 {
		return nil
	}
	now := time.Now().UTC()

	rows := make([]common3.Tuple, 0, len(certifications))
	for tokenID, certification := range certifications {
		if tokenID == nil {
			return errors.Errorf("invalid token-id, cannot be nil")
		}
		rows = append(rows, common3.Tuple{tokenID.TxId, tokenID.Index, certification, now})
	}
	query, args := q.InsertInto(db.table.Certifications).
		Fields("tx_id", "idx", "certification", "stored_at").
		Rows(rows).
		Format()
	if _, err := db.writeDB.ExecContext(ctx, query, args...); err != nil {
		return tokenDBError(err)
	}
	return nil
}

func (db *TokenStore) ExistsCertification(ctx context.Context, tokenID *token.ID) bool {
	if tokenID == nil {
		return false
	}

	query, args := q.Select().
		FieldsByName("certification").
		From(q.Table(db.table.Certifications)).
		Where(HasTokens("tx_id", "idx", tokenID)).
		Format(db.ci)

	certification, err := common.QueryUnique[[]byte](db.readDB, query, args...)
	if err != nil {
		logger.Warnf("tried to check certification existence for token id %s, err %s", tokenID, err)
		return false
	}

	result := len(certification) != 0
	if !result {
		logger.Warnf("tried to check certification existence for token id %s, got an empty certification", tokenID)
	}
	return result
}

func (db *TokenStore) GetCertifications(ctx context.Context, ids []*token.ID) ([][]byte, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query, args := q.Select().
		FieldsByName("tx_id", "idx", "certification").
		From(q.Table(db.table.Certifications)).
		Where(HasTokens("tx_id", "idx", ids...)).
		Format(db.ci)

	logger.Debug(query, args)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
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

func (db *TokenStore) GetSchema() string {
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

func (db *TokenStore) Close() error {
	return common2.Close(db.readDB, db.writeDB)
}

func (db *TokenStore) NewTokenDBTransaction() (driver.TokenStoreTransaction, error) {
	tx, err := db.writeDB.Begin()
	if err != nil {
		return nil, errors.Errorf("failed starting a db transaction")
	}
	return &TokenTransaction{ci: db.ci, table: &db.table, tx: tx}, nil
}

func (db *TokenStore) SetSupportedTokenFormats(formats []token.Format) error {
	db.sttMutex.Lock()
	db.supportedTokenFormats = formats
	db.sttMutex.Unlock()
	return nil
}

func (db *TokenStore) getSupportedTokenFormats() []token.Format {
	db.sttMutex.RLock()
	supportedTokenTypes := db.supportedTokenFormats
	db.sttMutex.RUnlock()
	return supportedTokenTypes
}

func (db *TokenStore) unspendableTokenFormats(ctx context.Context, walletID string, tokenType token.Type) ([]token.Format, error) {
	query, args := q.SelectDistinct().
		FieldsByName("ledger_type").
		From(q.Table(db.table.Tokens)).
		Where(HasTokenDetails(driver.QueryTokenDetailsParams{
			WalletID:  walletID,
			TokenType: tokenType,
			Spendable: driver.Any,
		}, nil)).
		Format(db.ci)

	logger.Debug(query, args)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "error querying db")
	}
	defer Close(rows)
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "error querying db")
	}
	// read the types from the query result and remove discard those in db.getSupportedTokenFormats()

	supported := collections.NewSet(db.getSupportedTokenFormats()...)
	logger.DebugfContext(ctx, "supported token formats are [%v]", supported)

	all := common.NewIterator(rows, func(f *token.Format) error { return rows.Scan(f) })
	unsupported := iterators.Filter(all, func(f *token.Format) bool { return !supported.Contains(*f) })
	return iterators.ReadAllValues(unsupported)
}

type TokenTransaction struct {
	table *tokenTables
	ci    common3.CondInterpreter
	tx    *sql.Tx
}

func (t *TokenTransaction) GetToken(ctx context.Context, tokenID token.ID, includeDeleted bool) (*token.Token, []string, error) {
	tokenTable, ownershipTable := q.Table(t.table.Tokens), q.Table(t.table.Ownership)
	query, args := q.Select().
		Fields(
			common3.FieldName("owner_raw"), common3.FieldName("token_type"), common3.FieldName("quantity"),
			ownershipTable.Field("wallet_id"), common3.FieldName("owner_wallet_id"),
		).
		From(tokenTable.Join(ownershipTable, cond.And(
			cond.Cmp(tokenTable.Field("tx_id"), "=", ownershipTable.Field("tx_id")),
			cond.Cmp(tokenTable.Field("idx"), "=", ownershipTable.Field("idx"))),
		)).
		Where(HasTokenDetails(driver.QueryTokenDetailsParams{
			IDs:            []*token.ID{&tokenID},
			IncludeDeleted: includeDeleted,
		}, tokenTable)).
		Format(t.ci)

	logger.Debug(query, args)
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer Close(rows)

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
	// We don't delete audit tokens, and we keep the 'ownership' relation.
	query, args := q.Update(t.table.Tokens).
		Set("is_deleted", true).
		Set("spent_by", deletedBy).
		Set("spent_at", time.Now().UTC()).
		Where(cond.And(cond.Eq("tx_id", tokenID.TxId), cond.Eq("idx", tokenID.Index))).
		Format(t.ci)

	logger.Debug(query, args)
	if _, err := t.tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "error setting token to deleted [%s]", tokenID.TxId)
	}

	return nil
}

func (t *TokenTransaction) StoreToken(ctx context.Context, tr driver.TokenRecord, owners []string) error {
	if len(tr.OwnerWalletID) == 0 && len(owners) == 0 && tr.Owner {
		return errors.Errorf("no owners specified [%s]", string(debug.Stack()))
	}

	// Store token
	query, args := q.InsertInto(t.table.Tokens).
		Fields("tx_id", "idx", "issuer_raw", "owner_raw", "owner_type", "owner_identity", "owner_wallet_id", "ledger", "ledger_type", "ledger_metadata", "token_type", "quantity", "amount", "stored_at", "owner", "auditor", "issuer").
		Row(tr.TxID, tr.Index, tr.IssuerRaw, tr.OwnerRaw, tr.OwnerType, tr.OwnerIdentity, tr.OwnerWalletID, tr.Ledger, tr.LedgerFormat, tr.LedgerMetadata, tr.Type, tr.Quantity, tr.Amount, time.Now().UTC(), tr.Owner, tr.Auditor, tr.Issuer).
		Format()
	logger.Debug(query, args)
	if _, err := t.tx.ExecContext(ctx, query, args...); err != nil {
		logger.Errorf("error storing token [%s] in table [%s] [%s]: [%s][%s]", tr.TxID, t.table.Tokens, query, err, string(debug.Stack()))
		return errors.Wrapf(err, "error storing token [%s] in table [%s]", tr.TxID, t.table.Tokens)
	}

	if len(owners) == 0 {
		return nil
	}

	// Store ownership
	rows := make([]common3.Tuple, len(owners))
	for i, eid := range owners {
		rows[i] = common3.Tuple{tr.TxID, tr.Index, eid}
	}
	query, args = q.InsertInto(t.table.Ownership).
		Fields("tx_id", "idx", "wallet_id").
		Rows(rows).
		Format()
	logger.Debug(query, args)

	if _, err := t.tx.ExecContext(ctx, query, args...); err != nil {
		logger.Errorf("error storing token ownerships [%s]: %s", query, err)
		return errors.Wrapf(err, "error storing token ownership [%s]", tr.TxID)
	}

	return nil
}

func (t *TokenTransaction) SetSpendable(ctx context.Context, tokenID token.ID, spendable bool) error {
	query, args := q.Update(t.table.Tokens).
		Set("spendable", spendable).
		Where(cond.And(cond.Eq("tx_id", tokenID.TxId), cond.Eq("idx", tokenID.Index))).
		Format(t.ci)

	logger.Debug(query, args)
	if _, err := t.tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "error setting spendable flag to [%v] for [%s]", spendable, tokenID.TxId)
	}
	return nil
}

func (t *TokenTransaction) SetSpendableBySupportedTokenFormats(ctx context.Context, formats []token.Format) error {
	// first set all spendable flags to false
	query, args := q.Update(t.table.Tokens).
		Set("spendable", false).
		Format(t.ci)

	logger.Debug(query, args)
	if _, err := t.tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "error setting spendable flag to false for all tokens")
	}

	// then set the spendable flags to true only for the supported token types
	query, args = q.Update(t.table.Tokens).
		Set("spendable", true).
		Where(cond.In("ledger_type", formats...)).
		Format(t.ci)

	logger.Debug(query, args)
	res, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrapf(err, "error setting spendable flag to true for token types [%v]", formats)
	} else {
		rows, _ := res.RowsAffected()
		logger.InfofContext(ctx, "rows affected [%d]", rows)
	}

	return nil
}

func (t *TokenTransaction) Commit() error {
	return t.tx.Commit()
}

func (t *TokenTransaction) Rollback() error {
	return t.tx.Rollback()
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
