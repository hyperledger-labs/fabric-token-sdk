/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package json

import (
	"encoding/json"
	"testing"
)

// Test cases for UnmarshalWithDisallowUnknownFields
func TestUnmarshalWithDisallowUnknownFields(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
	}

	tests := []struct {
		name     string
		jsonData []byte
		wantErr  bool
	}{
		{
			name:     "Valid JSON without unknown fields",
			jsonData: []byte(`{"name": "test"}`),
			wantErr:  false,
		},
		{
			name:     "JSON with unknown field",
			jsonData: []byte(`{"name": "test", "age": 30}`),
			wantErr:  true,
		},
		{
			name:     "Invalid JSON",
			jsonData: []byte(`invalid json`),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result TestStruct
			err := UnmarshalWithDisallowUnknownFields(tt.jsonData, &result)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalWithDisallowUnknownFields() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// RunBenchmark comparing UnmarshalWithDisallowUnknownFields to json.Unmarshal
func BenchmarkUnmarshal(b *testing.B) {
	type TestStruct struct {
		Name string `json:"name"`
	}

	jsonData := []byte(`{"name": "benchmark"}`)

	b.Run("With DisallowUnknownFields", func(b *testing.B) {
		for range b.N {
			var result TestStruct
			err := UnmarshalWithDisallowUnknownFields(jsonData, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Standard json.Unmarshal", func(b *testing.B) {
		for range b.N {
			var result TestStruct
			err := json.Unmarshal(jsonData, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
