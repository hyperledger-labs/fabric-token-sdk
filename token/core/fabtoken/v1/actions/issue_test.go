/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/test-go/testify/require"
)

func TestIssueAction_Serialization(t *testing.T) {
	action := &IssueAction{
		Issuer: []byte("issuer"),
		Outputs: []*Output{
			{
				Owner:    []byte("owner"),
				Type:     "a type",
				Quantity: "a quantity",
			},
		},
		Metadata: map[string][]byte{
			"foo": []byte("bar"),
		},
	}
	raw, err := action.Serialize()
	require.NoError(t, err, "failed to serialize a new transfer action")

	action2 := &IssueAction{}
	err = action2.Deserialize(raw)
	require.NoError(t, err, "failed to deserialize a new transfer action")
	assert.Equal(t, action, action2, "deserialized action is not equal to the original one")

	raw2, err := action2.Serialize()
	require.NoError(t, err, "failed to serialize a new transfer action")

	action3 := &IssueAction{}
	err = action3.Deserialize(raw2)
	require.NoError(t, err, "failed to deserialize a new transfer action")
	assert.Equal(t, action2, action3, "deserialized action is not equal to the original one")
}

func TestIssueAction_Validate(t *testing.T) {
	tests := []struct {
		name          string
		action        *IssueAction
		wantErr       bool
		expectedError string
	}{
		{
			name:          "",
			action:        &IssueAction{},
			wantErr:       true,
			expectedError: "issuer is not set",
		},
		{
			name: "",
			action: &IssueAction{
				Issuer: []byte("issuer"),
			},
			wantErr:       true,
			expectedError: "no outputs in issue action",
		},
		{
			name: "",
			action: &IssueAction{
				Issuer: []byte("issuer"),
				Outputs: []*Output{
					nil,
				},
			},
			wantErr:       true,
			expectedError: "nil output in issue action",
		},
		{
			name: "",
			action: &IssueAction{
				Issuer: []byte("issuer"),
				Outputs: []*Output{
					{},
				},
			},
			wantErr:       true,
			expectedError: "invalid output's type at index [0], output type is empty",
		},
		{
			name: "",
			action: &IssueAction{
				Issuer: []byte("issuer"),
				Outputs: []*Output{
					{
						Type: "type",
					},
				},
			},
			wantErr:       true,
			expectedError: "invalid output's quantity at index [0], output quantity is empty",
		},
		{
			name: "",
			action: &IssueAction{
				Issuer: []byte("issuer"),
				Outputs: []*Output{
					{
						Owner:    []byte("owner"),
						Type:     "type",
						Quantity: "11",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.action.Validate()
			if tt.wantErr {
				require.Error(t, err)
				require.EqualError(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
