/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nfttx

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithUniqueID(t *testing.T) {
	uniqueID := "test-unique-id-12345"

	// Create the option
	opt := WithUniqueID(uniqueID)
	require.NotNil(t, opt)

	// Apply the option to IssueOptions
	opts := &token.IssueOptions{
		Attributes: make(map[interface{}]interface{}),
	}

	err := opt(opts)
	require.NoError(t, err)

	// Verify the attribute was set
	key := "github.com/hyperledger-labs/fabric-token-sdk/token/services/nfttx/UniqueID"
	value, exists := opts.Attributes[key]
	assert.True(t, exists, "UniqueID attribute should be set")
	valueStr, ok := value.(string)
	require.True(t, ok, "value should be a string")
	assert.Equal(t, uniqueID, valueStr)
}

func TestWithUniqueID_EmptyString(t *testing.T) {
	opt := WithUniqueID("")
	require.NotNil(t, opt)

	opts := &token.IssueOptions{
		Attributes: make(map[interface{}]interface{}),
	}

	err := opt(opts)
	require.NoError(t, err)

	key := "github.com/hyperledger-labs/fabric-token-sdk/token/services/nfttx/UniqueID"
	value, exists := opts.Attributes[key]
	assert.True(t, exists)
	valueStr, ok := value.(string)
	require.True(t, ok, "value should be a string")
	assert.Empty(t, valueStr)
}

func TestWithUniqueID_MultipleApplications(t *testing.T) {
	// Test that applying the option multiple times overwrites the previous value
	opts := &token.IssueOptions{
		Attributes: make(map[interface{}]interface{}),
	}

	opt1 := WithUniqueID("first-id")
	err := opt1(opts)
	require.NoError(t, err)

	opt2 := WithUniqueID("second-id")
	err = opt2(opts)
	require.NoError(t, err)

	key := "github.com/hyperledger-labs/fabric-token-sdk/token/services/nfttx/UniqueID"
	value, exists := opts.Attributes[key]
	assert.True(t, exists)
	valueStr, ok := value.(string)
	require.True(t, ok, "value should be a string")
	assert.Equal(t, "second-id", valueStr, "second application should overwrite first")
}

// Made with Bob
