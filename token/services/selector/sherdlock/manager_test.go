/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"testing"
	"time"

	postgres2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/testutils"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestSufficientTokensOneReplica(t *testing.T) {
	replicas, terminate := startManagers(t, 1, NoBackoff, 5)
	defer terminate()
	testutils.TestSufficientTokensOneReplica(t, replicas[0])
}

func TestSufficientTokensOneReplicaNoRetry(t *testing.T) {
	replicas, terminate := startManagers(t, 1, NoBackoff, 0)
	defer terminate()
	testutils.TestSufficientTokensOneReplica(t, replicas[0])
}

func TestSufficientTokensBigDenominationsOneReplica(t *testing.T) {
	replicas, terminate := startManagers(t, 1, time.Second, 5)
	defer terminate()
	testutils.TestSufficientTokensBigDenominationsOneReplica(t, replicas[0])
}

func TestSufficientTokensBigDenominationsManyReplicas(t *testing.T) {
	replicas, terminate := startManagers(t, 3, time.Second, 5)
	defer terminate()
	testutils.TestSufficientTokensBigDenominationsManyReplicas(t, replicas)
}

func TestInsufficientTokensOneReplica(t *testing.T) {
	replicas, terminate := startManagers(t, 1, NoBackoff, 5)
	defer terminate()
	testutils.TestInsufficientTokensOneReplica(t, replicas[0])
}

func TestSufficientTokensManyReplicas(t *testing.T) {
	replicas, terminate := startManagers(t, 20, NoBackoff, 5)
	defer terminate()
	testutils.TestSufficientTokensManyReplicas(t, replicas)
}

func TestInsufficientTokensManyReplicas(t *testing.T) {
	replicas, terminate := startManagers(t, 10, 5*time.Second, 5)
	defer terminate()
	testutils.TestInsufficientTokensManyReplicas(t, replicas)
}

// Set up

func startManagers(t *testing.T, number int, backoff time.Duration, maxRetries int) ([]testutils.EnhancedManager, func()) {
	terminate, pgConnStr := common2.StartPostgresContainer(t)
	replicas := make([]testutils.EnhancedManager, number)

	for i := 0; i < number; i++ {
		replica, err := createManager(pgConnStr, backoff, maxRetries)
		assert.NoError(t, err)
		replicas[i] = replica
	}
	return replicas, terminate
}

func createManager(pgConnStr string, backoff time.Duration, maxRetries int) (testutils.EnhancedManager, error) {
	db, err := postgres2.OpenDB(pgConnStr, 10)
	if err != nil {
		return nil, err
	}
	opts := common2.NewDBOpts{
		DataSource:   pgConnStr,
		TablePrefix:  "test",
		CreateSchema: true,
	}

	lockDB, err := postgres.NewTokenLockDB(db, opts)
	if err != nil {
		return nil, err
	}
	tokenDB, err := postgres.NewTokenDB(db, opts)
	if err != nil {
		return nil, err
	}
	return testutils.NewEnhancedManager(NewManager(newMixedFetcher(tokenDB, newMetrics(&disabled.Provider{})), lockDB, testutils.TokenQuantityPrecision, backoff, maxRetries, 0), tokenDB), nil
}
