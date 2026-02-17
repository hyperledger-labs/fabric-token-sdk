/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package json

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestStruct struct {
	A string `json:"a"`
	B int    `json:"b"`
}

func TestUnmarshal(t *testing.T) {
	// Success
	data := []byte(`{"a": "hello", "b": 123}`)
	var v TestStruct
	err := Unmarshal(data, &v)
	assert.NoError(t, err)
	assert.Equal(t, "hello", v.A)
	assert.Equal(t, 123, v.B)

	// Unknown fields
	data = []byte(`{"a": "hello", "b": 123, "c": true}`)
	err = Unmarshal(data, &v)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown field \"c\"")

	// Invalid JSON
	data = []byte(`{"a": "hello", "b": 123`)
	err = Unmarshal(data, &v)
	assert.Error(t, err)
}

func TestMarshal(t *testing.T) {
	v := TestStruct{A: "hello", B: 123}
	data, err := Marshal(v)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"a": "hello", "b": 123}`, string(data))
}

func TestMarshalIndent(t *testing.T) {
	v := TestStruct{A: "hello", B: 123}
	data, err := MarshalIndent(v, "", "  ")
	assert.NoError(t, err)
	assert.Contains(t, string(data), "\"a\": \"hello\"")
}
