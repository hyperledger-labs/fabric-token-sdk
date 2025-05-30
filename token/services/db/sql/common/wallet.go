/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	q "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/query"
	common3 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/query/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/query/cond"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
)

type walletTables struct {
	Wallets string
}

type WalletStore struct {
	readDB  *sql.DB
	writeDB *sql.DB
	table   walletTables
	ci      common3.CondInterpreter
}

func newWalletStore(readDB, writeDB *sql.DB, tables walletTables, ci common3.CondInterpreter) *WalletStore {
	return &WalletStore{
		readDB:  readDB,
		writeDB: writeDB,
		table:   tables,
		ci:      ci,
	}
}

func NewWalletStore(readDB, writeDB *sql.DB, tables TableNames, ci common3.CondInterpreter) (*WalletStore, error) {
	return newWalletStore(readDB, writeDB, walletTables{Wallets: tables.Wallets}, ci), nil
}

func (db *WalletStore) CreateSchema() error {
	return common.InitSchema(db.writeDB, []string{db.GetSchema()}...)
}

func (db *WalletStore) GetWalletID(ctx context.Context, identity token.Identity, roleID int) (driver.WalletID, error) {
	idHash := identity.UniqueID()
	query, args := q.Select().
		FieldsByName("wallet_id").
		From(q.Table(db.table.Wallets)).
		Where(cond.And(cond.Eq("identity_hash", idHash), cond.Eq("role_id", roleID))).
		Format(db.ci)

	result, err := common.QueryUnique[driver.WalletID](db.readDB, query, args...)
	if err != nil {
		return "", errors.Wrapf(err, "failed getting wallet id for identity [%v]", idHash)
	}
	logger.Debugf("found wallet id for identity [%v]: %v", idHash, result)
	return result, nil
}

func (db *WalletStore) GetWalletIDs(ctx context.Context, roleID int) ([]driver.WalletID, error) {
	query, args := q.SelectDistinct().
		FieldsByName("wallet_id").
		From(q.Table(db.table.Wallets)).
		Where(cond.Eq("role_id", roleID)).
		Format(db.ci)
	logger.Debug(query)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	it := common.NewIterator(rows, func(walletID *driver.WalletID) error { return rows.Scan(walletID) })
	return iterators.ReadAllValues(it)
}

func (db *WalletStore) StoreIdentity(ctx context.Context, identity token.Identity, eID string, wID driver.WalletID, roleID int, meta []byte) error {
	// TODO AF Use upsert
	if db.IdentityExists(ctx, identity, wID, roleID) {
		return nil
	}

	query, args := q.InsertInto(db.table.Wallets).
		Fields("identity_hash", "meta", "wallet_id", "role_id", "created_at", "enrollment_id").
		Row(identity.UniqueID(), meta, wID, roleID, time.Now().UTC(), eID).
		Format()
	logger.Debug(query)

	_, err := db.writeDB.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrapf(err, "failed storing wallet [%v] for identity [%s]", wID, identity)
	}
	logger.Debugf("stored wallet [%v] for identity [%s]", wID, identity)
	return nil
}

func (db *WalletStore) LoadMeta(ctx context.Context, identity token.Identity, wID driver.WalletID, roleID int) ([]byte, error) {
	idHash := identity.UniqueID()
	query, args := q.Select().
		FieldsByName("meta").
		From(q.Table(db.table.Wallets)).
		Where(cond.And(cond.Eq("identity_hash", idHash), cond.Eq("wallet_id", wID), cond.Eq("role_id", roleID))).
		Format(db.ci)
	result, err := common.QueryUnique[[]byte](db.readDB, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed loading meta for id [%v]", idHash)
	}
	logger.Debugf("loaded meta for id [%v, %v]: %v", identity, idHash, result)
	return result, nil
}

func (db *WalletStore) IdentityExists(ctx context.Context, identity token.Identity, wID driver.WalletID, roleID int) bool {
	idHash := identity.UniqueID()
	query, args := q.Select().
		FieldsByName("wallet_id").
		From(q.Table(db.table.Wallets)).
		Where(cond.And(cond.Eq("identity_hash", idHash), cond.Eq("wallet_id", wID), cond.Eq("role_id", roleID))).
		Format(db.ci)
	result, err := common.QueryUnique[driver.WalletID](db.readDB, query, args...)
	if err != nil {
		logger.Errorf("failed looking up wallet-identity [%s-%s]: %w", wID, idHash, err)
	}
	logger.Debugf("found identity for wallet-identity [%v-%v]: %v", wID, idHash, result)

	return result != ""
}

func (db *WalletStore) GetSchema() string {
	return fmt.Sprintf(`
		-- Wallets
		CREATE TABLE IF NOT EXISTS %s (
			identity_hash TEXT NOT NULL,
			wallet_id TEXT NOT NULL,
			meta BYTEA,
            role_id INT NOT NULL,
			enrollment_id TEXT NOT NULL,	
			created_at TIMESTAMP,
			PRIMARY KEY(identity_hash, wallet_id, role_id)
		);
		CREATE INDEX IF NOT EXISTS idx_identity_hash_%s ON %s ( identity_hash );
		CREATE INDEX IF NOT EXISTS idx_identity_hash_and_wallet_and_role%s ON %s ( identity_hash, wallet_id, role_id );
		CREATE INDEX IF NOT EXISTS idx_identity_hash_and_role%s ON %s ( identity_hash, role_id );
		CREATE INDEX IF NOT EXISTS idx_role_id_%s ON %s ( role_id )
		`,
		db.table.Wallets,
		db.table.Wallets, db.table.Wallets,
		db.table.Wallets, db.table.Wallets,
		db.table.Wallets, db.table.Wallets,
		db.table.Wallets, db.table.Wallets,
	)
}

func (db *WalletStore) Close() error {
	return common2.Close(db.readDB, db.writeDB)
}
