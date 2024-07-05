/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	dbconfig "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/driver/unity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokenlockdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"go.uber.org/dig"
)

func NewTokenLockDBManager(in struct {
	dig.In
	ConfigService  driver2.ConfigService
	ConfigProvider *config2.Service
	Drivers        []db.NamedDriver[dbdriver.TokenLockDBDriver] `group:"tokenlockdb-drivers"`
}) *tokenlockdb.Manager {
	return tokenlockdb.NewHolder(in.Drivers).NewManager(in.ConfigService, dbconfig.NewConfig(in.ConfigProvider, "tokenlockdb.persistence.type", "db.persistence.type"))
}

func NewAuditDBManager(in struct {
	dig.In
	ConfigService  driver2.ConfigService
	ConfigProvider *config2.Service
	Drivers        []db.NamedDriver[dbdriver.AuditDBDriver] `group:"auditdb-drivers"`
}) *auditdb.Manager {
	return auditdb.NewHolder(in.Drivers).NewManager(in.ConfigService, dbconfig.NewConfig(in.ConfigProvider, "ttxdb.persistence.type", "db.persistence.type"))
}

func NewTokenDBManager(in struct {
	dig.In
	ConfigService  driver2.ConfigService
	ConfigProvider *config2.Service
	Drivers        []db.NamedDriver[dbdriver.TokenDBDriver] `group:"tokendb-drivers"`
}) *tokendb.Manager {
	return tokendb.NewHolder(in.Drivers).NewManager(in.ConfigService, dbconfig.NewConfig(in.ConfigProvider, "tokendb.persistence.type", "db.persistence.type"))
}

func NewTTXDBManager(in struct {
	dig.In
	ConfigService  driver2.ConfigService
	ConfigProvider *config2.Service
	Drivers        []db.NamedDriver[dbdriver.TTXDBDriver] `group:"ttxdb-drivers"`
}) *ttxdb.Manager {
	return ttxdb.NewHolder(in.Drivers).NewManager(in.ConfigService, dbconfig.NewConfig(in.ConfigProvider, "ttxdb.persistence.type", "db.persistence.type"))
}

func NewIdentityDBManager(in struct {
	dig.In
	ConfigService  driver2.ConfigService
	ConfigProvider *config2.Service
	Drivers        []db.NamedDriver[dbdriver.IdentityDBDriver] `group:"identitydb-drivers"`
}) *identitydb.Manager {
	return identitydb.NewManager(in.Drivers, in.ConfigService, dbconfig.NewConfig(in.ConfigProvider, "identitydb.persistence.type", "db.persistence.type"))
}

type DBDriverResult struct {
	dig.Out
	TTXDBDriver       db.NamedDriver[dbdriver.TTXDBDriver]       `group:"ttxdb-drivers"`
	TokenDBDriver     db.NamedDriver[dbdriver.TokenDBDriver]     `group:"tokendb-drivers"`
	TokenLockDBDriver db.NamedDriver[dbdriver.TokenLockDBDriver] `group:"tokenlockdb-drivers"`
	AuditDBDriver     db.NamedDriver[dbdriver.AuditDBDriver]     `group:"auditdb-drivers"`
	IdentityDBDriver  db.NamedDriver[dbdriver.IdentityDBDriver]  `group:"identitydb-drivers"`
}

func NewDBDrivers() DBDriverResult {
	ttxDBDriver, tokenDBDriver, tokenLockDBDriver, auditDBDriver, identityDBDriver := unity.NewDBDrivers()
	return DBDriverResult{
		TTXDBDriver:       ttxDBDriver,
		TokenDBDriver:     tokenDBDriver,
		TokenLockDBDriver: tokenLockDBDriver,
		AuditDBDriver:     auditDBDriver,
		IdentityDBDriver:  identityDBDriver,
	}
}

func newTokenDriverService(in struct {
	dig.In
	Drivers []driver.NamedDriver `group:"token-drivers"`
}) *driver.TokenDriverService {
	return driver.NewTokenDriverService(in.Drivers)
}
