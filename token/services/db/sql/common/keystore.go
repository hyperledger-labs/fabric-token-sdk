/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"database/sql"
	"encoding/json"
	"fmt"

	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/common"
	q "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query"
	common3 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/cond"
	"github.com/pkg/errors"
)

type keystoreTables struct {
	KeyStore string
}

type KeystoreStore struct {
	readDB  *sql.DB
	writeDB *sql.DB
	table   keystoreTables
	ci      common3.CondInterpreter
}

func newKeystoreStore(readDB, writeDB *sql.DB, tables keystoreTables, ci common3.CondInterpreter) *KeystoreStore {
	return &KeystoreStore{
		readDB:  readDB,
		writeDB: writeDB,
		table:   tables,
		ci:      ci,
	}
}

func NewKeystoreStore(readDB, writeDB *sql.DB, tables TableNames, ci common3.CondInterpreter) (*KeystoreStore, error) {
	return newKeystoreStore(
		readDB,
		writeDB,
		keystoreTables{
			KeyStore: tables.KeyStore,
		},
		ci,
	), nil
}

func (db *KeystoreStore) CreateSchema() error {
	return common.InitSchema(db.writeDB, []string{db.GetSchema()}...)
}

func (db *KeystoreStore) Close() error {
	return common2.Close(db.readDB, db.writeDB)
}

func (db *KeystoreStore) Put(id string, state interface{}) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return errors.Wrapf(err, "cannot marshal state with id [%s]", id)
	}
	query, args := q.InsertInto(db.table.KeyStore).
		Fields("key", "value").
		Row(id, raw).
		Format()
	logger.Debug(query, args)

	_, err = db.writeDB.Exec(query, args...)
	return err
}

func (db *KeystoreStore) Get(id string, state interface{}) error {
	query, args := q.Select().
		FieldsByName("value").
		From(q.Table(db.table.KeyStore)).
		Where(cond.Eq("key", id)).
		Format(db.ci)
	raw, err := common.QueryUnique[[]byte](db.readDB, query, args...)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, state); err != nil {
		return errors.Wrapf(err, "failed retrieving state [%s], cannot unmarshal state", id)
	}

	logger.Debugf("got state [%s,%s] successfully", id)
	return nil
}

func (db *KeystoreStore) GetSchema() string {
	return fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			key TEXT NOT NULL,
			val BYTEA NOT NULL,
			PRIMARY KEY (key)
		);
		`,
		db.table.KeyStore,
	)
}
