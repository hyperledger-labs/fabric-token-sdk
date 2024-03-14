/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"bytes"
	"database/sql"
	"fmt"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type cache interface {
	Get(key string) (interface{}, bool)
	Add(key string, value interface{})
	Delete(key string)
}

type identityTables struct {
	IdentityConfigurations string
	IdentityInfo           string
	Signers                string
}

type IdentityDB struct {
	db    *sql.DB
	table identityTables

	singerInfoCacheMutex sync.RWMutex
	singerInfoCache      cache
}

func newIdentityDB(db *sql.DB, tables identityTables, singerInfoCache cache) *IdentityDB {
	return &IdentityDB{
		db:              db,
		table:           tables,
		singerInfoCache: singerInfoCache,
	}
}

func NewIdentityDB(db *sql.DB, tablePrefix string, createSchema bool, signerInfoCache cache) (*IdentityDB, error) {
	tables, err := getTableNames(tablePrefix)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table names")
	}

	identityDB := newIdentityDB(db, identityTables{
		IdentityConfigurations: tables.IdentityConfigurations,
		IdentityInfo:           tables.IdentityInfo,
		Signers:                tables.Signers,
	}, signerInfoCache)
	if createSchema {
		if err = initSchema(db, identityDB.GetSchema()); err != nil {
			return nil, err
		}
	}
	return identityDB, nil
}

func (db *IdentityDB) AddConfiguration(wp driver.IdentityConfiguration) error {
	query := fmt.Sprintf("INSERT INTO %s (id, type, url, conf, raw) VALUES ($1, $2, $3, $4, $5)", db.table.IdentityConfigurations)
	logger.Debug(query)

	_, err := db.db.Exec(query, wp.ID, wp.Type, wp.URL, wp.Config, wp.Raw)
	return err
}

func (db *IdentityDB) IteratorConfigurations(configurationType string) (driver.Iterator[driver.IdentityConfiguration], error) {
	query := fmt.Sprintf("SELECT id, url, conf, raw FROM %s WHERE type = $1", db.table.IdentityConfigurations)
	logger.Debug(query)
	rows, err := db.db.Query(query, configurationType)
	if err != nil {
		return nil, err
	}
	return &IdentityConfigurationIterator{rows: rows, configurationType: configurationType}, nil
}

func (db *IdentityDB) ConfigurationExists(id, typ string) (bool, error) {
	result, err := QueryUnique[string](db.db,
		fmt.Sprintf("SELECT id FROM %s WHERE id=$1 AND type=$2", db.table.IdentityConfigurations),
		id, typ,
	)
	if err != nil {
		return false, errors.Wrapf(err, "failed getting configuration for [%s:%s]", id, typ)
	}
	logger.Debugf("found configuration for [%s:%s]", id, typ)
	return len(result) != 0, nil
}

func (db *IdentityDB) StoreIdentityData(id []byte, identityAudit []byte, tokenMetadata []byte, tokenMetadataAudit []byte) error {
	query := fmt.Sprintf("INSERT INTO %s (identity_hash, identity, identityAuditInfo, tokenMetadata, tokenMetadataAuditInfo) VALUES ($1, $2, $3, $4, $5)", db.table.IdentityInfo)
	logger.Debug(query)

	h := view.Identity(id).String()
	_, err := db.db.Exec(query, h, id, identityAudit, tokenMetadata, tokenMetadataAudit)
	if err != nil {
		// does the record already exists?
		auditInfo, err2 := db.GetAuditInfo(id)
		if err2 != nil {
			return err
		}
		if !bytes.Equal(auditInfo, identityAudit) {
			return errors.Wrapf(err, "different audit info stored for [%s], [%s]!=[%s]", id, hash.Hashable(auditInfo), hash.Hashable(identityAudit))
		}
		return nil
	}
	return err
}

func (db *IdentityDB) GetAuditInfo(id []byte) ([]byte, error) {
	h := view.Identity(id).String()
	query := fmt.Sprintf("SELECT identityAuditInfo FROM %s WHERE identity_hash = $1", db.table.IdentityInfo)
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

func (db *IdentityDB) GetTokenInfo(id []byte) ([]byte, []byte, error) {
	h := view.Identity(id).String()
	query := fmt.Sprintf("SELECT tokenMetadata, tokenMetadataAuditInfo FROM %s WHERE identity_hash = $1", db.table.IdentityInfo)
	logger.Debug(query)
	row := db.db.QueryRow(query, h)
	var tokenMetadata []byte
	var tokenMetadataAuditInfo []byte
	err := row.Scan(&tokenMetadata, &tokenMetadataAuditInfo)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, errors.Wrapf(err, "error querying db")
	}
	return tokenMetadata, tokenMetadataAuditInfo, nil
}

func (db *IdentityDB) StoreSignerInfo(id, info []byte) error {
	query := fmt.Sprintf("INSERT INTO %s (identity_hash, identity, info) VALUES ($1, $2, $3)", db.table.Signers)
	h := view.Identity(id).String()
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("store signer info [%s]: [%s][%s]", query, h, hash.Hashable(info))
	}
	_, err := db.db.Exec(query, h, id, info)
	if err != nil {
		if exists, err2 := db.SignerInfoExists(id); err2 == nil && exists {
			logger.Debugf("signer info [%s] exists, no error to return", h)
		} else {
			return err
		}
	}

	db.singerInfoCacheMutex.Lock()
	db.singerInfoCache.Add(h, true)
	db.singerInfoCacheMutex.Unlock()
	return nil
}

func (db *IdentityDB) SignerInfoExists(id []byte) (bool, error) {
	h := view.Identity(id).String()

	// is in cache?
	db.singerInfoCacheMutex.RLock()
	v, ok := db.singerInfoCache.Get(h)
	if ok {
		db.singerInfoCacheMutex.RUnlock()
		return v != nil && v.(bool), nil
	}
	db.singerInfoCacheMutex.RUnlock()

	// get from store
	db.singerInfoCacheMutex.Lock()
	defer db.singerInfoCacheMutex.Unlock()

	// is in cache, first?
	v, ok = db.singerInfoCache.Get(h)
	if ok {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("hit the cache, len state [%b]", v.(bool))
		}
		return v != nil && v.(bool), nil
	}

	// get from store and store in cache
	exists, err := db.signerInfoExists(h)
	if err != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("failed getting state [%s]", h)
		}
		db.singerInfoCache.Delete(h)
		return false, err
	}
	db.singerInfoCache.Add(h, exists)
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("signer info [%s] exists [%v]", h, exists)
	}
	return exists, nil
}

func (db *IdentityDB) GetSignerInfo(identity []byte) ([]byte, error) {
	h := view.Identity(identity).String()
	query := fmt.Sprintf("SELECT info FROM %s WHERE identity_hash = $1", db.table.Signers)
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

func (db *IdentityDB) signerInfoExists(h string) (bool, error) {
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

type IdentityConfigurationIterator struct {
	rows              *sql.Rows
	configurationType string
}

func (w *IdentityConfigurationIterator) Close() error {
	return w.rows.Close()
}

func (w *IdentityConfigurationIterator) HasNext() bool {
	return w.rows.Next()
}

func (w *IdentityConfigurationIterator) Next() (driver.IdentityConfiguration, error) {
	var c driver.IdentityConfiguration
	c.Type = w.configurationType
	err := w.rows.Scan(&c.ID, &c.URL, &c.Config, &c.Raw)
	return c, err
}

func (db *IdentityDB) GetSchema() string {
	return fmt.Sprintf(`
		-- IdentityConfigurations
		CREATE TABLE IF NOT EXISTS %s (
			id TEXT NOT NULL,
            type TEXT NOT NULL,  
			url TEXT NOT NULL,
			conf BYTEA,
			raw BYTEA,
			PRIMARY KEY(id, type)
		);
		CREATE INDEX IF NOT EXISTS idx_ic_type_%s ON %s ( type );
		CREATE INDEX IF NOT EXISTS idx_ic_id_type_%s ON %s ( id, type );

		-- IdentityInfo
		CREATE TABLE IF NOT EXISTS %s (
            identity_hash TEXT NOT NULL PRIMARY KEY,
			identity BYTEA NOT NULL,
			identityAuditInfo BYTEA NOT NULL,
			tokenMetadata BYTEA,
			tokenMetadataAuditInfo BYTEA
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
		db.table.IdentityConfigurations, db.table.IdentityConfigurations,
		db.table.IdentityConfigurations, db.table.IdentityConfigurations,
		db.table.IdentityInfo,
		db.table.IdentityInfo, db.table.IdentityInfo,
		db.table.Signers,
		db.table.Signers, db.table.Signers,
	)
}
