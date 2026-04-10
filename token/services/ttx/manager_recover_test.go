/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"context"
	"errors"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- stub types ---

type stubExistenceChecker struct {
	exists bool
	err    error
}

func (s *stubExistenceChecker) TransactionExists(_ context.Context, _ string) (bool, error) {
	return s.exists, s.err
}

type stubStatusSetter struct {
	calls     []setStatusCall
	returnErr error
}

type setStatusCall struct {
	txID    string
	status  storage.TxStatus
	message string
}

func (s *stubStatusSetter) SetStatus(_ context.Context, txID string, status storage.TxStatus, message string) error {
	s.calls = append(s.calls, setStatusCall{txID: txID, status: status, message: message})
	return s.returnErr
}

// --- tests ---

// TestRecoverCommittedPending_TokensAlreadyCommitted is the primary regression test.
//
// Scenario: node crashed after tokens.Append wrote to tokenDB but before
// ttxDB.SetStatus(Confirmed) ran.  On restart, recoverCommittedPending must
// detect that tokenDB already has the txID and call SetStatus(Confirmed)
// directly — without relying on block re-delivery.
func TestRecoverCommittedPending_TokensAlreadyCommitted(t *testing.T) {
	checker := &stubExistenceChecker{exists: true}
	setter := &stubStatusSetter{}

	recovered := recoverCommittedPending(context.Background(), "tx-abc", checker, setter)

	require.True(t, recovered, "should report recovery when tokens are already committed")
	require.Len(t, setter.calls, 1)
	assert.Equal(t, "tx-abc", setter.calls[0].txID)
	assert.Equal(t, storage.Confirmed, setter.calls[0].status)
	assert.NotEmpty(t, setter.calls[0].message)
}

// TestRecoverCommittedPending_TokensNotYetCommitted covers the normal restart
// path: the node restarted before tokens.Append ran, so recovery must NOT set
// Confirmed — the finality listener should handle it instead.
func TestRecoverCommittedPending_TokensNotYetCommitted(t *testing.T) {
	checker := &stubExistenceChecker{exists: false}
	setter := &stubStatusSetter{}

	recovered := recoverCommittedPending(context.Background(), "tx-xyz", checker, setter)

	assert.False(t, recovered, "should not recover when tokens are not yet in tokenDB")
	assert.Empty(t, setter.calls, "SetStatus must not be called when tokens are absent")
}

// TestRecoverCommittedPending_ExistenceCheckError confirms that a transient
// storage error on TransactionExists causes a graceful fallback (returns false)
// rather than a panic or incorrect recovery.
func TestRecoverCommittedPending_ExistenceCheckError(t *testing.T) {
	checker := &stubExistenceChecker{err: errors.New("db unavailable")}
	setter := &stubStatusSetter{}

	recovered := recoverCommittedPending(context.Background(), "tx-err", checker, setter)

	assert.False(t, recovered, "error in existence check must not be treated as recovery")
	assert.Empty(t, setter.calls)
}

// TestRecoverCommittedPending_SetStatusError verifies that when SetStatus
// fails the function returns false so the caller falls back to registering a
// finality listener — preventing silent data loss.
func TestRecoverCommittedPending_SetStatusError(t *testing.T) {
	checker := &stubExistenceChecker{exists: true}
	setter := &stubStatusSetter{returnErr: errors.New("write failed")}

	recovered := recoverCommittedPending(context.Background(), "tx-fail", checker, setter)

	assert.False(t, recovered, "SetStatus failure must cause fallback, not silent recovery")
	require.Len(t, setter.calls, 1, "SetStatus should have been attempted once")
}

// TestRecoverCommittedPending_Idempotent confirms that calling recoverCommittedPending
// twice for the same txID is safe: the second call hits SetStatus again
// (which is a no-op SQL UPDATE in production) and still returns true.
func TestRecoverCommittedPending_Idempotent(t *testing.T) {
	checker := &stubExistenceChecker{exists: true}
	setter := &stubStatusSetter{}

	first := recoverCommittedPending(context.Background(), "tx-dup", checker, setter)
	second := recoverCommittedPending(context.Background(), "tx-dup", checker, setter)

	assert.True(t, first)
	assert.True(t, second)
	assert.Len(t, setter.calls, 2, "both calls should attempt SetStatus (SQL UPDATE is idempotent)")
}
