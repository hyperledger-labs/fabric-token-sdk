/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package unity

import (
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	sql3 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/sqlite"
	"github.com/pkg/errors"
)

const (
	// optsKey is the key for the opts in the config
	optsKey   = "db.persistence.opts"
	envVarKey = "UNITYDB_DATASOURCE"

	UnityPersistence driver2.PersistenceType = "unity"
)

type constructors[D any] map[common2.SQLDriverType]common.NewDBFunc[D]

func newUnityDriver[V any, D db.DBDriver[V]](dbOpener *common.DBOpener, constructors constructors[V]) db.NamedDriver[D] {
	var d db.DBDriver[V] = &unityDriver[V]{
		dbOpener:     dbOpener,
		constructors: constructors,
	}
	return db.NamedDriver[D]{
		Name:   UnityPersistence,
		Driver: d.(D),
	}
}

type unityDriver[V any] struct {
	dbOpener     *common.DBOpener
	constructors constructors[V]
}

func (d *unityDriver[V]) Open(cp dbdriver.ConfigProvider, tmsID token.TMSID) (V, error) {
	sqlDB, opts, err := d.dbOpener.OpenWithOpts(cp, tmsID)
	if err != nil {
		return utils.Zero[V](), errors.Wrapf(err, "failed to open db at [%s:%s]", optsKey, envVarKey)
	}
	constructor, ok := d.constructors[opts.Driver]
	if !ok {
		return utils.Zero[V](), errors.New("constructor not found")
	}
	return constructor(sqlDB, common.NewDBOptsFromOpts(*opts))
}

type IdentityDBDriver struct {
	identityDriver *unityDriver[dbdriver.IdentityDB]
	walletDriver   *unityDriver[dbdriver.WalletDB]
}

func (t *IdentityDBDriver) OpenWalletDB(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.WalletDB, error) {
	return t.walletDriver.Open(cp, tmsID)
}

func (t *IdentityDBDriver) OpenIdentityDB(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.IdentityDB, error) {
	return t.identityDriver.Open(cp, tmsID)
}

func NewDBDrivers() (db.NamedDriver[dbdriver.TTXDBDriver], db.NamedDriver[dbdriver.TokenDBDriver], db.NamedDriver[dbdriver.TokenNotifierDriver], db.NamedDriver[dbdriver.TokenLockDBDriver], db.NamedDriver[dbdriver.AuditDBDriver], db.NamedDriver[dbdriver.IdentityDBDriver]) {
	root := common.NewSQLDBOpener(optsKey, envVarKey)

	return newUnityDriver[dbdriver.TokenTransactionDB, dbdriver.TTXDBDriver](root, constructors[dbdriver.TokenTransactionDB]{
			sql3.SQLite:   sqlite.NewTransactionDB,
			sql3.Postgres: postgres.NewTransactionDB,
		}),
		newUnityDriver[dbdriver.TokenDB, dbdriver.TokenDBDriver](root, constructors[dbdriver.TokenDB]{
			sql3.SQLite:   sqlite.NewTokenDB,
			sql3.Postgres: postgres.NewTokenDB,
		}),
		newUnityDriver[dbdriver.TokenNotifier, dbdriver.TokenNotifierDriver](root, constructors[dbdriver.TokenNotifier]{
			sql3.SQLite:   sqlite.NewTokenNotifier,
			sql3.Postgres: postgres.NewTokenNotifier,
		}),
		newUnityDriver[dbdriver.TokenLockDB, dbdriver.TokenLockDBDriver](root, constructors[dbdriver.TokenLockDB]{
			sql3.SQLite:   sqlite.NewTokenLockDB,
			sql3.Postgres: postgres.NewTokenLockDB,
		}),
		newUnityDriver[dbdriver.AuditTransactionDB, dbdriver.AuditDBDriver](root, constructors[dbdriver.AuditTransactionDB]{
			sql3.SQLite:   sqlite.NewAuditTransactionDB,
			sql3.Postgres: postgres.NewAuditTransactionDB,
		}),
		db.NamedDriver[dbdriver.IdentityDBDriver]{Name: UnityPersistence, Driver: &IdentityDBDriver{
			identityDriver: &unityDriver[dbdriver.IdentityDB]{
				dbOpener: root,
				constructors: constructors[dbdriver.IdentityDB]{
					sql3.SQLite:   sqlite.NewIdentityDB,
					sql3.Postgres: postgres.NewIdentityDB,
				},
			},
			walletDriver: &unityDriver[dbdriver.WalletDB]{
				dbOpener: root,
				constructors: constructors[dbdriver.WalletDB]{
					sql3.SQLite:   sqlite.NewWalletDB,
					sql3.Postgres: postgres.NewWalletDB,
				},
			},
		}}
}
