/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/common"
	tdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type tokenTables struct {
	Tokens         string
	Ownership      string
	PublicParams   string
	Certifications string
	Requests       string
}

type TokenStore struct {
	readDB   *sql.DB
	writeDB  *sql.DB
	table    tokenTables
	pf       sq.PlaceholderFormat
	notifier driver.TokenNotifier

	sttMutex              sync.RWMutex
	supportedTokenFormats []token.Format
}

func newTokenStore(readDB, writeDB *sql.DB, tables tokenTables, pf sq.PlaceholderFormat, notifier driver.TokenNotifier) *TokenStore {
	return &TokenStore{
		readDB:   readDB,
		writeDB:  writeDB,
		table:    tables,
		pf:       pf,
		notifier: notifier,
	}
}

func NewTokenStoreWithNotifier(readDB, writeDB *sql.DB, tables TableNames, pf sq.PlaceholderFormat, notifier driver.TokenNotifier) (*TokenStore, error) {
	return newTokenStore(readDB, writeDB, tokenTables{
		Tokens:         tables.Tokens,
		Ownership:      tables.Ownership,
		PublicParams:   tables.PublicParams,
		Certifications: tables.Certifications,
		Requests:       tables.Requests,
	}, pf, notifier), nil
}

func (db *TokenStore) CreateSchema() error {
	return common.InitSchema(db.writeDB, db.GetSchema())
}

func (db *TokenStore) Notifier() (driver.TokenNotifier, error) {
	if db.notifier == nil {
		return nil, storage.ErrNotSupported
	}

	return db.notifier, nil
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

	query, args, err := sq.Update(db.table.Tokens).
		Set("is_deleted", true).
		Set("spent_by", deletedBy).
		Set("spent_at", time.Now().UTC()).
		Where(HasTokens("tx_id", "idx", ids...)).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return errors.Wrapf(err, "error building delete tokens query")
	}
	logging.Debug(logger, query, args)
	if _, err := db.writeDB.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "error setting tokens to deleted [%v]", ids)
	}

	return nil
}

// IsMine just checks if the token is in the local storage and not deleted
func (db *TokenStore) IsMine(ctx context.Context, txID string, index uint64) (bool, error) {
	query, args, err := sq.Select("tx_id").
		From(db.table.Tokens).
		Where(sq.And{sq.Eq{"tx_id": txID}, sq.Eq{"idx": index}, sq.Eq{"is_deleted": false}, sq.Eq{"owner": true}}).
		Limit(1).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return false, err
	}

	id, err := common.QueryUnique[string](db.readDB, query, args...)

	logger.DebugfContext(ctx, "token [%s:%d] is mine [%s]", txID, index, id)

	return id == txID, err
}

// UnspentTokensIterator returns an iterator over all unspent tokens
func (db *TokenStore) UnspentTokensIterator(ctx context.Context) (tdriver.UnspentTokensIterator, error) {
	return db.UnspentTokensIteratorBy(ctx, "", "")
}

// UnspentTokensIteratorBy returns an iterator of unspent tokens owned by the
// passed wallet id and of the passed type. Empty tokenType returns all types.
//
// Implemented as a single SQL with two UNION ALL branches so each side can
// use its own index, instead of a cross-table OR predicate that PostgreSQL
// cannot resolve with the partial index on (owner_wallet_id, token_type)
// WHERE is_deleted=false AND owner=true:
//
//  1. tokens directly owned: filters tokens.owner_wallet_id, hits the partial
//     index in microseconds. Joins ownership by primary key to preserve the
//     pre-PR semantic that a token must have at least one ownership row to
//     be visible (StoreToken can persist a tokens row without an ownership
//     row when the owners slice is empty, and the original INNER JOIN
//     intentionally excluded those).
//  2. tokens reachable via the ownership-delegation table: joins ownership
//     -> tokens by primary key. Returns zero rows when delegation is not
//     configured, at which point that branch is essentially free.
//
// Both branches share a single SQL statement and therefore a single
// connection from the pool, which avoids the deadlock that would arise if
// two concurrent QueryContexts each tried to acquire a second connection.
// PostgreSQL 9.6+ may also execute the branches in parallel via parallel
// append. UNION ALL is used (not UNION) to skip the per-row sort/hash
// dedup pass; duplicates between the two branches (and within branch 1 when
// a token has multiple ownership rows) are filtered at the iterator layer.
func (db *TokenStore) UnspentTokensIteratorBy(ctx context.Context, walletID string, tokenType token.Type) (tdriver.UnspentTokensIterator, error) {
	query, args, err := sq.Select("t.tx_id", "t.idx", "owner_raw", "token_type", "quantity").
		From(db.table.Tokens + " AS t").
		Join(db.table.Ownership + " AS o ON t.tx_id = o.tx_id AND t.idx = o.idx").
		Where(HasTokenDetails(driver.QueryTokenDetailsParams{
			WalletID:  walletID,
			TokenType: tokenType,
		}, "t")).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return nil, err
	}

	logging.Debug(logger, query, args)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "error querying unspent tokens for wallet [%s] type [%s]", walletID, tokenType)
	}

	return &dedupedTokenRowsIterator{
		rows: rows,
		seen: make(map[string]struct{}),
	}, nil
}

// dedupedTokenRowsIterator yields one (token, ownership.wallet_id) pair per
// (tx_id, idx, ownership.wallet_id) tuple. UNION ALL between branches 1 and
// 2 can emit the same (token, ownership) row twice when walletID matches
// both tokens.owner_wallet_id and ownership.wallet_id of the same row;
// pre-PR behaviour was a single row in that case. Distinct (token,
// ownership) pairs (e.g. shared-ownership tokens with multiple wallets in
// the ownership table) are preserved — they have different keys.
//
// The trailing wallet_id column is read for the dedup key only and is not
// surfaced on token.UnspentToken. ownership.wallet_id can be NULL when the
// LEFT JOIN finds no matching ownership row (a tokens row with
// owner_wallet_id set but no entry in the ownership table — StoreToken
// allows that when owners is empty), so it is scanned as sql.NullString.
type dedupedTokenRowsIterator struct {
	rows *sql.Rows
	seen map[string]struct{}
}

func (it *dedupedTokenRowsIterator) Close() {
	_ = it.rows.Close()
}

func (it *dedupedTokenRowsIterator) Next() (*token.UnspentToken, error) {
	for it.rows.Next() {
		var t token.UnspentToken
		var ownerID sql.NullString
		if err := it.rows.Scan(&t.Id.TxId, &t.Id.Index, &t.Owner, &t.Type, &t.Quantity, &ownerID); err != nil {
			return nil, err
		}
		// "\x00" prefix on a Valid wallet_id can never collide with the
		// empty-string value used for NULL because the prefix is reserved
		// here; without it, NULL and "" would share a key.
		var ownerKey string
		if ownerID.Valid {
			ownerKey = "\x00" + ownerID.String
		}
		key := fmt.Sprintf("%s:%d:%s", t.Id.TxId, t.Id.Index, ownerKey)
		if _, dup := it.seen[key]; dup {
			continue
		}
		it.seen[key] = struct{}{}

		return &t, nil
	}

	return nil, nil
}

// SpendableTokensIteratorBy returns the minimum information about the tokens needed for the selector
func (db *TokenStore) SpendableTokensIteratorBy(ctx context.Context, walletID string, typ token.Type) (tdriver.SpendableTokensIterator, error) {
	query, args, err := sq.Select("tx_id", "idx", "token_type", "quantity", "owner_wallet_id").
		From(db.table.Tokens).
		Where(HasTokenDetails(driver.QueryTokenDetailsParams{
			WalletID:           walletID,
			TokenType:          typ,
			Spendable:          driver.SpendableOnly,
			LedgerTokenFormats: db.getSupportedTokenFormats(),
		}, "")).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return nil, errors.Wrapf(err, "error querying db")
	}

	logging.Debug(logger, query, args)
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
	query, args, err := sq.Select("tx_id", "idx", "ledger", "ledger_metadata", "ledger_type").
		From(db.table.Tokens).
		Where(HasTokenDetails(details, "")).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return nil, errors.Wrapf(err, "error querying db")
	}

	logging.Debug(logger, query, args)

	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "error querying db")
	}

	return common.NewIterator(rows, func(tok *token.LedgerToken) error {
		return rows.Scan(&tok.ID.TxId, &tok.ID.Index, &tok.Token, &tok.TokenMetadata, &tok.Format)
	}), nil
}

// Balance returns the sun of the amounts, with 64 bits of precision, of the tokens with type and EID equal to those passed as arguments.
func (db *TokenStore) Balance(ctx context.Context, walletID string, typ token.Type) (*big.Int, error) {
	return db.balance(ctx, driver.QueryTokenDetailsParams{
		WalletID:  walletID,
		TokenType: typ,
	})
}

func (db *TokenStore) balance(ctx context.Context, opts driver.QueryTokenDetailsParams) (*big.Int, error) {
	query, args, err := sq.Select("SUM(amount)").
		From(db.table.Tokens + " AS t").
		Join(db.table.Ownership + " AS o ON t.tx_id = o.tx_id AND t.idx = o.idx").
		Where(HasTokenDetails(opts, "t")).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return nil, err
	}

	sum, err := common.QueryUnique[*uint64](db.readDB, query, args...)
	if err != nil || sum == nil {
		return nil, err
	}

	return new(big.Int).SetUint64(*sum), nil
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

// ListUnspentTokensByWallets issues a single SELECT with an IN clause on
// the owning wallet, then partitions the result rows in Go. This avoids
// the N round-trip cost of calling ListUnspentTokensBy in a loop when a
// caller (e.g. a bank-node wallet list endpoint) needs balances for many
// wallets at once. Empty input returns an empty map without querying.
func (db *TokenStore) ListUnspentTokensByWallets(ctx context.Context, walletIDs []string, typ token.Type) (map[string]*token.UnspentTokens, error) {
	if len(walletIDs) == 0 {
		return map[string]*token.UnspentTokens{}, nil
	}

	// owner_wallet_id lives on Tokens, wallet_id lives on Ownership. The
	// WHERE condition (HasTokenDetails) matches a row if either column is
	// in walletIDs, so we project both and bucket each row (below) under
	// whichever column is actually in the requested set. A single token
	// can show up through the Ownership join multiple times if it has
	// multiple wallet owners; that is intentional — each (token, wallet)
	// pair contributes one row.
	query, args, err := sq.Select("o.wallet_id", "t.owner_wallet_id", "t.tx_id", "t.idx", "owner_raw", "token_type", "quantity").
		From(db.table.Tokens + " AS t").
		Join(db.table.Ownership + " AS o ON t.tx_id = o.tx_id AND t.idx = o.idx").
		Where(HasTokenDetails(driver.QueryTokenDetailsParams{
			WalletIDs: walletIDs,
			TokenType: typ,
		}, "t")).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return nil, err
	}

	logging.Debug(logger, query, args)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer Close(rows)

	// The WHERE clause matches rows where EITHER ownership.wallet_id OR
	// tokens.owner_wallet_id is in walletIDs, but the two columns are not
	// constrained to agree (StoreToken writes them independently). Bucket
	// under whichever column is actually in the requested set, preferring
	// ownership.wallet_id so a single input id always maps to a single key.
	walletIDSet := make(map[string]struct{}, len(walletIDs))
	for _, id := range walletIDs {
		walletIDSet[id] = struct{}{}
	}

	result := make(map[string]*token.UnspentTokens, len(walletIDs))
	for rows.Next() {
		var walletCol, ownerWalletCol sql.NullString
		ut := &token.UnspentToken{}
		if err := rows.Scan(
			&walletCol,
			&ownerWalletCol,
			&ut.Id.TxId, &ut.Id.Index, &ut.Owner, &ut.Type, &ut.Quantity,
		); err != nil {
			return nil, err
		}
		walletID := ""
		if walletCol.Valid && walletCol.String != "" {
			if _, ok := walletIDSet[walletCol.String]; ok {
				walletID = walletCol.String
			}
		}
		if walletID == "" && ownerWalletCol.Valid && ownerWalletCol.String != "" {
			if _, ok := walletIDSet[ownerWalletCol.String]; ok {
				walletID = ownerWalletCol.String
			}
		}
		if walletID == "" {
			continue
		}
		bucket, ok := result[walletID]
		if !ok {
			bucket = &token.UnspentTokens{}
			result[walletID] = bucket
		}
		bucket.Tokens = append(bucket.Tokens, ut)
	}

	return result, rows.Err()
}

// ListAuditTokens returns the audited tokens associated to the passed ids
func (db *TokenStore) ListAuditTokens(ctx context.Context, ids ...*token.ID) ([]*token.Token, error) {
	if len(ids) == 0 {
		return []*token.Token{}, nil
	}

	query, args, err := sq.Select("tx_id", "idx", "owner_raw", "token_type", "quantity").
		From(db.table.Tokens).
		Where(sq.And{HasTokens("tx_id", "idx", ids...), sq.Eq{"auditor": true}}).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return nil, err
	}

	logging.Debug(logger, query, args)
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
	query, args, err := sq.Select("tx_id", "idx", "owner_raw", "token_type", "quantity", "issuer_raw").
		From(db.table.Tokens).
		Where(sq.Eq{"issuer": true}).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return nil, err
	}

	logging.Debug(logger, query)
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

	query, args, err := sq.Select("tx_id", "idx", "ledger").
		From(db.table.Tokens).
		Where(HasTokens("tx_id", "idx", ids...)).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return nil, err
	}

	logging.Debug(logger, query, args)
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

	query, args, err := sq.Select("tx_id", "idx", "ledger", "ledger_type", "ledger_metadata").
		From(db.table.Tokens).
		Where(HasTokens("tx_id", "idx", ids...)).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return nil, nil, nil, err
	}

	logging.Debug(logger, query, args)
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

	query, args, err := sq.Select("tx_id", "idx", "owner_raw", "token_type", "quantity").
		From(db.table.Tokens).
		Where(sq.And{HasTokens("tx_id", "idx", inputs...), sq.Eq{"is_deleted": false}, sq.Eq{"owner": true}}).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return nil, err
	}

	logging.Debug(logger, query, args)
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
	query, args, err := sq.Select("t.tx_id", "t.idx", "owner_identity", "owner_type", "wallet_id", "token_type", "amount", "is_deleted", "spent_by", "stored_at").
		From(db.table.Tokens + " AS t").
		Join(db.table.Ownership + " AS o ON t.tx_id = o.tx_id AND t.idx = o.idx").
		Where(HasTokenDetails(params, "t")).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return nil, err
	}

	logging.Debug(logger, query, args)
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

	query, args, err := sq.Select("tx_id", "idx", "spent_by", "is_deleted").
		From(db.table.Tokens).
		Where(HasTokens("tx_id", "idx", inputs...)).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return nil, nil, err
	}

	logging.Debug(logger, query, args)
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
	query, args, err := sq.Select("tx_id").
		From(db.table.Tokens).
		Where(sq.Eq{"tx_id": id}).
		Limit(1).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return false, err
	}

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

	query, args, err := sq.Insert(db.table.PublicParams).
		Columns("raw", "raw_hash", "stored_at").
		Values(raw, rawHash, time.Now().UTC()).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return err
	}
	logger.DebugfContext(ctx, query, fmt.Sprintf("store public parameters (%d bytes), hash [%s]", len(raw), logging.Base64(rawHash)))
	_, err = db.writeDB.ExecContext(ctx, query, args...)

	return err
}

func (db *TokenStore) PublicParams(ctx context.Context) ([]byte, error) {
	query, args, err := sq.Select("raw").
		From(db.table.PublicParams).
		OrderBy("stored_at DESC").
		Limit(1).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return nil, err
	}

	return common.QueryUnique[[]byte](db.readDB, query, args...)
}

func (db *TokenStore) PublicParamsByHash(ctx context.Context, rawHash tdriver.PPHash) ([]byte, error) {
	query, args, err := sq.Select("raw").
		From(db.table.PublicParams).
		Where(sq.Eq{"raw_hash": rawHash}).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return nil, err
	}

	return common.QueryUnique[[]byte](db.readDB, query, args...)
}

func (db *TokenStore) StoreCertifications(ctx context.Context, certifications map[*token.ID][]byte) error {
	if len(certifications) == 0 {
		return nil
	}
	now := time.Now().UTC()

	insert := sq.Insert(db.table.Certifications).Columns("tx_id", "idx", "certification", "stored_at")
	for tokenID, certification := range certifications {
		if tokenID == nil {
			return errors.Errorf("invalid token-id, cannot be nil")
		}
		insert = insert.Values(tokenID.TxId, tokenID.Index, certification, now)
	}
	query, args, err := insert.PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return err
	}
	if _, err := db.writeDB.ExecContext(ctx, query, args...); err != nil {
		return tokenDBError(err)
	}

	return nil
}

func (db *TokenStore) ExistsCertification(ctx context.Context, tokenID *token.ID) bool {
	if tokenID == nil {
		return false
	}

	query, args, err := sq.Select("certification").
		From(db.table.Certifications).
		Where(HasTokens("tx_id", "idx", tokenID)).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		logger.Warnf("tried to check certification existence for token id %s, err %s", tokenID, err)

		return false
	}

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

	query, args, err := sq.Select("tx_id", "idx", "certification").
		From(db.table.Certifications).
		Where(HasTokens("tx_id", "idx", ids...)).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build query")
	}

	logging.Debug(logger, query, args)
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
		-- Requests
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
		CREATE INDEX IF NOT EXISTS idx_owner_wallet_id_%s ON %s ( owner_wallet_id );
		CREATE INDEX IF NOT EXISTS idx_owner_wallet_part_%s ON %s ( owner_wallet_id, token_type ) WHERE is_deleted = false AND owner = true;

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
		db.table.Requests, db.table.Requests, db.table.Requests, db.table.Requests, db.table.Requests,
		db.table.Tokens,
		db.table.Tokens, db.table.Tokens,
		db.table.Tokens, db.table.Tokens,
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

	return &TokenTransaction{pf: db.pf, table: &db.table, tx: tx}, nil
}

func (db *TokenStore) ContinueTokenDBTransaction(tx driver.Transaction) (driver.TokenStoreTransaction, error) {
	sqlTx, ok := tx.Impl().(*sql.Tx)
	if !ok {
		return nil, errors.Errorf("failed continuing a db transaction, expecting an sql transaction")
	}

	return &TokenTransaction{pf: db.pf, table: &db.table, tx: sqlTx}, nil
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
	query, args, err := sq.Select("DISTINCT ledger_type").
		From(db.table.Tokens).
		Where(HasTokenDetails(driver.QueryTokenDetailsParams{
			WalletID:  walletID,
			TokenType: tokenType,
			Spendable: driver.Any,
		}, "")).
		PlaceholderFormat(db.pf).ToSql()
	if err != nil {
		return nil, errors.Wrapf(err, "error querying db")
	}

	logging.Debug(logger, query, args)
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
	pf    sq.PlaceholderFormat
	tx    *sql.Tx
}

func (t *TokenTransaction) GetToken(ctx context.Context, tokenID token.ID, includeDeleted bool) (*token.Token, []string, error) {
	query, args, err := sq.Select("owner_raw", "token_type", "quantity", "o.wallet_id", "owner_wallet_id").
		From(t.table.Tokens + " AS t").
		LeftJoin(t.table.Ownership + " AS o ON t.tx_id = o.tx_id AND t.idx = o.idx").
		Where(HasTokenDetails(driver.QueryTokenDetailsParams{
			IDs:            []*token.ID{&tokenID},
			IncludeDeleted: includeDeleted,
		}, "t")).
		PlaceholderFormat(t.pf).ToSql()
	if err != nil {
		return nil, nil, err
	}

	logging.Debug(logger, query, args)
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
	query, args, err := sq.Update(t.table.Tokens).
		Set("is_deleted", true).
		Set("spent_by", deletedBy).
		Set("spent_at", time.Now().UTC()).
		Where(sq.And{sq.Eq{"tx_id": tokenID.TxId}, sq.Eq{"idx": tokenID.Index}}).
		PlaceholderFormat(t.pf).ToSql()
	if err != nil {
		return errors.Wrapf(err, "error building delete query for token [%s]", tokenID.TxId)
	}

	logging.Debug(logger, query, args)
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
	query, args, err := sq.Insert(t.table.Tokens).
		Columns("tx_id", "idx", "issuer_raw", "owner_raw", "owner_type", "owner_identity", "owner_wallet_id", "ledger", "ledger_type", "ledger_metadata", "token_type", "quantity", "amount", "stored_at", "owner", "auditor", "issuer").
		Values(tr.TxID, tr.Index, tr.IssuerRaw, tr.OwnerRaw, tr.OwnerType, tr.OwnerIdentity, tr.OwnerWalletID, tr.Ledger, tr.LedgerFormat, tr.LedgerMetadata, tr.Type, tr.Quantity, tr.Amount, time.Now().UTC(), tr.Owner, tr.Auditor, tr.Issuer).
		PlaceholderFormat(t.pf).ToSql()
	if err != nil {
		return errors.Wrapf(err, "error building store token query [%s]", tr.TxID)
	}
	logging.Debug(logger, query, args)
	if _, err := t.tx.ExecContext(ctx, query, args...); err != nil {
		logger.Errorf("error storing token [%s] in table [%s] [%s]: [%s][%s]", tr.TxID, t.table.Tokens, query, err, string(debug.Stack()))

		return errors.Wrapf(err, "error storing token [%s] in table [%s]", tr.TxID, t.table.Tokens)
	}

	if len(owners) == 0 {
		logger.Debugf("no additional owner reference apart [%s]", tr.OwnerWalletID)

		return nil
	}

	// Store ownership
	insert := sq.Insert(t.table.Ownership).Columns("tx_id", "idx", "wallet_id")
	for _, eid := range owners {
		insert = insert.Values(tr.TxID, tr.Index, eid)
	}
	query, args, err = insert.PlaceholderFormat(t.pf).ToSql()
	if err != nil {
		return errors.Wrapf(err, "error building store ownership query [%s]", tr.TxID)
	}
	logging.Debug(logger, query, args)

	if _, err := t.tx.ExecContext(ctx, query, args...); err != nil {
		logger.Errorf("error storing token ownerships [%s]: %s", query, err)

		return errors.Wrapf(err, "error storing token ownership [%s]", tr.TxID)
	}

	return nil
}

func (t *TokenTransaction) SetSpendable(ctx context.Context, tokenID token.ID, spendable bool) error {
	query, args, err := sq.Update(t.table.Tokens).
		Set("spendable", spendable).
		Where(sq.And{sq.Eq{"tx_id": tokenID.TxId}, sq.Eq{"idx": tokenID.Index}}).
		PlaceholderFormat(t.pf).ToSql()
	if err != nil {
		return errors.Wrapf(err, "error building set spendable query")
	}

	logging.Debug(logger, query, args)
	if _, err := t.tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "error setting spendable flag to [%v] for [%s]", spendable, tokenID.TxId)
	}

	return nil
}

func (t *TokenTransaction) SetSpendableBySupportedTokenFormats(ctx context.Context, formats []token.Format) error {
	// first set all spendable flags to false
	query, args, err := sq.Update(t.table.Tokens).
		Set("spendable", false).
		PlaceholderFormat(t.pf).ToSql()
	if err != nil {
		return errors.Wrapf(err, "error building set spendable false query")
	}

	logging.Debug(logger, query, args)
	if _, err := t.tx.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "error setting spendable flag to false for all tokens")
	}

	// then set the spendable flags to true only for the supported token types
	query, args, err = sq.Update(t.table.Tokens).
		Set("spendable", true).
		Where(sq.Eq{"ledger_type": formats}).
		PlaceholderFormat(t.pf).ToSql()
	if err != nil {
		return errors.Wrapf(err, "error building set spendable true query")
	}

	logging.Debug(logger, query, args)
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
