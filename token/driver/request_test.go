/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
)

func TestTokenRequestSerialization(t *testing.T) {
	req := &TokenRequest{
		Issues: [][]byte{
			[]byte("issue1"),
			[]byte("issue2"),
		},
		Transfers:         [][]byte{[]byte("transfer1")},
		Signatures:        [][]byte{[]byte("signature1")},
		AuditorSignatures: [][]byte{[]byte("auditor_signature1")},
	}
	raw, err := req.Bytes()
	assert.NoError(t, err)

	req2 := &TokenRequest{}
	err = req2.FromBytes(raw)
	assert.NoError(t, err)
	assert.Equal(t, req, req2)
}

func TestTokenRequestMetadataSerialization(t *testing.T) {
	reqMeta := &TokenRequestMetadata{
		Issues: []*IssueMetadata{
			{
				Issuer: []byte{1, 2, 3},
				Inputs: []*IssueInputMetadata{},
				Outputs: []*IssueOutputMetadata{
					{
						OutputMetadata: []byte("token_info1"),
						Receivers: []*AuditableIdentity{
							{
								Identity:  []byte("receiver1"),
								AuditInfo: []byte("audit_info1"),
							},
						},
					},
					{
						OutputMetadata: []byte("token_info2"),
						Receivers: []*AuditableIdentity{
							{
								Identity:  []byte("receiver2"),
								AuditInfo: []byte("audit_info2"),
							},
						},
					},
				},
				ExtraSigners: []Identity{
					[]byte("issue_extra_signer1"),
					[]byte("issue_extra_signer2"),
				},
			},
		},
		Transfers: []*TransferMetadata{
			{
				Inputs: []*TransferInputMetadata{
					{
						TokenID: &token.ID{
							TxId:  "txid1",
							Index: 1,
						},
						Senders: []*AuditableIdentity{
							{
								Identity:  []byte("sender1"),
								AuditInfo: []byte("sender1_audit_info"),
							},
						},
					},
					{
						TokenID: &token.ID{
							TxId:  "txid2",
							Index: 1,
						},
						Senders: []*AuditableIdentity{
							{
								Identity:  []byte("sender2"),
								AuditInfo: []byte("sender2_audit_info"),
							},
						},
					},
				},
				Outputs: []*TransferOutputMetadata{
					{
						OutputAuditInfo: []byte("token_info_3"),
						OutputMetadata:  []byte("token_meta_3"),
						Receivers: []*AuditableIdentity{
							{
								Identity:  []byte("receiver3"),
								AuditInfo: []byte("audit_info3"),
							},
						},
					},
					{
						OutputAuditInfo: []byte("token_info_4"),
						OutputMetadata:  []byte("token_meta_4"),
						Receivers: []*AuditableIdentity{
							{
								Identity:  []byte("receiver4"),
								AuditInfo: []byte("audit_info4"),
							},
						},
					},
				},
				ExtraSigners: []Identity{
					[]byte("extra_signer1"),
					[]byte("extra_signer2"),
				},
			},
		},
		Application: map[string][]byte{
			"key1": []byte("value1"),
			"key2": []byte("value2"),
		},
	}

	raw, err := reqMeta.Bytes()
	assert.NoError(t, err)

	reqMeta2 := &TokenRequestMetadata{}
	err = reqMeta2.FromBytes(raw)
	assert.NoError(t, err)
	raw2, err := reqMeta2.Bytes()
	assert.NoError(t, err)

	assert.Equal(t, reqMeta, reqMeta2)
	assert.Equal(t, raw, raw2)
}
