/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	"errors"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// waitForAnswers — covers every branch of the extracted helper
// ---------------------------------------------------------------------------

func TestWaitForAnswers_AllSucceed(t *testing.T) {
	ch := make(chan *answer, 2)
	ch <- &answer{response: &SpendResponse{}, party: view.Identity("p1")}
	ch <- &answer{response: &SpendResponse{}, party: view.Identity("p2")}

	err := waitForAnswers(ch, 2, time.Second)
	require.NoError(t, err)
}

func TestWaitForAnswers_ZeroCount(t *testing.T) {
	ch := make(chan *answer)
	err := waitForAnswers(ch, 0, time.Second)
	require.NoError(t, err)
}

func TestWaitForAnswers_TransportError(t *testing.T) {
	ch := make(chan *answer, 1)
	ch <- &answer{err: errors.New("session dropped"), party: view.Identity("p1")}

	err := waitForAnswers(ch, 1, time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session dropped")
}

func TestWaitForAnswers_ApplicationError(t *testing.T) {
	ch := make(chan *answer, 1)
	ch <- &answer{
		response: &SpendResponse{Err: errors.New("signature refused")},
		party:    view.Identity("p1"),
	}

	err := waitForAnswers(ch, 1, time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature refused")
}

func TestWaitForAnswers_Timeout(t *testing.T) {
	ch := make(chan *answer)

	start := time.Now()
	err := waitForAnswers(ch, 1, 50*time.Millisecond)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
	assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond)
}

func TestWaitForAnswers_ErrorOnSecondAnswer(t *testing.T) {
	ch := make(chan *answer, 2)
	ch <- &answer{response: &SpendResponse{}, party: view.Identity("p1")}
	ch <- &answer{err: errors.New("second party failed"), party: view.Identity("p2")}

	err := waitForAnswers(ch, 2, time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "second party failed")
}

// ---------------------------------------------------------------------------
// RequestSpendView struct — constructor and WithTimeout
// ---------------------------------------------------------------------------

func TestNewRequestSpendView_NilToken(t *testing.T) {
	v := NewRequestSpendView(nil)
	require.NotNil(t, v)
	assert.Error(t, v.err)
}

func TestRequestSpendView_DefaultTimeout(t *testing.T) {
	v := &RequestSpendView{timeout: defaultSpendRequestTimeout}
	assert.Equal(t, defaultSpendRequestTimeout, v.timeout)
}

func TestRequestSpendView_WithTimeout(t *testing.T) {
	v := &RequestSpendView{timeout: defaultSpendRequestTimeout}
	v.WithTimeout(10 * time.Second)
	assert.Equal(t, 10*time.Second, v.timeout)
}
