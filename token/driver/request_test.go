/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/stretchr/testify/assert"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
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
	reqMeta := TokenRequestMetadata{
		Issues: []IssueMetadata{
			{
				Issuer: []byte{1, 2, 3},
				Outputs: [][]byte{
					[]byte("output1"),
					[]byte("output2"),
				},
				TokenInfo: [][]byte{
					[]byte("token_info1"),
					[]byte("token_info2"),
				},
				Receivers: []view.Identity{
					[]byte("receiver1"),
					[]byte("receiver2"),
				},
				AuditInfos: [][]byte{
					[]byte("audit_info1"),
					[]byte("audit_info2"),
				},
			},
		},
		Transfers: []TransferMetadata{
			{
				TokenIDs: []*token2.ID{
					{
						TxId:  "",
						Index: 1,
					},
					{
						TxId:  "txid2",
						Index: 2,
					},
				},
				Outputs: [][]byte{
					[]byte("output1"),
					[]byte("output2"),
				},
				TokenInfo: [][]byte{
					[]byte("token_info1"),
					[]byte("token_info2"),
				},
				Senders: []view.Identity{
					[]byte("sender1"),
					[]byte("sender2"),
				},
				SenderAuditInfos: [][]byte{
					[]byte("sender_audit_info1"),
					[]byte("sender_audit_info2"),
				},
				Receivers: []view.Identity{
					[]byte("receiver1"),
					[]byte("receiver2"),
				},
				ReceiverIsSender: []bool{true, false},
				ReceiverAuditInfos: [][]byte{
					[]byte("receiver_audit_info1"),
					[]byte("receiver_audit_info2"),
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

	assert.Equal(t, reqMeta, *reqMeta2)
	assert.Equal(t, raw, raw2)
}
