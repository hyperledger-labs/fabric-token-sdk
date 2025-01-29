/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package meta

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransferActionMetadata(t *testing.T) {
	tests := []struct {
		name     string
		attrs    map[interface{}]interface{}
		expected map[string][]byte
	}{
		{
			name: "Basic metadata extraction",
			attrs: map[interface{}]interface{}{
				"TransferMetadataPrefixKey1": []byte("value1"),
				"TransferMetadataPrefixKey2": []byte("value2"),
			},
			expected: map[string][]byte{
				"Key1": []byte("value1"),
				"Key2": []byte("value2"),
			},
		},
		{
			name:     "Empty attrs",
			attrs:    map[interface{}]interface{}{},
			expected: map[string][]byte{},
		},
		{
			name: "Non-string keys",
			attrs: map[interface{}]interface{}{
				123: []byte("value"),
			},
			expected: map[string][]byte{},
		},
		{
			name: "Invalid value types",
			attrs: map[interface{}]interface{}{
				"TransferMetadataPrefixKey": 123,
			},
			expected: map[string][]byte{},
		},
		{
			name: "Exact prefix match",
			attrs: map[interface{}]interface{}{
				"TransferMetadataPrefix": []byte("value"),
			},
			expected: map[string][]byte{
				"": []byte("value"),
			},
		},
		{
			name: "No prefix match",
			attrs: map[interface{}]interface{}{
				"WrongPrefixKey": []byte("value"),
			},
			expected: map[string][]byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TransferActionMetadata(tt.attrs)
			assert.Equal(t, len(result), len(tt.expected), "Expected %d items, got %d", len(tt.expected), len(result))
			for key, value := range tt.expected {
				gotValue, ok := result[key]
				assert.True(t, ok, "Expected key %s to exist in result", key)
				assert.Equal(t, value, gotValue, "Expected value for %s: %v, got: %v", key, value, gotValue)
			}
		})
	}
}
