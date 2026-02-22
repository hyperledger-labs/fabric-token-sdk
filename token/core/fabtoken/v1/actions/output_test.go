/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutput_Serialize(t *testing.T) {
	output := &Output{
		Owner:    []byte("owner"),
		Type:     "type",
		Quantity: "100",
	}
	raw, err := output.Serialize()
	require.NoError(t, err)
	require.NotNil(t, raw)

	output2 := &Output{}
	err = output2.Deserialize(raw)
	require.NoError(t, err)
	assert.Equal(t, output, output2)
}

func TestOutput_IsRedeem(t *testing.T) {
	output := &Output{
		Owner:    []byte("owner"),
		Type:     "type",
		Quantity: "100",
	}
	assert.False(t, output.IsRedeem())

	output.Owner = nil
	assert.True(t, output.IsRedeem())
}

func TestOutput_GetOwner(t *testing.T) {
	owner := []byte("owner")
	output := &Output{
		Owner: owner,
	}
	assert.Equal(t, owner, output.GetOwner())
}

func TestOutput_Validate(t *testing.T) {
	tests := []struct {
		name       string
		output     *Output
		checkOwner bool
		wantErr    bool
	}{
		{
			name: "valid",
			output: &Output{
				Owner:    []byte("owner"),
				Type:     "type",
				Quantity: "100",
			},
			checkOwner: true,
			wantErr:    false,
		},
		{
			name: "missing owner",
			output: &Output{
				Owner:    nil,
				Type:     "type",
				Quantity: "100",
			},
			checkOwner: true,
			wantErr:    true,
		},
		{
			name: "missing owner, don't check",
			output: &Output{
				Owner:    nil,
				Type:     "type",
				Quantity: "100",
			},
			checkOwner: false,
			wantErr:    false,
		},
		{
			name: "missing type",
			output: &Output{
				Owner:    []byte("owner"),
				Type:     "",
				Quantity: "100",
			},
			checkOwner: true,
			wantErr:    true,
		},
		{
			name: "missing quantity",
			output: &Output{
				Owner:    []byte("owner"),
				Type:     "type",
				Quantity: "",
			},
			checkOwner: true,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.output.Validate(tt.checkOwner)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestOutputMetadata_Serialization(t *testing.T) {
	metadata := &OutputMetadata{
		Issuer: []byte("issuer"),
	}
	raw, err := metadata.Serialize()
	require.NoError(t, err)
	require.NotNil(t, raw)

	metadata2 := &OutputMetadata{}
	err = metadata2.Deserialize(raw)
	require.NoError(t, err)
	assert.Equal(t, metadata, metadata2)
}

func TestOutput_DeserializeError(t *testing.T) {
	output := &Output{}
	err := output.Deserialize([]byte("invalid"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed deserializing token")
}

func TestOutputMetadata_DeserializeError(t *testing.T) {
	metadata := &OutputMetadata{}
	err := metadata.Deserialize([]byte("invalid"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed deserializing metadata")
}
