/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nfttx

import (
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCollectEndorsementsView(t *testing.T) {
	// Create a mock transaction
	mockTTX := &ttx.Transaction{}
	tx := &Transaction{Transaction: mockTTX}

	view := NewCollectEndorsementsView(tx)

	require.NotNil(t, view)
	// The view should be the result of ttx.NewCollectEndorsementsView
	assert.IsType(t, ttx.NewCollectEndorsementsView(mockTTX), view)
}

func TestNewOrderingAndFinalityView(t *testing.T) {
	mockTTX := &ttx.Transaction{}
	tx := &Transaction{Transaction: mockTTX}

	view := NewOrderingAndFinalityView(tx)

	require.NotNil(t, view)
	assert.IsType(t, ttx.NewOrderingAndFinalityView(mockTTX), view)
}

func TestNewOrderingAndFinalityWithTimeoutView(t *testing.T) {
	mockTTX := &ttx.Transaction{}
	tx := &Transaction{Transaction: mockTTX}
	timeout := 30 * time.Second

	view := NewOrderingAndFinalityWithTimeoutView(tx, timeout)

	require.NotNil(t, view)
	assert.IsType(t, ttx.NewOrderingAndFinalityWithTimeoutView(mockTTX, timeout), view)
}

func TestNewOrderingAndFinalityWithTimeoutView_DifferentTimeouts(t *testing.T) {
	mockTTX := &ttx.Transaction{}
	tx := &Transaction{Transaction: mockTTX}

	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{"zero timeout", 0},
		{"1 second", 1 * time.Second},
		{"30 seconds", 30 * time.Second},
		{"1 minute", 1 * time.Minute},
		{"5 minutes", 5 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewOrderingAndFinalityWithTimeoutView(tx, tt.timeout)
			require.NotNil(t, view)
		})
	}
}

func TestNewFinalityView(t *testing.T) {
	mockTTX := &ttx.Transaction{}
	tx := &Transaction{Transaction: mockTTX}

	t.Run("without options", func(t *testing.T) {
		view := NewFinalityView(tx)
		require.NotNil(t, view)
		assert.IsType(t, ttx.NewFinalityView(mockTTX), view)
	})

	t.Run("with options", func(t *testing.T) {
		// Create a dummy option
		opt := func(o *ttx.TxOptions) error {
			return nil
		}

		view := NewFinalityView(tx, opt)
		require.NotNil(t, view)
	})
}

func TestNewAcceptView(t *testing.T) {
	mockTTX := &ttx.Transaction{}
	tx := &Transaction{Transaction: mockTTX}

	view := NewAcceptView(tx)

	require.NotNil(t, view)
	assert.IsType(t, ttx.NewAcceptView(mockTTX), view)
}

func TestViewFunctions_NilTransaction(t *testing.T) {
	// Test that view functions handle nil transaction gracefully
	// Note: These will likely panic in real usage, but we're testing the wrapper behavior

	t.Run("NewCollectEndorsementsView with nil", func(t *testing.T) {
		tx := &Transaction{Transaction: nil}
		// This should not panic during view creation
		view := NewCollectEndorsementsView(tx)
		assert.NotNil(t, view)
	})

	t.Run("NewOrderingAndFinalityView with nil", func(t *testing.T) {
		tx := &Transaction{Transaction: nil}
		view := NewOrderingAndFinalityView(tx)
		assert.NotNil(t, view)
	})

	t.Run("NewAcceptView with nil", func(t *testing.T) {
		tx := &Transaction{Transaction: nil}
		view := NewAcceptView(tx)
		assert.NotNil(t, view)
	})
}

// Made with Bob
