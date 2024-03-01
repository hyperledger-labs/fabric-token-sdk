/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"database/sql"
	"fmt"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
)

type identityTables struct {
	Identities string
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
		Identities: tables.Identities,
	})
	if createSchema {
		if err = initSchema(db, identityDB.GetSchema()); err != nil {
			return nil, err
		}
	}
	return identityDB, nil
}

func (db *IdentityDB) AddConfiguration(wp driver.IdentityConfiguration) error {
	query := fmt.Sprintf("INSERT INTO %s (identity_id, url) VALUES ($1, $2)", db.table.Identities)
	logger.Debug(query)

	_, err := db.db.Exec(query, wp.ID, wp.URL)
	return err
}

func (db *IdentityDB) IteratorConfigurations() (driver.Iterator[driver.IdentityConfiguration], error) {
	query := fmt.Sprintf("SELECT identity_id, url FROM %s", db.table.Identities)
	logger.Debug(query)
	rows, err := db.db.Query(query)
	if err != nil {
		return nil, err
	}
	return &WalletPathStorageIterator{rows: rows}, nil
}

type WalletPathStorageIterator struct {
	rows *sql.Rows
}

func (w *WalletPathStorageIterator) Close() error {
	return w.rows.Close()
}

func (w *WalletPathStorageIterator) HasNext() bool {
	return w.rows.Next()
}

func (w *WalletPathStorageIterator) Next() (driver.IdentityConfiguration, error) {
	var c driver.IdentityConfiguration
	err := w.rows.Scan(&c.ID, &c.URL)
	return c, err
}

func (db *IdentityDB) GetSchema() string {
	return fmt.Sprintf(`
		-- Identities
		CREATE TABLE IF NOT EXISTS %s (
			identity_id TEXT NOT NULL PRIMARY KEY,
			url TEXT NOT NULL
		);`,
		db.table.Identities,
	)
}
