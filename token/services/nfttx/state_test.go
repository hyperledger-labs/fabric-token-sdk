/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nfttx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing

type mockLinearState struct {
	linearID string
}

func (m *mockLinearState) SetLinearID(id string) string {
	m.linearID = id

	return id
}

type mockAutoLinearState struct {
	linearID string
	err      error
}

func (m *mockAutoLinearState) GetLinearID() (string, error) {
	return m.linearID, m.err
}

func TestLinearState_Interface(t *testing.T) {
	// Test that mockLinearState implements LinearState
	var _ LinearState = &mockLinearState{}

	state := &mockLinearState{}
	testID := "test-linear-id-123"

	returnedID := state.SetLinearID(testID)

	assert.Equal(t, testID, returnedID)
	assert.Equal(t, testID, state.linearID)
}

func TestAutoLinearState_Interface(t *testing.T) {
	// Test that mockAutoLinearState implements AutoLinearState
	var _ AutoLinearState = &mockAutoLinearState{}

	t.Run("success", func(t *testing.T) {
		expectedID := "auto-generated-id"
		state := &mockAutoLinearState{
			linearID: expectedID,
			err:      nil,
		}

		id, err := state.GetLinearID()

		require.NoError(t, err)
		assert.Equal(t, expectedID, id)
	})

	t.Run("error", func(t *testing.T) {
		state := &mockAutoLinearState{
			linearID: "",
			err:      assert.AnError,
		}

		id, err := state.GetLinearID()

		require.Error(t, err)
		assert.Empty(t, id)
	})
}

func TestLinearState_SetLinearID_Multiple(t *testing.T) {
	state := &mockLinearState{}

	// Set ID multiple times
	id1 := state.SetLinearID("first-id")
	assert.Equal(t, "first-id", id1)
	assert.Equal(t, "first-id", state.linearID)

	id2 := state.SetLinearID("second-id")
	assert.Equal(t, "second-id", id2)
	assert.Equal(t, "second-id", state.linearID)
}

func TestAutoLinearState_GetLinearID_Consistency(t *testing.T) {
	expectedID := "consistent-id"
	state := &mockAutoLinearState{
		linearID: expectedID,
		err:      nil,
	}

	// Call multiple times to ensure consistency
	for range 5 {
		id, err := state.GetLinearID()
		require.NoError(t, err)
		assert.Equal(t, expectedID, id)
	}
}

// Made with Bob
