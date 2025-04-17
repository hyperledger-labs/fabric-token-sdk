/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"bytes"
	"database/sql"
	"fmt"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/cache/secondcache"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	tdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type cache[T any] interface {
	Get(key string) (T, bool)
	GetOrLoad(key string, loader func() (T, error)) (T, bool, error)
	Add(key string, value T)
	Delete(key string)
}

type identityTables struct {
	IdentityConfigurations string
	IdentityInfo           string
	Signers                string
}

type IdentityDB struct {
	readDB  *sql.DB
	writeDB *sql.DB
	table   identityTables
	ci      common.Interpreter

	signerCacheLock sync.RWMutex
	signerInfoCache cache[bool]
	auditInfoCache  cache[[]byte]
}

func newIdentityDB(readDB, writeDB *sql.DB, tables identityTables, singerInfoCache cache[bool], auditInfoCache cache[[]byte], ci common.Interpreter) *IdentityDB {
	return &IdentityDB{
		readDB:          readDB,
		writeDB:         writeDB,
		table:           tables,
		signerInfoCache: singerInfoCache,
		auditInfoCache:  auditInfoCache,
		ci:              ci,
	}
}

func NewCachedIdentityDB(readDB, writeDB *sql.DB, tables tableNames, ci common.Interpreter) (*IdentityDB, error) {
	return NewIdentityDB(
		readDB,
		writeDB,
		tables,
		secondcache.NewTyped[bool](1000),
		secondcache.NewTyped[[]byte](1000),
		ci,
	)
}

func NewIdentityDB(readDB, writeDB *sql.DB, tables tableNames, signerInfoCache cache[bool], auditInfoCache cache[[]byte], ci common.Interpreter) (*IdentityDB, error) {
	return newIdentityDB(
		readDB,
		writeDB,
		identityTables{
			IdentityConfigurations: tables.IdentityConfigurations,
			IdentityInfo:           tables.IdentityInfo,
			Signers:                tables.Signers,
		},
		signerInfoCache,
		auditInfoCache,
		ci,
	), nil
}

func (db *IdentityDB) CreateSchema() error {
	return common.InitSchema(db.writeDB, []string{db.GetSchema()}...)
}

func (db *IdentityDB) AddConfiguration(wp driver.IdentityConfiguration) error {
	query, err := NewInsertInto(db.table.IdentityConfigurations).Rows("id, type, url, conf, raw").Compile()
	if err != nil {
		return errors.Wrapf(err, "failed compiling query")
	}
	logger.Debug(query, wp.ID, wp.Type, wp.URL, wp.Config, wp.Raw)

	_, err = db.writeDB.Exec(query, wp.ID, wp.Type, wp.URL, wp.Config, wp.Raw)
	return err
}

func (db *IdentityDB) IteratorConfigurations(configurationType string) (identity.ConfigurationIterator, error) {
	query, err := NewSelect("id, url, conf, raw").From(db.table.IdentityConfigurations).Where("type = $1").Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed compiling query")
	}
	logger.Debug(query)
	rows, err := db.readDB.Query(query, configurationType)
	if err != nil {
		return nil, err
	}
	return &IdentityConfigurationIterator{rows: rows, configurationType: configurationType}, nil
}

func (db *IdentityDB) ConfigurationExists(id, typ, url string) (bool, error) {
	query, err := NewSelect("id").From(db.table.IdentityConfigurations).Where("id=$1 AND type=$2 AND url=$3").Compile()
	if err != nil {
		return false, errors.Wrapf(err, "failed compiling query")
	}
	result, err := common.QueryUnique[string](db.readDB, query, id, typ, url)
	if err != nil {
		return false, errors.Wrapf(err, "failed getting configuration for [%s:%s:%s]", id, typ, url)
	}
	logger.Debugf("found configuration for [%s:%s:%s]", id, typ, url)
	return len(result) != 0, nil
}

func (db *IdentityDB) StoreIdentityData(id []byte, identityAudit []byte, tokenMetadata []byte, tokenMetadataAudit []byte) error {
	// logger.Infof("store identity data for [%s] from [%s]", view.Identity(id), string(debug.Stack()))
	query, err := NewInsertInto(db.table.IdentityInfo).Rows("identity_hash, identity, identity_audit_info, token_metadata, token_metadata_audit_info").Compile()
	if err != nil {
		return errors.Wrapf(err, "failed compiling query")
	}
	logger.Debug(query)

	h := token.Identity(id).String()
	_, err = db.writeDB.Exec(query, h, id, identityAudit, tokenMetadata, tokenMetadataAudit)
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
		// logger.Infof("get identity data for [%s] from [%s]", view.Identity(id), string(debug.Stack()))
		query, err := NewSelect("identity_audit_info").From(db.table.IdentityInfo).Where("identity_hash = $1").Compile()
		if err != nil {
			return nil, errors.Wrapf(err, "failed compiling query")
		}
		logger.Debug(query)
		row := db.readDB.QueryRow(query, h)
		var info []byte
		err = row.Scan(&info)
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
	// logger.Infof("get identity data for [%s] from [%s]", view.Identity(id), string(debug.Stack()))
	query, err := NewSelect("token_metadata, token_metadata_audit_info").From(db.table.IdentityInfo).Where("identity_hash = $1").Compile()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed compiling query")
	}
	logger.Debug(query)
	row := db.readDB.QueryRow(query, h)
	var tokenMetadata []byte
	var tokenMetadataAuditInfo []byte
	err = row.Scan(&tokenMetadata, &tokenMetadataAuditInfo)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, errors.Wrapf(err, "error querying db")
	}
	return tokenMetadata, tokenMetadataAuditInfo, nil
}

func (db *IdentityDB) StoreSignerInfo(id, info []byte) error {
	query, err := NewInsertInto(db.table.Signers).Rows("identity_hash, identity, info").Compile()
	if err != nil {
		return errors.Wrapf(err, "failed compiling query")
	}
	h := token.Identity(id).String()
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("store signer info [%s]: [%s][%s]", query, h, hash.Hashable(info))
	}
	_, err = db.writeDB.Exec(query, h, id, info)
	if err != nil {
		if exists, err2 := db.SignerInfoExists(id); err2 == nil && exists {
			logger.Debugf("signer info [%s] exists, no error to return", h)
		} else {
			return err
		}
	}

	db.signerCacheLock.Lock()
	defer db.signerCacheLock.Unlock()
	db.signerInfoCache.Add(h, true)
	return nil
}

func (db *IdentityDB) GetExistingSignerInfo(ids ...tdriver.Identity) ([]string, error) {
	idHashes := make([]string, len(ids))
	for i, id := range ids {
		idHashes[i] = id.UniqueID()
	}

	result := make([]string, 0)
	notFound := make([]string, 0)

	db.signerCacheLock.RLock()
	for _, idHash := range idHashes {
		if v, ok := db.signerInfoCache.Get(idHash); !ok {
			notFound = append(notFound, idHash)
		} else if v {
			result = append(result, idHash)
		}
	}
	if len(notFound) == 0 {
		defer db.signerCacheLock.RUnlock()
		return result, nil
	}
	db.signerCacheLock.RUnlock()

	idHashes = notFound
	notFound = make([]string, 0)
	db.signerCacheLock.Lock()
	defer db.signerCacheLock.Unlock()
	for _, idHash := range idHashes {
		if v, ok := db.signerInfoCache.Get(idHash); !ok {
			notFound = append(notFound, idHash)
		} else if v {
			result = append(result, idHash)
		}
	}
	if len(notFound) == 0 {
		return result, nil
	}

	idHashes = notFound
	condition := db.ci.InStrings("identity_hash", idHashes)
	ctr := 1
	query, err := NewSelect("identity_hash").From(db.table.Signers).Where(condition.ToString(&ctr)).Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed compiling query")
	}
	logger.Debug(query, condition.Params())
	rows, err := db.readDB.Query(query, condition.Params()...)
	if err != nil {
		return nil, errors.Wrapf(err, "error querying db")
	}
	defer Close(rows)
	found := collections.NewSet[string]()
	for rows.Next() {
		var idHash string
		if err := rows.Scan(&idHash); err != nil {
			return nil, err
		}
		found.Add(idHash)
	}
	for _, idHash := range idHashes {
		db.signerInfoCache.Add(idHash, found.Contains(idHash))
	}
	return append(result, found.ToSlice()...), nil
}

func (db *IdentityDB) SignerInfoExists(id []byte) (bool, error) {
	existing, err := db.GetExistingSignerInfo(id)
	if err != nil {
		return false, err
	}
	return len(existing) > 0, nil
}

func (db *IdentityDB) GetSignerInfo(identity []byte) ([]byte, error) {
	h := token.Identity(identity).String()
	query, err := NewSelect("info").From(db.table.Signers).Where("identity_hash = $1").Compile()
	if err != nil {
		return nil, errors.Wrapf(err, "failed compiling query")
	}
	logger.Debug(query)
	row := db.readDB.QueryRow(query, h)
	var info []byte
	err = row.Scan(&info)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "error querying db")
	}
	return info, nil
}

func (db *IdentityDB) Close() error {
	return common2.Close(db.readDB, db.writeDB)
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
			PRIMARY KEY(id, type, url)
		);
		CREATE INDEX IF NOT EXISTS idx_ic_type_%s ON %s ( type );
		CREATE INDEX IF NOT EXISTS idx_ic_id_type_%s ON %s ( id, type, url );

		-- IdentityInfo
		CREATE TABLE IF NOT EXISTS %s (
            identity_hash TEXT NOT NULL PRIMARY KEY,
			identity BYTEA NOT NULL,
			identity_audit_info BYTEA NOT NULL,
			token_metadata BYTEA,
			token_metadata_audit_info BYTEA
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
