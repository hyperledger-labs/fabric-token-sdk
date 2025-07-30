/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"database/sql"
	"encoding/json"
	"fmt"

	dcommon "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/common"
	q "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query"
	qcommon "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/common"
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
	ci      qcommon.CondInterpreter
}

func newKeystoreStore(readDB, writeDB *sql.DB, tables keystoreTables, ci qcommon.CondInterpreter) *KeystoreStore {
	return &KeystoreStore{
		readDB:  readDB,
		writeDB: writeDB,
		table:   tables,
		ci:      ci,
	}
}

func NewKeystoreStore(readDB, writeDB *sql.DB, tables TableNames, ci qcommon.CondInterpreter) (*KeystoreStore, error) {
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
	return dcommon.Close(db.readDB, db.writeDB)
}

func (db *KeystoreStore) Put(key string, state interface{}) error {
	if state == nil {
		return errors.New("cannot store nil state")
	}
	if len(key) == 0 {
		return errors.New("cannot store empty key")
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return errors.Wrapf(err, "cannot marshal state with key [%s]", key)
	}
	query, args := q.InsertInto(db.table.KeyStore).
		Fields("key", "val").
		Row(key, raw).
		Format()
	logger.Debug(query, args)

	_, err = db.writeDB.Exec(query, args...)
	return err
}

func (db *KeystoreStore) Get(key string, state interface{}) error {
	query, args := q.Select().
		FieldsByName("val").
		From(q.Table(db.table.KeyStore)).
		Where(cond.Eq("key", key)).
		Format(db.ci)
	raw, err := common.QueryUnique[[]byte](db.readDB, query, args...)
	if err != nil {
		return errors.Wrapf(err, "failed retrieving key [%s]", key)
	}
	if len(raw) == 0 {
		return errors.Errorf("key [%s] does not exist", key)
	}
	if err := json.Unmarshal(raw, state); err != nil {
		return errors.Wrapf(err, "failed retrieving key [%s], cannot unmarshal state", key)
	}

	logger.Debugf("got key [%s] successfully", key)
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
