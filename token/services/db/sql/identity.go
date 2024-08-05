/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"bytes"
	"database/sql"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/cache/secondcache"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type cache[T any] interface {
	Get(key string) (T, bool)
	GetOrLoad(key string, loader func() (T, error)) (T, bool, error)
	Add(key string, value T)
	Delete(key string)
}

type IdentityDB struct {
	db              *sql.DB
	signerInfoCache cache[bool]
	auditInfoCache  cache[[]byte]
}

func newIdentityDB(db *sql.DB, singerInfoCache cache[bool], auditInfoCache cache[[]byte]) *IdentityDB {
	return &IdentityDB{
		db:              db,
		signerInfoCache: singerInfoCache,
		auditInfoCache:  auditInfoCache,
	}
}

func NewCachedIdentityDB(db *sql.DB, createSchema bool) (driver.IdentityDB, error) {
	return NewIdentityDB(
		db,
		createSchema,
		secondcache.NewTyped[bool](1000),
		secondcache.NewTyped[[]byte](1000),
	)
}

func NewIdentityDB(db *sql.DB, createSchema bool, signerInfoCache cache[bool], auditInfoCache cache[[]byte]) (*IdentityDB, error) {
	identityDB := newIdentityDB(
		db,
		signerInfoCache,
		auditInfoCache,
	)
	if createSchema {
		if err := initSchema(db, identityDB.GetSchema()); err != nil {
			return nil, err
		}
	}
	return identityDB, nil
}

func (db *IdentityDB) AddConfiguration(wp driver.IdentityConfiguration) error {
	query := "INSERT INTO identity_configurations (id, type, url, conf, raw) VALUES ($1, $2, $3, $4, $5)"
	logger.Debug(query)

	_, err := db.db.Exec(query, wp.ID, wp.Type, wp.URL, wp.Config, wp.Raw)
	return err
}

func (db *IdentityDB) IteratorConfigurations(configurationType string) (driver.Iterator[driver.IdentityConfiguration], error) {
	query := "SELECT id, url, conf, raw FROM identity_configurations WHERE type = $1"
	logger.Debug(query)
	rows, err := db.db.Query(query, configurationType)
	if err != nil {
		return nil, err
	}
	return &IdentityConfigurationIterator{rows: rows, configurationType: configurationType}, nil
}

func (db *IdentityDB) ConfigurationExists(id, typ string) (bool, error) {
	result, err := QueryUnique[string](db.db,
		"SELECT id FROM identity_configurations WHERE id=$1 AND type=$2",
		id, typ,
	)
	if err != nil {
		return false, errors.Wrapf(err, "failed getting configuration for [%s:%s]", id, typ)
	}
	logger.Debugf("found configuration for [%s:%s]", id, typ)
	return len(result) != 0, nil
}

func (db *IdentityDB) StoreIdentityData(id []byte, identityAudit []byte, tokenMetadata []byte, tokenMetadataAudit []byte) error {
	//logger.Infof("store identity data for [%s] from [%s]", view.Identity(id), string(debug.Stack()))
	query := "INSERT INTO identity_information (identity_hash, identity, identity_audit_info, token_metadata, token_metadata_audit_info) VALUES ($1, $2, $3, $4, $5)"
	logger.Debug(query)

	h := token.Identity(id).String()
	_, err := db.db.Exec(query, h, id, identityAudit, tokenMetadata, tokenMetadataAudit)
	if err != nil {
		// does the record already exists?
		auditInfo, err2 := db.GetAuditInfo(id)
		if err2 != nil {
			return err
		}
		if !bytes.Equal(auditInfo, identityAudit) {
			return errors.Wrapf(err, "different audit info stored for [%s], [%s]!=[%s]", h, hash.Hashable(auditInfo), hash.Hashable(identityAudit))
		}
		return nil
	}

	db.auditInfoCache.Add(h, identityAudit)

	return err
}

func (db *IdentityDB) GetAuditInfo(id []byte) ([]byte, error) {
	h := token.Identity(id).String()

	value, _, err := db.auditInfoCache.GetOrLoad(h, func() ([]byte, error) {
		//logger.Infof("get identity data for [%s] from [%s]", view.Identity(id), string(debug.Stack()))
		query := "SELECT identity_audit_info FROM identity_information WHERE identity_hash = $1"
		logger.Debug(query)
		row := db.db.QueryRow(query, h)
		var info []byte
		err := row.Scan(&info)
		if err == nil {
			return info, nil
		}
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "error querying db")
	})
	return value, err
}

func (db *IdentityDB) GetTokenInfo(id []byte) ([]byte, []byte, error) {
	h := token.Identity(id).String()
	//logger.Infof("get identity data for [%s] from [%s]", view.Identity(id), string(debug.Stack()))
	query := "SELECT token_metadata, token_metadata_audit_info FROM identity_information WHERE identity_hash = $1"
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
	query := "INSERT INTO identity_signers (identity_hash, identity, info) VALUES ($1, $2, $3)"
	h := token.Identity(id).String()
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

	db.signerInfoCache.Add(h, true)
	return nil
}

func (db *IdentityDB) SignerInfoExists(id []byte) (bool, error) {
	h := token.Identity(id).String()

	value, _, err := db.signerInfoCache.GetOrLoad(h, func() (bool, error) {
		query := "SELECT info FROM identity_signers WHERE identity_hash = $1"
		logger.Debug(query)
		row := db.db.QueryRow(query, h)
		var info []byte
		err := row.Scan(&info)
		if err == nil {
			return true, nil
		}
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, errors.Wrapf(err, "error querying db")
	})
	return value, err
}

func (db *IdentityDB) GetSignerInfo(identity []byte) ([]byte, error) {
	h := token.Identity(identity).String()
	query := "SELECT info FROM identity_signers WHERE identity_hash = $1"
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
	return `
	-- IdentityConfigurations
	CREATE TABLE IF NOT EXISTS identity_configurations (
		id TEXT NOT NULL,
		type TEXT NOT NULL,  
		url TEXT NOT NULL,
		conf BYTEA,
		raw BYTEA,
		PRIMARY KEY(id, type)
	);
	CREATE INDEX IF NOT EXISTS idx_ic_type ON identity_configurations ( type );

	-- IdentityInfo
	CREATE TABLE IF NOT EXISTS identity_information (
		identity_hash TEXT NOT NULL PRIMARY KEY,
		identity BYTEA NOT NULL,
		identity_audit_info BYTEA NOT NULL,
		token_metadata BYTEA,
		token_metadata_audit_info BYTEA
	);

	-- Signers
	CREATE TABLE IF NOT EXISTS identity_signers (
		identity_hash TEXT NOT NULL PRIMARY KEY,
		identity BYTEA NOT NULL,
		info BYTEA
	);
	`
}
