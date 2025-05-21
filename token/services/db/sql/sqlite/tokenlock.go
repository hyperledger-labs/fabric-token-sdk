/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"time"

	common3 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/common"
	q "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/query"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/query/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

type TokenLockStore struct {
	*common.TokenLockStore
	ci common2.CondInterpreter
}

func IsStale(tokenLocks common2.TableName, requests common2.TableName, leaseExpiry time.Duration) *isStale {
	return &isStale{tokenLocks: tokenLocks, requests: requests, leaseExpiry: leaseExpiry}
}

type isStale struct {
	tokenLocks  common2.TableName
	requests    common2.TableName
	leaseExpiry time.Duration
}

func (c *isStale) WriteString(_ common2.CondInterpreter, sb common2.Builder) {
	sb.WriteString("tx_id IN (SELECT ").
		WriteString(string(c.tokenLocks)).
		WriteString(".tx_id FROM ").
		WriteString(string(c.tokenLocks)).
		WriteString(" JOIN ").
		WriteString(string(c.requests)).
		WriteString(" ON ").
		WriteString(string(c.tokenLocks)).
		WriteString(".tx_id = ").
		WriteString(string(c.requests)).
		WriteString(".tx_id WHERE ").
		WriteString(string(c.requests)).
		WriteString(".status IN (").
		WriteParam(driver.Deleted).
		WriteString(") OR ").
		WriteString(string(c.tokenLocks)).
		WriteString(".created_at < datetime('now', '-").
		WriteParam(c.leaseExpiry.Seconds()).
		WriteString(" seconds')")
}

func (db *TokenLockStore) Cleanup(leaseExpiry time.Duration) error {
	query, params := q.DeleteFrom(db.Table.TokenLocks).
		Where(IsStale(common2.TableName(db.Table.TokenLocks), common2.TableName(db.Table.Requests), leaseExpiry)).
		Format(db.ci)

	db.Logger.Debug(query, params)
	_, err := db.WriteDB.Exec(query, params...)
	if err != nil {
		db.Logger.Errorf("query failed: %s", query)
	}
	return err
}

func NewTokenLockStore(dbs *common3.RWDB, tableNames common.TableNames) (*TokenLockStore, error) {
	tldb, err := common.NewTokenLockStore(dbs.ReadDB, dbs.WriteDB, tableNames, sqlite.NewConditionInterpreter())
	if err != nil {
		return nil, err
	}
	return &TokenLockStore{TokenLockStore: tldb, ci: sqlite.NewConditionInterpreter()}, nil
}
