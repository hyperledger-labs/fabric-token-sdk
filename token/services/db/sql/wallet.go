/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"database/sql"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
)

type WalletDB struct {
	db *sql.DB
}

func NewWalletDB(db *sql.DB, createSchema bool) (driver.WalletDB, error) {
	walletDB := &WalletDB{
		db: db,
	}
	if createSchema {
		if err := initSchema(db, walletDB.GetSchema()); err != nil {
			return nil, errors.Wrapf(err, "failed to create schema")
		}
	}
	return walletDB, nil
}

func (db *WalletDB) GetWalletID(identity token.Identity, roleID int) (driver.WalletID, error) {
	idHash := identity.UniqueID()
	result, err := QueryUnique[driver.WalletID](db.db,
		"SELECT wallet_id FROM wallets WHERE identity_hash=$1 AND role_id=$2",
		idHash, roleID,
	)
	if err != nil {
		return "", errors.Wrapf(err, "failed getting wallet id for identity [%v]", idHash)
	}
	logger.Debugf("found wallet id for identity [%v]: %v", idHash, result)
	return result, nil
}

func (db *WalletDB) GetWalletIDs(roleID int) ([]driver.WalletID, error) {
	query := "SELECT DISTINCT wallet_id FROM wallets WHERE role_id = $1"
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

func (db *WalletDB) StoreIdentity(identity token.Identity, eID string, wID driver.WalletID, roleID int, meta []byte) error {
	if db.IdentityExists(identity, wID, roleID) {
		return nil
	}

	query := "INSERT INTO wallets (identity_hash, meta, wallet_id, role_id, created_at, enrollment_id) VALUES ($1, $2, $3, $4, $5, $6)"
	logger.Debug(query)

	idHash := identity.UniqueID()
	_, err := db.db.Exec(query, idHash, meta, wID, roleID, time.Now().UTC(), eID)
	if err != nil {
		return errors.Wrapf(err, "failed storing wallet [%v] for identity [%v]", wID, idHash)
	}
	logger.Debugf("stored wallet [%v] for identity [%v]", wID, idHash)
	return nil
}

func (db *WalletDB) LoadMeta(identity token.Identity, wID driver.WalletID, roleID int) ([]byte, error) {
	idHash := identity.UniqueID()
	result, err := QueryUnique[[]byte](db.db,
		"SELECT meta FROM wallets WHERE identity_hash=$1 AND wallet_id=$2 AND role_id=$3",
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
	result, err := QueryUnique[driver.WalletID](db.db,
		"SELECT wallet_id FROM wallets WHERE identity_hash=$1 AND wallet_id=$2 AND role_id=$3",
		idHash, wID, roleID,
	)
	if err != nil {
		logger.Errorf("failed looking up wallet-identity [%s-%s]: %w", wID, idHash, err)
	}
	logger.Debugf("found identity for wallet-identity [%v-%v]: %v", wID, idHash, result)

	return result != ""
}

func (db *WalletDB) GetSchema() string {
	return `
	-- Wallets
	CREATE TABLE IF NOT EXISTS wallets (
		identity_hash TEXT NOT NULL,
		wallet_id TEXT NOT NULL,
		meta BYTEA,
		role_id INT NOT NULL,
		enrollment_id TEXT NOT NULL,	
		created_at TIMESTAMP,
		PRIMARY KEY(identity_hash, wallet_id, role_id)
	);
	CREATE INDEX IF NOT EXISTS idx_identity_hash_wallets ON wallets ( identity_hash );
	CREATE INDEX IF NOT EXISTS idx_identity_hash_and_wallet_and_role ON wallets ( identity_hash, wallet_id, role_id );
	CREATE INDEX IF NOT EXISTS idx_identity_hash_and_role ON wallets ( identity_hash, role_id );
	CREATE INDEX IF NOT EXISTS idx_role_id_wallets ON wallets ( role_id )
	`
}
