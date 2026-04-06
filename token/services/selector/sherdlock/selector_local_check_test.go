/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"context"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelector_SkipsLocallyLockedTokens(t *testing.T) {
	dup := &token2.UnspentTokenInWallet{
		Id:       token2.ID{TxId: "tx1", Index: 0},
		Type:     "USD",
		Quantity: "1",
	}

	mockFetcher := &mockTokenFetcher{
		unspentTokensIteratorByFunc: func(_ context.Context, _ string, _ token2.Type) (iterator[*token2.UnspentTokenInWallet], error) {
			return collections.NewSliceIterator([]*token2.UnspentTokenInWallet{dup, dup}), nil
		},
	}

	lockCounter := &countingLocker{}
	m := NewMetrics(&disabled.Provider{})
	sel := NewSelector(logger, mockFetcher, lockCounter, 64, m)

	_, _, err := sel.Select(context.Background(), &ownerFilter{id: "wallet1"}, "2", "USD")

	require.Error(t, err)
	assert.True(t, errors.Is(err, token.SelectorInsufficientFunds))
	assert.Equal(t, 1, lockCounter.tryLockCalls, "expected a single lock attempt for duplicated token")
}

type countingLocker struct {
	tryLockCalls int
}

func (l *countingLocker) TryLock(_ context.Context, _ *token2.ID) bool {
	l.tryLockCalls++

	return true
}

func (l *countingLocker) UnlockAll(_ context.Context) error {
	return nil
}
