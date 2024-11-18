/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"encoding/json"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
)

func TestIssueActionMetadata(t *testing.T) {
	tests := []struct {
		name        string
		attributes  map[string][]byte
		opts        *driver.IssueOptions
		expected    map[string][]byte
		errExpected bool
	}{
		{
			name:       "empty attributes",
			attributes: map[string][]byte{},
			opts: &driver.IssueOptions{
				Attributes: map[interface{}]interface{}{},
			},
			expected:    map[string][]byte{},
			errExpected: false,
		},
		{
			name: "missing tokenID",
			attributes: map[string][]byte{
				NetworkKey: []byte("network"),
			},
			opts: &driver.IssueOptions{
				Attributes: map[interface{}]interface{}{
					NetworkKey: "network",
				},
			},
			expected: map[string][]byte{
				NetworkKey: []byte("network"),
			},
			errExpected: false,
		},
		{
			name: "missing network",
			attributes: map[string][]byte{
				TokenIDKey: []byte("tokenID"),
			},
			opts: &driver.IssueOptions{
				Attributes: map[interface{}]interface{}{
					TokenIDKey: "tokenID",
				},
			},
			expected: map[string][]byte{
				TokenIDKey: []byte("tokenID"),
			},
			errExpected: false,
		},
		{
			name: "valid attributes",
			attributes: map[string][]byte{
				NetworkKey: []byte("network"),
				TokenIDKey: []byte("tokenID"),
			},
			opts: &driver.IssueOptions{
				Attributes: map[interface{}]interface{}{
					NetworkKey: "network",
					TokenIDKey: &token.ID{
						TxId:  "a_transaction",
						Index: 2,
					},
				},
			},
			expected: func() map[string][]byte {
				metadata := &IssueMetadata{
					OriginTokenID: &token.ID{
						TxId:  "a_transaction",
						Index: 2,
					},
					OriginNetwork: "network",
				}
				marshalled, err := json.Marshal(metadata)
				assert.NoError(t, err)
				key := common.Hashable(marshalled).String()
				res := map[string][]byte{
					NetworkKey:             []byte("network"),
					TokenIDKey:             []byte("tokenID"),
					key:                    marshalled,
					key + "proof_of_claim": nil,
				}
				return res
			}(),
			errExpected: false,
		},
		{
			name:       "valid attributes with proof",
			attributes: map[string][]byte{},
			opts: &driver.IssueOptions{
				Attributes: map[interface{}]interface{}{
					NetworkKey: "network",
					TokenIDKey: &token.ID{
						TxId:  "a_transaction",
						Index: 2,
					},
					ProofKey: []byte("proof"),
				},
			},
			expected: func() map[string][]byte {
				metadata := &IssueMetadata{
					OriginTokenID: &token.ID{
						TxId:  "a_transaction",
						Index: 2,
					},
					OriginNetwork: "network",
				}
				marshalled, err := json.Marshal(metadata)
				assert.NoError(t, err)
				key := common.Hashable(marshalled).String()
				res := map[string][]byte{
					key:                    marshalled,
					key + "proof_of_claim": []byte("proof"),
				}
				return res
			}(),
			errExpected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := IssueActionMetadata(test.attributes, test.opts)
			if test.errExpected && err == nil {
				t.Errorf("expected error but got none")
			}
			if !test.errExpected && err != nil {
				t.Errorf("got error %v", err)
			}
			assert.Equal(t, test.expected, result)
		})
	}
}
