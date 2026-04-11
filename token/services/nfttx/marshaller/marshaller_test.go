/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package marshaller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestStruct struct {
	Name  string
	Value int
	Tags  []string
}

func TestMarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name: "simple struct",
			input: TestStruct{
				Name:  "test",
				Value: 42,
				Tags:  []string{"tag1", "tag2"},
			},
			wantErr: false,
		},
		{
			name:    "nil value",
			input:   nil,
			wantErr: false,
		},
		{
			name: "map",
			input: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
			},
			wantErr: false,
		},
		{
			name:    "string",
			input:   "simple string",
			wantErr: false,
		},
		{
			name:    "number",
			input:   42,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := Marshal(tt.input)
			if tt.wantErr {
				assert.Error(t, err)

				return
			}
			require.NoError(t, err)
			assert.NotNil(t, data)
			assert.NotEmpty(t, data)
		})
	}
}

func TestUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		target  interface{}
		wantErr bool
	}{
		{
			name:    "valid json to struct",
			data:    []byte(`{"Name":"test","Value":42,"Tags":["tag1","tag2"]}`),
			target:  &TestStruct{},
			wantErr: false,
		},
		{
			name:    "invalid json",
			data:    []byte(`{invalid json}`),
			target:  &TestStruct{},
			wantErr: true,
		},
		{
			name:    "empty json",
			data:    []byte(`{}`),
			target:  &TestStruct{},
			wantErr: false,
		},
		{
			name:    "json to map",
			data:    []byte(`{"key":"value"}`),
			target:  &map[string]interface{}{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Unmarshal(tt.data, tt.target)
			if tt.wantErr {
				assert.Error(t, err)

				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	original := TestStruct{
		Name:  "roundtrip",
		Value: 100,
		Tags:  []string{"a", "b", "c"},
	}

	// Marshal
	data, err := Marshal(original)
	require.NoError(t, err)

	// Unmarshal
	var result TestStruct
	err = Unmarshal(data, &result)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.Name, result.Name)
	assert.Equal(t, original.Value, result.Value)
	assert.Equal(t, original.Tags, result.Tags)
}

// Made with Bob
