/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"bytes"
	"database/sql"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
)

type identityTables struct {
	IdentityConfigurations string
	AuditInfo              string
	Signers                string
}

type IdentityDB struct {
	db    *sql.DB
	table identityTables
}

func newIdentityDB(db *sql.DB, tables identityTables) *IdentityDB {
	return &IdentityDB{
		db:    db,
		table: tables,
	}
}

func NewIdentityDB(db *sql.DB, tablePrefix, name string, createSchema bool) (*IdentityDB, error) {
	tables, err := getTableNames(tablePrefix, name)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table names")
	}

	identityDB := newIdentityDB(db, identityTables{
		IdentityConfigurations: tables.IdentityConfigurations,
		AuditInfo:              tables.AuditInfo,
		Signers:                tables.Signers,
	})
	if createSchema {
		if err = initSchema(db, identityDB.GetSchema()); err != nil {
			return nil, err
		}
	}
	return identityDB, nil
}

func (db *IdentityDB) AddConfiguration(wp driver.IdentityConfiguration) error {
	query := fmt.Sprintf("INSERT INTO %s (identity_id, type, url) VALUES ($1, $2, $3)", db.table.IdentityConfigurations)
	logger.Debug(query)

	_, err := db.db.Exec(query, wp.ID, wp.Type, wp.URL)
	return err
}

func (db *IdentityDB) IteratorConfigurations(configurationType string) (driver.Iterator[driver.IdentityConfiguration], error) {
	query := fmt.Sprintf("SELECT identity_id, url FROM %s WHERE type = $1", db.table.IdentityConfigurations)
	logger.Debug(query)
	rows, err := db.db.Query(query, configurationType)
	if err != nil {
		return nil, err
	}
	return &WalletPathStorageIterator{rows: rows, configurationType: configurationType}, nil
}

func (db *IdentityDB) StoreAuditInfo(id, info []byte) error {
	query := fmt.Sprintf("INSERT INTO %s (identity_hash, identity, info) VALUES ($1, $2, $3)", db.table.AuditInfo)
	logger.Debug(query)

	h := view.Identity(id).String()
	_, err := db.db.Exec(query, h, id, info)
	if err != nil {
		// does the record already exists?
		auditInfo, err2 := db.GetAuditInfo(id)
		if err2 != nil {
			return err
		}
		if !bytes.Equal(auditInfo, info) {
			return errors.Wrapf(err, "different audit info stored for [%s], [%s]!=[%s]", id, hash.Hashable(auditInfo), hash.Hashable(info))
		}
		return nil
	}
	return err
}

func (db *IdentityDB) GetAuditInfo(id []byte) ([]byte, error) {
	h := view.Identity(id).String()
	query := fmt.Sprintf("SELECT info FROM %s WHERE identity_hash = $1", db.table.AuditInfo)
	logger.Debug(query)
	row := db.db.QueryRow(query, h)
	var info []byte
	err := row.Scan(&info)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "error querying db")
	}
	return info, nil
}

func (db *IdentityDB) StoreSignerInfo(id, info []byte) error {
	query := fmt.Sprintf("INSERT INTO %s (identity_hash, identity, info) VALUES ($1, $2, $3)", db.table.Signers)
	logger.Debug(query)

	h := view.Identity(id).String()
	_, err := db.db.Exec(query, h, id, info)
	return err
}

func (db *IdentityDB) SignerInfoExists(id []byte) (bool, error) {
	h := view.Identity(id).String()
	query := fmt.Sprintf("SELECT info FROM %s WHERE identity_hash = $1", db.table.Signers)
	logger.Debug(query)
	row := db.db.QueryRow(query, h)
	var info []byte
	err := row.Scan(&info)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, errors.Wrapf(err, "error querying db")
	}
	return true, nil
}

type WalletPathStorageIterator struct {
	rows              *sql.Rows
	configurationType string
}

func (w *WalletPathStorageIterator) Close() error {
	return w.rows.Close()
}

func (w *WalletPathStorageIterator) HasNext() bool {
	return w.rows.Next()
}

func (w *WalletPathStorageIterator) Next() (driver.IdentityConfiguration, error) {
	var c driver.IdentityConfiguration
	c.Type = w.configurationType
	err := w.rows.Scan(&c.ID, &c.URL)
	return c, err
}

func (db *IdentityDB) GetSchema() string {
	return fmt.Sprintf(`
		-- IdentityConfigurations
		CREATE TABLE IF NOT EXISTS %s (
			identity_id TEXT NOT NULL PRIMARY KEY,
            type TEXT NOT NULL,  
			url TEXT NOT NULL
		);
		-- AuditInfo
		CREATE TABLE IF NOT EXISTS %s (
            identity_hash TEXT NOT NULL PRIMARY KEY,
			identity BYTEA NOT NULL,
			info BYTEA NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_audits_%s ON %s ( identity_hash );

		-- Signers
		CREATE TABLE IF NOT EXISTS %s (
            identity_hash TEXT NOT NULL PRIMARY KEY,
			identity BYTEA NOT NULL,
			info BYTEA
		);
		CREATE INDEX IF NOT EXISTS idx_signers_%s ON %s ( identity_hash );
		`,
		db.table.IdentityConfigurations,
		db.table.AuditInfo,
		db.table.AuditInfo, db.table.AuditInfo,
		db.table.Signers,
		db.table.Signers, db.table.Signers,
	)
}
