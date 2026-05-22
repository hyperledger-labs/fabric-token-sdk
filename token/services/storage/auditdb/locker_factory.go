/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditdb

import (
	"database/sql"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

// WriteDBProvider is an optional interface that SQL-backed AuditTransactionStore
// implementations may satisfy to expose their underlying *sql.DB.
// The postgres driver's AuditTransactionStore already implements this.
type WriteDBProvider interface {
	WriteDB() *sql.DB
}

// EIDLeasesTableProvider is an optional interface that SQL-backed stores may
// satisfy to expose the formatted EID-leases table name.
type EIDLeasesTableProvider interface {
	EIDLeasesTable() string
}

// NewLockerFromConfig builds the appropriate Locker based on cfg.
// When the backend is "postgres", store must implement WriteDBProvider and
// EIDLeasesTableProvider so the locker can share the connection pool and use
// the correct table name.
func NewLockerFromConfig(cfg LockerConfig, store any) (Locker, error) {
	switch cfg.Backend {
	case LockerBackendPostgres:
		dbp, ok := store.(WriteDBProvider)
		if !ok {
			return nil, errors.New("postgres locker backend requires a SQL-backed audit store (WriteDBProvider)")
		}
		tp, ok := store.(EIDLeasesTableProvider)
		if !ok {
			return nil, errors.New("postgres locker backend requires a store that provides EIDLeasesTable()")
		}

		return NewPostgresLocker(dbp.WriteDB(), tp.EIDLeasesTable(), cfg.Postgres)
	case LockerBackendMemory, "":
		return newMemoryLocker(), nil
	default:
		return nil, errors.Errorf("unknown locker backend: %s", cfg.Backend)
	}
}
