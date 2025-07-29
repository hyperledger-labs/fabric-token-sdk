/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/cache/secondcache"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/hash"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/common"
	q "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query"
	common3 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/cond"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	tdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/pkg/errors"
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

type IdentityStore struct {
	readDB  *sql.DB
	writeDB *sql.DB
	table   identityTables
	ci      common3.CondInterpreter

	signerInfoCache cache[bool]
	auditInfoCache  cache[[]byte]
}

func newIdentityStore(readDB, writeDB *sql.DB, tables identityTables, singerInfoCache cache[bool], auditInfoCache cache[[]byte], ci common3.CondInterpreter) *IdentityStore {
	return &IdentityStore{
		readDB:          readDB,
		writeDB:         writeDB,
		table:           tables,
		signerInfoCache: singerInfoCache,
		auditInfoCache:  auditInfoCache,
		ci:              ci,
	}
}

func NewCachedIdentityStore(readDB, writeDB *sql.DB, tables TableNames, ci common3.CondInterpreter) (*IdentityStore, error) {
	return NewIdentityStore(
		readDB,
		writeDB,
		tables,
		secondcache.NewTyped[bool](1000),
		secondcache.NewTyped[[]byte](1000),
		ci,
	)
}

func NewIdentityStore(readDB, writeDB *sql.DB, tables TableNames, signerInfoCache cache[bool], auditInfoCache cache[[]byte], ci common3.CondInterpreter) (*IdentityStore, error) {
	return newIdentityStore(
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

func (db *IdentityStore) CreateSchema() error {
	return common.InitSchema(db.writeDB, []string{db.GetSchema()}...)
}

func (db *IdentityStore) AddConfiguration(ctx context.Context, wp driver.IdentityConfiguration) error {
	query, args := q.InsertInto(db.table.IdentityConfigurations).
		Fields("id", "type", "url", "conf", "raw").
		Row(wp.ID, wp.Type, wp.URL, wp.Config, wp.Raw).
		Format()
	logger.Debug(query, args)

	_, err := db.writeDB.ExecContext(ctx, query, args...)
	return err
}

func (db *IdentityStore) IteratorConfigurations(ctx context.Context, configurationType string) (driver3.IdentityConfigurationIterator, error) {
	query, args := q.Select().
		FieldsByName("id", "url", "conf", "raw").
		From(q.Table(db.table.IdentityConfigurations)).
		Where(cond.Eq("type", configurationType)).
		Format(db.ci)
	logger.Debug(query, args)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &IdentityConfigurationIterator{rows: rows, configurationType: configurationType}, nil
}

func (db *IdentityStore) ConfigurationExists(ctx context.Context, id, typ, url string) (bool, error) {
	query, args := q.Select().
		FieldsByName("id").
		From(q.Table(db.table.IdentityConfigurations)).
		Where(cond.And(cond.Eq("id", id), cond.Eq("type", typ), cond.Eq("url", url))).
		Format(db.ci)
	result, err := common.QueryUnique[string](db.readDB, query, args...)
	if err != nil {
		return false, errors.Wrapf(err, "failed getting configuration for [%s:%s:%s]", id, typ, url)
	}
	logger.DebugfContext(ctx, "found configuration for [%s:%s:%s]", id, typ, url)
	return len(result) != 0, nil
}

func (db *IdentityStore) StoreIdentityData(ctx context.Context, id []byte, identityAudit []byte, tokenMetadata []byte, tokenMetadataAudit []byte) error {
	logger.DebugfContext(ctx, "store identity data for [%s]", view.Identity(id))
	h := token.Identity(id).String()
	query, args := q.InsertInto(db.table.IdentityInfo).
		Fields("identity_hash", "identity", "identity_audit_info", "token_metadata", "token_metadata_audit_info").
		Row(h, id, identityAudit, tokenMetadata, tokenMetadataAudit).
		Format()
	logger.Debug(query, args)

	_, err := db.writeDB.ExecContext(ctx, query, args...)
	if err != nil {
		// does the record already exists?
		logger.DebugfContext(ctx, "store identity data failed, check if audit info exists")
		auditInfo, err2 := db.GetAuditInfo(ctx, id)
		if err2 != nil {
			return err
		}
		if !bytes.Equal(auditInfo, identityAudit) {
			return errors.Wrapf(err, "different audit info stored for [%s], [%s]!=[%s]", h, hash.Hashable(auditInfo), hash.Hashable(identityAudit))
		}
		logger.DebugfContext(ctx, "audit info exists")
		return nil
	}

	logger.DebugfContext(ctx, "audit info cache update")
	db.auditInfoCache.Add(h, identityAudit)
	logger.DebugfContext(ctx, "audit info cache update done")

	return err
}

func (db *IdentityStore) GetAuditInfo(ctx context.Context, id []byte) ([]byte, error) {
	h := token.Identity(id).String()
	logger.DebugfContext(ctx, "get audit info for [%s]", h)

	value, _, err := db.auditInfoCache.GetOrLoad(h, func() ([]byte, error) {
		logger.DebugfContext(ctx, "load from backend identity data for [%s]", view.Identity(id))
		query, args := q.Select().
			FieldsByName("identity_audit_info").
			From(q.Table(db.table.IdentityInfo)).
			Where(cond.Eq("identity_hash", h)).
			Format(db.ci)
		return common.QueryUnique[[]byte](db.readDB, query, args...)
	})
	return value, err
}

func (db *IdentityStore) GetTokenInfo(ctx context.Context, id []byte) ([]byte, []byte, error) {
	h := token.Identity(id).String()
	logger.DebugfContext(ctx, "get identity data for [%s]", h)

	query, args := q.Select().
		FieldsByName("token_metadata", "token_metadata_audit_info").
		From(q.Table(db.table.IdentityInfo)).
		Where(cond.Eq("identity_hash", h)).
		Format(db.ci)
	logger.Debug(query, args)

	row := db.readDB.QueryRowContext(ctx, query, args...)
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

func (db *IdentityStore) StoreSignerInfo(ctx context.Context, id, info []byte) error {
	h := token.Identity(id).String()

	logger.DebugfContext(ctx, "store signer info for [%s]", h)
	query, args := q.InsertInto(db.table.Signers).
		Fields("identity_hash", "identity", "info").
		Row(h, id, info).
		Format()

	logger.Debug(query, h, hash.Hashable(info))
	_, err := db.writeDB.ExecContext(ctx, query, args...)
	if err != nil {
		if exists, err2 := db.SignerInfoExists(ctx, id); err2 == nil && exists {
			logger.DebugfContext(ctx, "signer info [%s] exists, no error to return", h)
		} else {
			return err
		}
	}

	db.signerInfoCache.Add(h, true)

	logger.DebugfContext(ctx, "store signer info done")
	return nil
}

func (db *IdentityStore) GetExistingSignerInfo(ctx context.Context, ids ...tdriver.Identity) ([]string, error) {
	idHashes := make([]string, len(ids))
	for i, id := range ids {
		idHashes[i] = id.UniqueID()
	}

	result := make([]string, 0)
	notFound := make([]string, 0)

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
	notFound = make([]string, 0)
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

	query, args := q.Select().
		FieldsByName("identity_hash").
		From(q.Table(db.table.Signers)).
		Where(cond.In("identity_hash", idHashes...)).
		Format(db.ci)

	logger.Debug(query, args)
	rows, err := db.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "error querying db")
	}
	it := common.NewIterator(rows, func(idHash *string) error { return rows.Scan(idHash) })

	found, err := iterators.Reduce(it, iterators.ToSet[string]())
	if err != nil {
		return nil, err
	}
	for _, idHash := range idHashes {
		db.signerInfoCache.Add(idHash, found.Contains(idHash))
	}
	return append(result, found.ToSlice()...), nil
}

func (db *IdentityStore) SignerInfoExists(ctx context.Context, id []byte) (bool, error) {
	existing, err := db.GetExistingSignerInfo(ctx, id)
	if err != nil {
		return false, err
	}
	return len(existing) > 0, nil
}

func (db *IdentityStore) GetSignerInfo(ctx context.Context, identity []byte) ([]byte, error) {
	query, args := q.Select().
		FieldsByName("info").
		From(q.Table(db.table.Signers)).
		Where(cond.Eq("identity_hash", token.Identity(identity).UniqueID())).
		Format(db.ci)
	return common.QueryUnique[[]byte](db.readDB, query, args...)
}

func (db *IdentityStore) Close() error {
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

func (db *IdentityStore) GetSchema() string {
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
