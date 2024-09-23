/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	sql2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	dbconfig "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/driver/unity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	tokensql "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokenlockdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"go.uber.org/dig"
)

// Token Lock DB

func NewTokenLockDBManager(in struct {
	dig.In
	ConfigService  driver2.ConfigService
	ConfigProvider *config2.Service
	Drivers        []db.NamedDriver[dbdriver.TokenLockDBDriver] `group:"tokenlockdb-drivers"`
}) *tokenlockdb.Manager {
	return tokenlockdb.NewHolder(in.Drivers).NewManager(in.ConfigService, dbconfig.NewConfig(in.ConfigProvider, "tokenlockdb.persistence.type", "db.persistence.type"))
}

// Audit DB

func NewAuditDBManager(in struct {
	dig.In
	ConfigService  driver2.ConfigService
	ConfigProvider *config2.Service
	Drivers        []db.NamedDriver[dbdriver.AuditDBDriver] `group:"auditdb-drivers"`
}) *auditdb.Manager {
	return auditdb.NewHolder(in.Drivers).NewManager(in.ConfigService, dbconfig.NewConfig(in.ConfigProvider, "auditdb.persistence.type", "db.persistence.type"))
}

// Transaction DB

func NewTTXDBManager(in struct {
	dig.In
	ConfigService  driver2.ConfigService
	ConfigProvider *config2.Service
	Drivers        []db.NamedDriver[dbdriver.TTXDBDriver] `group:"ttxdb-drivers"`
}) *ttxdb.Manager {
	return ttxdb.NewHolder(in.Drivers).NewManager(in.ConfigService, dbconfig.NewConfig(in.ConfigProvider, "ttxdb.persistence.type", "db.persistence.type"))
}

// Identity DB

func NewIdentityDBManager(in struct {
	dig.In
	ConfigService  driver2.ConfigService
	ConfigProvider *config2.Service
	Drivers        []db.NamedDriver[dbdriver.IdentityDBDriver] `group:"identitydb-drivers"`
}) *identitydb.Manager {
	return identitydb.NewManager(in.Drivers, in.ConfigService, dbconfig.NewConfig(in.ConfigProvider, "identitydb.persistence.type", "db.persistence.type"))
}

// Token DB

type TokenDriverResult struct {
	dig.Out
	DBDriver       db.NamedDriver[dbdriver.TokenDBDriver]       `group:"tokendb-drivers"`
	NotifierDriver db.NamedDriver[dbdriver.TokenNotifierDriver] `group:"tokennotifier-drivers"`
}

func NewTokenDrivers() TokenDriverResult {
	return TokenDriverResult{
		DBDriver:       db.NamedDriver[dbdriver.TokenDBDriver]{Name: sql2.SQLPersistence, Driver: tokensql.NewDBDriver()},
		NotifierDriver: db.NamedDriver[dbdriver.TokenNotifierDriver]{Name: sql2.SQLPersistence, Driver: tokensql.NewNotifierDriver()},
	}
}

func NewTokenManagers(in struct {
	dig.In
	ConfigService   driver2.ConfigService
	ConfigProvider  *config2.Service
	DBDrivers       []db.NamedDriver[dbdriver.TokenDBDriver]       `group:"tokendb-drivers"`
	NotifierDrivers []db.NamedDriver[dbdriver.TokenNotifierDriver] `group:"tokennotifier-drivers"`
}) (*tokendb.Manager, *tokendb.NotifierManager) {
	dbConfig := dbconfig.NewConfig(in.ConfigProvider, "tokendb.persistence.type", "db.persistence.type")
	return tokendb.NewHolder(in.DBDrivers).NewManager(in.ConfigService, dbConfig),
		tokendb.NewNotifierHolder(in.NotifierDrivers).NewManager(in.ConfigService, dbConfig)
}

// Unity

type DBDriverResult struct {
	dig.Out
	TTXDBDriver         db.NamedDriver[dbdriver.TTXDBDriver]         `group:"ttxdb-drivers"`
	TokenDBDriver       db.NamedDriver[dbdriver.TokenDBDriver]       `group:"tokendb-drivers"`
	TokenNotifierDriver db.NamedDriver[dbdriver.TokenNotifierDriver] `group:"tokennotifier-drivers"`
	TokenLockDBDriver   db.NamedDriver[dbdriver.TokenLockDBDriver]   `group:"tokenlockdb-drivers"`
	AuditDBDriver       db.NamedDriver[dbdriver.AuditDBDriver]       `group:"auditdb-drivers"`
	IdentityDBDriver    db.NamedDriver[dbdriver.IdentityDBDriver]    `group:"identitydb-drivers"`
}

func NewDBDrivers() DBDriverResult {
	ttxDBDriver, tokenDBDriver, tokenNotifierDriver, tokenLockDBDriver, auditDBDriver, identityDBDriver := unity.NewDBDrivers()
	return DBDriverResult{
		TTXDBDriver:         ttxDBDriver,
		TokenDBDriver:       tokenDBDriver,
		TokenNotifierDriver: tokenNotifierDriver,
		TokenLockDBDriver:   tokenLockDBDriver,
		AuditDBDriver:       auditDBDriver,
		IdentityDBDriver:    identityDBDriver,
	}
}

// Token SDK

func newTokenDriverService(in struct {
	dig.In
	Drivers []driver.NamedFactory[driver.Driver] `group:"token-drivers"`
}) *driver.TokenDriverService {
	return driver.NewTokenDriverService(in.Drivers)
}
