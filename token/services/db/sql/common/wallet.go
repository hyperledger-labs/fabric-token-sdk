/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
)

type walletTables struct {
	Wallets string
}

type WalletDB struct {
	readDB  *sql.DB
	writeDB *sql.DB
	table   walletTables
}

func newWalletDB(readDB, writeDB *sql.DB, tables walletTables) *WalletDB {
	return &WalletDB{
		readDB:  readDB,
		writeDB: writeDB,
		table:   tables,
	}
}

func NewWalletDB(readDB, writeDB *sql.DB, opts NewDBOpts) (driver.WalletDB, error) {
	tables, err := GetTableNames(opts.TablePrefix)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table names [%s]", opts.TablePrefix)
	}

	walletDB := newWalletDB(readDB, writeDB, walletTables{Wallets: tables.Wallets})
	if opts.CreateSchema {
		if err = common.InitSchema(writeDB, []string{walletDB.GetSchema()}...); err != nil {
			return nil, errors.Wrapf(err, "failed to create schema")
		}
	}
	return walletDB, nil
}

func (db *WalletDB) GetWalletID(identity token.Identity, roleID int) (driver.WalletID, error) {
	idHash := identity.UniqueID()
	result, err := common.QueryUnique[driver.WalletID](db.readDB,
		fmt.Sprintf("SELECT wallet_id FROM %s WHERE identity_hash=$1 AND role_id=$2", db.table.Wallets),
		idHash, roleID,
	)
	if err != nil {
		return "", errors.Wrapf(err, "failed getting wallet id for identity [%v]", idHash)
	}
	logger.Debugf("found wallet id for identity [%v]: %v", idHash, result)
	return result, nil
}

func (db *WalletDB) GetWalletIDs(roleID int) ([]driver.WalletID, error) {
	query, err := NewSelectDistinct("wallet_id").From(db.table.Wallets).Where("role_id = $1").Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed compiling query")
	}
	logger.Debug(query)
	rows, err := db.readDB.Query(query, roleID)
	if err != nil {
		return nil, err
	}
	defer Close(rows)

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

func (db *WalletDB) StoreIdentity(identity token.Identity, eID string, wID driver.WalletID, roleID int, meta []byte) error {
	if db.IdentityExists(identity, wID, roleID) {
		return nil
	}

	query, err := NewInsertInto(db.table.Wallets).Rows("identity_hash, meta, wallet_id, role_id, created_at, enrollment_id").Compile()
	if err != nil {
		return errors.Wrapf(err, "failed compiling query")
	}
	logger.Debug(query)

	idHash := identity.UniqueID()
	_, err = db.writeDB.Exec(query, idHash, meta, wID, roleID, time.Now().UTC(), eID)
	if err != nil {
		return errors.Wrapf(err, "failed storing wallet [%v] for identity [%v]", wID, idHash)
	}
	logger.Debugf("stored wallet [%v] for identity [%v]", wID, idHash)
	return nil
}

func (db *WalletDB) LoadMeta(identity token.Identity, wID driver.WalletID, roleID int) ([]byte, error) {
	idHash := identity.UniqueID()
	result, err := common.QueryUnique[[]byte](db.readDB,
		fmt.Sprintf("SELECT meta FROM %s WHERE identity_hash=$1 AND wallet_id=$2 AND role_id=$3", db.table.Wallets),
		idHash, wID, roleID,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed loading meta for id [%v]", idHash)
	}
	logger.Debugf("loaded meta for id [%v, %v]: %v", identity, idHash, result)
	return result, nil
}

func (db *WalletDB) IdentityExists(identity token.Identity, wID driver.WalletID, roleID int) bool {
	idHash := identity.UniqueID()
	result, err := common.QueryUnique[driver.WalletID](db.readDB,
		fmt.Sprintf("SELECT wallet_id FROM %s WHERE identity_hash=$1 AND wallet_id=$2 AND role_id=$3", db.table.Wallets),
		idHash, wID, roleID,
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
