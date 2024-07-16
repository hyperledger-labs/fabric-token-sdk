/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	sql2 "database/sql"
	"testing"
	"time"

	utils2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/testutils"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestSufficientTokensOneReplica(t *testing.T) {
	replicas, terminate := startManagers(t, 1, NoBackoff)
	defer terminate()
	testutils.TestSufficientTokensOneReplica(t, replicas[0])
}

func TestSufficientTokensBigDenominationsOneReplica(t *testing.T) {
	replicas, terminate := startManagers(t, 1, time.Second)
	defer terminate()
	testutils.TestSufficientTokensBigDenominationsOneReplica(t, replicas[0])
}

func TestSufficientTokensBigDenominationsManyReplicas(t *testing.T) {
	replicas, terminate := startManagers(t, 3, time.Second)
	defer terminate()
	testutils.TestSufficientTokensBigDenominationsManyReplicas(t, replicas)
}

func TestInsufficientTokensOneReplica(t *testing.T) {
	replicas, terminate := startManagers(t, 1, NoBackoff)
	defer terminate()
	testutils.TestInsufficientTokensOneReplica(t, replicas[0])
}

func TestSufficientTokensManyReplicas(t *testing.T) {
	replicas, terminate := startManagers(t, 20, NoBackoff)
	defer terminate()
	testutils.TestSufficientTokensManyReplicas(t, replicas)
}

func TestInsufficientTokensManyReplicas(t *testing.T) {
	replicas, terminate := startManagers(t, 10, 5*time.Second)
	defer terminate()
	testutils.TestInsufficientTokensManyReplicas(t, replicas)
}

// Set up

func startManagers(t *testing.T, number int, backoff time.Duration) ([]testutils.EnhancedManager, func()) {
	terminate, pgConnStr := sql.StartPostgresContainer(t)
	replicas := make([]testutils.EnhancedManager, number)

	for i := 0; i < number; i++ {
		replica, err := createManager(pgConnStr, backoff)
		assert.NoError(t, err)
		replicas[i] = replica
	}
	return replicas, terminate
}

func createManager(pgConnStr string, backoff time.Duration) (testutils.EnhancedManager, error) {
	lockDB, err := initDB("postgres", pgConnStr, "test", 10, sql.NewTokenLockDB)
	if err != nil {
		return nil, err
	}

	tokenDB, err := initDB("postgres", pgConnStr, "test", 10, sql.NewTokenDB)
	if err != nil {
		return nil, err
	}
	return testutils.NewEnhancedManager(NewManager(tokenDB, lockDB, newMetrics(&disabled.Provider{}), testutils.TokenQuantityPrecision, backoff), tokenDB), nil

}

func initDB[T any](driverName, dataSourceName, tablePrefix string, maxOpenConns int, constructor func(*sql2.DB, string, bool) (T, error)) (T, error) {
	d := sql.NewSQLDBOpener("", "")
	sqlDB, err := d.OpenSQLDB(driverName, dataSourceName, maxOpenConns, false)
	if err != nil {
		return utils2.Zero[T](), err
	}
	return constructor(sqlDB, tablePrefix, true)
}
