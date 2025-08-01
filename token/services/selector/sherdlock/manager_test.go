/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/multiplexed"
	postgres2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/dbtest"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/testutils"
	_ "github.com/jackc/pgx/v5/stdlib"
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
	replicas, terminate := startManagers(t, 3, 2*time.Second, 10)
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
	t.Helper()
	terminate, pgConnStr := startContainer(t)
	replicas := make([]testutils.EnhancedManager, number)

	for i := range number {
		replica, err := createManager(context.Background(), pgConnStr, backoff, maxRetries)
		assert.NoError(t, err)
		replicas[i] = replica
	}
	return replicas, terminate
}

func createManager(ctx context.Context, pgConnStr string, backoff time.Duration, maxRetries int) (testutils.EnhancedManager, error) {
	d := postgres.NewDriverWithDbProvider(multiplexed.MockTypeConfig(postgres2.Persistence, postgres2.Config{
		TablePrefix:  "test",
		DataSource:   pgConnStr,
		MaxOpenConns: 10,
	}), &dbProvider{})
	lockDB, err := d.NewTokenLock(ctx, "")
	if err != nil {
		return nil, err
	}
	tokenDB, err := d.NewToken(ctx, "")
	if err != nil {
		return nil, errors.Join(err, lockDB.Close())
	}

	fetcher := newMixedFetcher(tokenDB.(dbtest.TestTokenDB), newMetrics(&disabled.Provider{}))
	manager := NewManager(fetcher, lockDB, testutils.TokenQuantityPrecision, backoff, maxRetries, 0, 0)

	return testutils.NewEnhancedManager(manager, tokenDB.(dbtest.TestTokenDB)), nil
}

func startContainer(t *testing.T) (func(), string) {
	t.Helper()
	cfg := postgres2.DefaultConfig(t.Name())
	terminate, err := postgres2.StartPostgresWithFmt([]*postgres2.ContainerConfig{cfg})
	assert.NoError(t, err)
	return terminate, cfg.DataSource()
}

type dbProvider struct{}

func (p *dbProvider) Get(opts postgres2.Opts) (*common.RWDB, error) { return postgres2.Open(opts) }
