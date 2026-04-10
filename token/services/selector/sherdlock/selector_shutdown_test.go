/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sherdlock

import (
	"context"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStubbornSelector_ContextCancellation verifies that when the caller's context
// is cancelled during backoff, Select returns ctx.Err() and unlocks all tokens.
func TestStubbornSelector_ContextCancellation(t *testing.T) {
	t.Run("returns ctx.Err on cancellation during backoff", func(t *testing.T) {
		unlockAllCalled := false

		mockFetcher := &mockTokenFetcher{
			unspentTokensIteratorByFunc: func(_ context.Context, _ string, _ token2.Type) (iterator[*token2.UnspentTokenInWallet], error) {
				tok := &token2.UnspentTokenInWallet{
					Id:       token2.ID{TxId: "tx1", Index: 0},
					Type:     "USD",
					Quantity: "100",
				}

				return collections.NewSliceIterator([]*token2.UnspentTokenInWallet{tok}), nil
			},
		}

		// TryLock always returns false: all tokens appear locked by others, triggering backoff.
		mockLck := &cancelTestLocker{
			tryLockResult:   false,
			unlockAllCalled: &unlockAllCalled,
		}

		m := NewMetrics(&disabled.Provider{})
		// Use a backoff interval far exceeding the context timeout so ctx.Done()
		// fires in the backoff select before time.After can.
		sel := NewStubbornSelector(
			logger,
			mockFetcher,
			mockLck,
			64,
			time.Hour,
			10,
			m,
		)

		// 50 ms is far shorter than time.Hour backoff — ctx.Done() fires first.
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()

		_, _, err := sel.Select(ctx, &ownerFilter{id: "wallet1"}, "100", "USD")

		require.Error(t, err)
		require.ErrorIs(t, err, context.DeadlineExceeded, "expected deadline exceeded, got: %v", err)
		assert.True(t, unlockAllCalled, "UnlockAll must be called on context cancellation")
	})
}

// cancelTestLocker is a locker where TryLock always returns false (simulating all
// tokens locked by others) and records whether UnlockAll was called.
type cancelTestLocker struct {
	tryLockResult   bool
	unlockAllCalled *bool
}

func (l *cancelTestLocker) TryLock(_ context.Context, _ *token2.ID) bool {
	return l.tryLockResult
}

func (l *cancelTestLocker) UnlockAll(_ context.Context) error {
	*l.unlockAllCalled = true

	return nil
}

type ownerFilter struct {
	id string
}

func (o *ownerFilter) ID() string { return o.id }
