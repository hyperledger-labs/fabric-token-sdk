/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package token tests Request serialization and application metadata handling.
package token

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/stretchr/testify/assert"
	"github.com/test-go/testify/require"
)

// TestRequestSerialization verifies that a Request can be serialized to bytes and
// deserialized back without data loss. Tests both full serialization (Bytes/FromBytes)
// and audit-specific serialization (MarshalToAudit).
func TestRequestSerialization(t *testing.T) {
	r := NewRequest(nil, "hello world")
	r.Actions = &driver.TokenRequest{
		Issues: [][]byte{
			[]byte("issue1"),
			[]byte("issue2"),
		},
		Transfers:  [][]byte{[]byte("transfer1")},
		Signatures: [][]byte{[]byte("signature1")},
		AuditorSignatures: []*driver.AuditorSignature{
			{
				Identity:  Identity("auditor1"),
				Signature: []byte("signature1"),
			},
		},
	}
	raw, err := r.Bytes()
	require.NoError(t, err)

	r2 := NewRequest(nil, "")
	err = r2.FromBytes(raw)
	require.NoError(t, err)
	raw2, err := r2.Bytes()
	require.NoError(t, err)

	assert.Equal(t, raw, raw2)

	mRaw, err := r.MarshalToAudit()
	require.NoError(t, err)
	mRaw2, err := r2.MarshalToAudit()
	require.NoError(t, err)

	assert.Equal(t, mRaw, mRaw2)
}

// TestRequest_ApplicationMetadata verifies that ApplicationMetadata correctly retrieves
// metadata values by key, returning nil for non-existent keys.
func TestRequest_ApplicationMetadata(t *testing.T) {
	// Test case: No application metadata set
	request := &Request{
		Metadata: &driver.TokenRequestMetadata{
			Application: map[string][]byte{},
		},
	}

	// Retrieve non-existent metadata
	data := request.ApplicationMetadata("key")
	assert.Nil(t, data)

	// Test case: Application metadata set
	request = &Request{
		Metadata: &driver.TokenRequestMetadata{
			Application: map[string][]byte{
				"key1": []byte("value1"),
				"key2": []byte("value2"),
			},
		},
	}

	// Retrieve existing metadata
	data = request.ApplicationMetadata("key1")
	assert.Equal(t, []byte("value1"), data)

	// Retrieve non-existent metadata
	data = request.ApplicationMetadata("non_existent_key")
	assert.Nil(t, data)
}

// TestRequest_SetApplicationMetadata verifies that SetApplicationMetadata correctly
// stores metadata values, initializing the metadata structure if needed.
func TestRequest_SetApplicationMetadata(t *testing.T) {
	// Test case: No application metadata set
	request := &Request{}

	// Set application metadata
	request.SetApplicationMetadata("key", []byte("value"))

	// Assert metadata set correctly
	assert.NotNil(t, request.Metadata)
	assert.NotNil(t, request.Metadata.Application)
	assert.Equal(t, []byte("value"), request.Metadata.Application["key"])

	// Test case: Application metadata already set
	request = &Request{
		Metadata: &driver.TokenRequestMetadata{
			Application: map[string][]byte{
				"key1": []byte("value1"),
			},
		},
	}

	// Set additional application metadata
	request.SetApplicationMetadata("key2", []byte("value2"))

	// Assert metadata set correctly
	assert.NotNil(t, request.Metadata)
	assert.NotNil(t, request.Metadata.Application)
	assert.Equal(t, []byte("value1"), request.Metadata.Application["key1"])
	assert.Equal(t, []byte("value2"), request.Metadata.Application["key2"])
}
