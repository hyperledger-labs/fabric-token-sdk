/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/sherdlock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/sherdlock/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetcherProviderUnit(t *testing.T) {
	mockSSM := &mocks.FakeTokenDBStoreServiceManager{}
	metricsProvider, _ := setupMetricsMocks()

	provider := sherdlock.NewFetcherProvider(mockSSM, metricsProvider, sherdlock.Mixed, 0, 0, 0)

	t.Run("GetFetcher_Error", func(t *testing.T) {
		mockSSM.StoreServiceByTMSIdReturns(nil, errors.New("ssm error"))
		_, err := provider.GetFetcher(token.TMSID{Network: "n1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ssm error")
	})
}
