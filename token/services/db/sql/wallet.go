/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
)

type walletTables struct {
	Wallets string
}

type WalletDB struct {
	db    *sql.DB
	table walletTables
}

func newWalletDB(db *sql.DB, tables walletTables) *WalletDB {
	return &WalletDB{
		db:    db,
		table: tables,
	}
}

func NewWalletDB(db *sql.DB, tablePrefix, name string, createSchema bool) (*WalletDB, error) {
	tables, err := getTableNames(tablePrefix, name)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table names for prefix [%s] and name [%s]", tablePrefix, name)
	}

	walletDB := newWalletDB(db, walletTables{Wallets: tables.Wallets})
	if createSchema {
		if err = initSchema(db, walletDB.GetSchema()); err != nil {
			return nil, errors.Wrapf(err, "failed to create schema")
		}
	}
	return walletDB, nil
}

func (db *WalletDB) StoreWalletID(driver.WalletID) error {
	return nil
}

func (db *WalletDB) GetWalletID(id view.Identity) (driver.WalletID, error) {
	idHash := id.Hash()
	result, err := QueryUnique[driver.WalletID](db.db,
		fmt.Sprintf("SELECT wallet_id FROM %s WHERE identity_id=$1", db.table.Wallets),
		idHash,
	)
	if err != nil {
		return "", errors.Wrapf(err, "failed getting wallet id for identity [%v]", idHash)
	}
	logger.Debugf("found wallet id for identity [%v]: %v", idHash, result)
	return result, nil
}

func (db *WalletDB) GetWalletIDs(roleID int) ([]driver.WalletID, error) {
	query := fmt.Sprintf("SELECT DISTINCT wallet_id FROM %s WHERE role = $1", db.table.Wallets)
	logger.Debug(query)
	rows, err := db.db.Query(query, roleID)
	if err != nil {
		return nil, err
	}

	var walletIDs []driver.WalletID
	for rows.Next() {
		var walletID driver.WalletID
		if err := rows.Scan(&walletID); err != nil {
			return nil, err
		}
		walletIDs = append(walletIDs, walletID)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	logger.Debugf("found %d wallet ids: [%v]", len(walletIDs), walletIDs)
	return walletIDs, nil
}

func (db *WalletDB) StoreIdentity(identity view.Identity, wID driver.WalletID, roleID int, meta any) error {
	if db.IdentityExists(identity, wID) {
		return nil
	}

	query := fmt.Sprintf("INSERT INTO %s (identity_id, meta, wallet_id, role, created_at) VALUES ($1, $2, $3, $4, $5)", db.table.Wallets)
	logger.Debug(query)
	metaEncoded, err := json.Marshal(meta)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal metadata")
	}

	idHash := identity.Hash()
	_, err = db.db.Exec(query, idHash, metaEncoded, wID, roleID, time.Now().UTC())
	if err != nil {
		return errors.Wrapf(err, "failed storing wallet [%v] for identity [%v]", wID, idHash)
	}
	logger.Debugf("stored wallet [%v] for identity [%v]", wID, idHash)
	return nil
}

func (db *WalletDB) LoadMeta(identity view.Identity, meta any) error {
	idHash := identity.Hash()
	result, err := QueryUnique[string](db.db,
		fmt.Sprintf("SELECT meta FROM %s WHERE identity_id=$1", db.table.Wallets),
		idHash,
	)
	if err != nil {
		return errors.Wrapf(err, "failed loading meta for id [%v]", idHash)
	}
	logger.Debugf("Loaded meta for id [%v, %v]: %v", identity, idHash, result)
	return json.Unmarshal([]byte(result), &meta)
}

func (db *WalletDB) IdentityExists(identity view.Identity, wID driver.WalletID) bool {
	idHash := identity.Hash()
	result, err := QueryUnique[driver.WalletID](db.db,
		fmt.Sprintf("SELECT wallet_id FROM %s WHERE identity_id=$1 AND wallet_id=$2", db.table.Wallets),
		idHash, wID,
	)
	if err != nil {
		logger.Errorf("failed looking up wallet-identity [%s-%s]: %w", wID, idHash, err)
	}
	logger.Debugf("found identity for wallet-identity [%v-%v]: %v", wID, idHash, result)

	return result != ""
}

func (db *WalletDB) GetSchema() string {
	return fmt.Sprintf(`
		-- Wallets
		CREATE TABLE IF NOT EXISTS %s (
			identity_id BYTEA NOT NULL,
			wallet_id TEXT NOT NULL,
			meta TEXT NOT NULL DEFAULT '',
            role INT NOT NULL,
			created_at TIMESTAMP,
			PRIMARY KEY(identity_id, wallet_id, role) ON CONFLICT REPLACE
		);
		CREATE INDEX IF NOT EXISTS idx_identity_id_%s ON %s ( identity_id );
		CREATE INDEX IF NOT EXISTS idx_identity_id_and_wallet_%s ON %s ( identity_id, wallet_id )
		`,
		db.table.Wallets,
		db.table.Wallets, db.table.Wallets,
		db.table.Wallets, db.table.Wallets,
	)
}
