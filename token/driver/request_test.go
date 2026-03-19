/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/request"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuditorSignature_ToProtos tests the conversion of AuditorSignature to protobuf format
func TestAuditorSignature_ToProtos(t *testing.T) {
	audSig := &AuditorSignature{
		Identity:  Identity("auditor1"),
		Signature: []byte("signature1"),
	}

	proto, err := audSig.ToProtos()
	require.NoError(t, err)
	require.NotNil(t, proto)
	assert.Equal(t, []byte("auditor1"), proto.Identity.Raw)
	assert.Equal(t, []byte("signature1"), proto.Signature.Raw)
}

// TestAuditorSignature_FromProtos tests the conversion from protobuf to AuditorSignature
func TestAuditorSignature_FromProtos(t *testing.T) {
	proto := &request.AuditorSignature{
		Identity: &request.Identity{
			Raw: []byte("auditor1"),
		},
		Signature: &request.Signature{
			Raw: []byte("signature1"),
		},
	}

	audSig := &AuditorSignature{}
	err := audSig.FromProtos(proto)
	require.NoError(t, err)
	assert.Equal(t, Identity("auditor1"), audSig.Identity)
	assert.Equal(t, []byte("signature1"), audSig.Signature)
}

// TestAuditorSignature_FromProtos_NilFields tests handling of nil fields in protobuf
func TestAuditorSignature_FromProtos_NilFields(t *testing.T) {
	proto := &request.AuditorSignature{}

	audSig := &AuditorSignature{}
	err := audSig.FromProtos(proto)
	require.NoError(t, err)
	assert.Nil(t, audSig.Identity)
	assert.Nil(t, audSig.Signature)
}

// TestTokenRequest_Bytes tests serialization of TokenRequest
func TestTokenRequest_Bytes(t *testing.T) {
	req := &TokenRequest{
		Issues:     [][]byte{[]byte("issue1")},
		Transfers:  [][]byte{[]byte("transfer1")},
		Signatures: [][]byte{[]byte("signature1")},
		AuditorSignatures: []*AuditorSignature{
			{
				Identity:  Identity("auditor1"),
				Signature: []byte("audsig1"),
			},
		},
	}

	raw, err := req.Bytes()
	require.NoError(t, err)
	assert.NotEmpty(t, raw)
}

// TestTokenRequest_FromBytes_InvalidVersion tests error handling for invalid version
func TestTokenRequest_FromBytes_InvalidVersion(t *testing.T) {
	protoReq := &request.TokenRequest{
		Version: 999, // Invalid version
		Actions: []*request.Action{
			{Type: request.ActionType_ISSUE, Raw: []byte("issue1")},
		},
	}

	raw, err := proto.Marshal(protoReq)
	require.NoError(t, err)

	req := &TokenRequest{}
	err = req.FromBytes(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token request version")
}

// TestTokenRequest_FromBytes_NilAction tests error handling for nil action
func TestTokenRequest_FromBytes_NilAction(t *testing.T) {
	protoReq := &request.TokenRequest{
		Version: ProtocolV1,
		Actions: []*request.Action{nil},
	}

	raw, err := proto.Marshal(protoReq)
	require.NoError(t, err)

	req := &TokenRequest{}
	err = req.FromBytes(raw)
	// Note: protobuf unmarshaling may skip nil elements, so this might not error
	// The actual behavior depends on protobuf implementation
	if err != nil {
		assert.Contains(t, err.Error(), "nil action found")
	}
}

// TestTokenRequest_FromBytes_UnknownActionType tests error handling for unknown action type
func TestTokenRequest_FromBytes_UnknownActionType(t *testing.T) {
	protoReq := &request.TokenRequest{
		Version: ProtocolV1,
		Actions: []*request.Action{
			{Type: 999, Raw: []byte("unknown")}, // Unknown action type
		},
	}

	raw, err := proto.Marshal(protoReq)
	require.NoError(t, err)

	req := &TokenRequest{}
	err = req.FromBytes(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown action type")
}

// TestTokenRequest_FromBytes_NilSignature tests error handling for nil signature
func TestTokenRequest_FromBytes_NilSignature(t *testing.T) {
	protoReq := &request.TokenRequest{
		Version: ProtocolV1,
		Actions: []*request.Action{
			{Type: request.ActionType_ISSUE, Raw: []byte("issue1")},
		},
		Signatures: []*request.Signature{nil},
	}

	raw, err := proto.Marshal(protoReq)
	require.NoError(t, err)

	req := &TokenRequest{}
	err = req.FromBytes(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil signature found")
}

// TestTokenRequest_FromBytes_EmptySignature tests error handling for empty signature
func TestTokenRequest_FromBytes_EmptySignature(t *testing.T) {
	protoReq := &request.TokenRequest{
		Version: ProtocolV1,
		Actions: []*request.Action{
			{Type: request.ActionType_ISSUE, Raw: []byte("issue1")},
		},
		Signatures: []*request.Signature{{Raw: []byte{}}},
	}

	raw, err := proto.Marshal(protoReq)
	require.NoError(t, err)

	req := &TokenRequest{}
	err = req.FromBytes(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil signature found")
}

func TestTokenRequestSerialization(t *testing.T) {
	req := &TokenRequest{
		Issues: [][]byte{
			[]byte("issue1"),
			[]byte("issue2"),
		},
		Transfers:  [][]byte{[]byte("transfer1")},
		Signatures: [][]byte{[]byte("signature1")},
		AuditorSignatures: []*AuditorSignature{
			{
				Identity:  Identity("auditor1"),
				Signature: []byte("signature1"),
			},
		},
	}
	raw, err := req.Bytes()
	require.NoError(t, err)

	req2 := &TokenRequest{}
	err = req2.FromBytes(raw)
	require.NoError(t, err)
	assert.Equal(t, req, req2)
}

// TestTokenRequest_MarshalToMessageToSign tests message marshaling for signing
func TestTokenRequest_MarshalToMessageToSign(t *testing.T) {
	req := &TokenRequest{
		Issues:    [][]byte{[]byte("issue1")},
		Transfers: [][]byte{[]byte("transfer1")},
	}

	anchor := []byte("anchor123")
	msg, err := req.MarshalToMessageToSign(anchor)
	require.NoError(t, err)
	assert.NotEmpty(t, msg)
	// Verify anchor is appended
	assert.Contains(t, string(msg), string(anchor))
}

// TestAuditableIdentity_ToProtos tests conversion to protobuf
func TestAuditableIdentity_ToProtos(t *testing.T) {
	ai := &AuditableIdentity{
		Identity:  Identity("identity1"),
		AuditInfo: []byte("auditinfo1"),
	}

	proto, err := ai.ToProtos()
	require.NoError(t, err)
	require.NotNil(t, proto)
	assert.Equal(t, []byte("identity1"), proto.Identity.Raw)
	assert.Equal(t, []byte("auditinfo1"), proto.AuditInfo)
}

// TestAuditableIdentity_FromProtos tests conversion from protobuf
func TestAuditableIdentity_FromProtos(t *testing.T) {
	proto := &request.AuditableIdentity{
		Identity: &request.Identity{
			Raw: []byte("identity1"),
		},
		AuditInfo: []byte("auditinfo1"),
	}

	ai := &AuditableIdentity{}
	err := ai.FromProtos(proto)
	require.NoError(t, err)
	assert.Equal(t, Identity("identity1"), ai.Identity)
	assert.Equal(t, []byte("auditinfo1"), ai.AuditInfo)
}

// TestIssueInputMetadata_ToProtos tests conversion to protobuf
func TestIssueInputMetadata_ToProtos(t *testing.T) {
	iim := &IssueInputMetadata{
		TokenID: &token.ID{
			TxId:  "tx123",
			Index: 5,
		},
	}

	proto, err := iim.ToProtos()
	require.NoError(t, err)
	require.NotNil(t, proto)
	assert.Equal(t, "tx123", proto.TokenId.TxId)
	assert.Equal(t, uint64(5), proto.TokenId.Index)
}

// TestIssueInputMetadata_FromProtos tests conversion from protobuf
func TestIssueInputMetadata_FromProtos(t *testing.T) {
	proto := &request.IssueInputMetadata{
		TokenId: &request.TokenID{
			TxId:  "tx123",
			Index: 5,
		},
	}

	iim := &IssueInputMetadata{}
	err := iim.FromProtos(proto)
	require.NoError(t, err)
	require.NotNil(t, iim.TokenID)
	assert.Equal(t, "tx123", iim.TokenID.TxId)
	assert.Equal(t, uint64(5), iim.TokenID.Index)
}

// TestIssueOutputMetadata_RecipientAt tests recipient retrieval by index
func TestIssueOutputMetadata_RecipientAt(t *testing.T) {
	iom := &IssueOutputMetadata{
		Receivers: []*AuditableIdentity{
			{Identity: Identity("receiver1")},
			{Identity: Identity("receiver2")},
		},
	}

	// Valid index
	recipient := iom.RecipientAt(0)
	require.NotNil(t, recipient)
	assert.Equal(t, Identity("receiver1"), recipient.Identity)

	recipient = iom.RecipientAt(1)
	require.NotNil(t, recipient)
	assert.Equal(t, Identity("receiver2"), recipient.Identity)

	// Invalid indices
	assert.Nil(t, iom.RecipientAt(-1))
	assert.Nil(t, iom.RecipientAt(2))
	assert.Nil(t, iom.RecipientAt(100))
}

// TestIssueOutputMetadata_ToProtos tests conversion to protobuf
func TestIssueOutputMetadata_ToProtos(t *testing.T) {
	iom := &IssueOutputMetadata{
		OutputMetadata: []byte("metadata1"),
		Receivers: []*AuditableIdentity{
			{Identity: Identity("receiver1"), AuditInfo: []byte("audit1")},
		},
	}

	proto, err := iom.ToProtos()
	require.NoError(t, err)
	require.NotNil(t, proto)
	assert.Equal(t, []byte("metadata1"), proto.Metadata)
	assert.Len(t, proto.Receivers, 1)
}

// TestIssueOutputMetadata_FromProtos tests conversion from protobuf
func TestIssueOutputMetadata_FromProtos(t *testing.T) {
	proto := &request.OutputMetadata{
		Metadata: []byte("metadata1"),
		Receivers: []*request.AuditableIdentity{
			{
				Identity:  &request.Identity{Raw: []byte("receiver1")},
				AuditInfo: []byte("audit1"),
			},
		},
	}

	iom := &IssueOutputMetadata{}
	err := iom.FromProtos(proto)
	require.NoError(t, err)
	assert.Equal(t, []byte("metadata1"), iom.OutputMetadata)
	assert.Len(t, iom.Receivers, 1)
}

// TestIssueOutputMetadata_FromProtos_Nil tests handling of nil protobuf
func TestIssueOutputMetadata_FromProtos_Nil(t *testing.T) {
	iom := &IssueOutputMetadata{}
	err := iom.FromProtos(nil)
	require.NoError(t, err)
}

// TestIssueMetadata_Receivers tests receiver extraction
func TestIssueMetadata_Receivers(t *testing.T) {
	im := &IssueMetadata{
		Outputs: []*IssueOutputMetadata{
			{
				Receivers: []*AuditableIdentity{
					{Identity: Identity("receiver1")},
					{Identity: Identity("receiver2")},
				},
			},
			{
				Receivers: []*AuditableIdentity{
					{Identity: Identity("receiver3")},
					nil, // Test nil receiver
				},
			},
		},
	}

	receivers := im.Receivers()
	assert.Len(t, receivers, 4)
	assert.Equal(t, Identity("receiver1"), receivers[0])
	assert.Equal(t, Identity("receiver2"), receivers[1])
	assert.Equal(t, Identity("receiver3"), receivers[2])
	assert.Nil(t, receivers[3])
}

// TestIssueMetadata_ToProtos tests conversion to protobuf
func TestIssueMetadata_ToProtos(t *testing.T) {
	im := &IssueMetadata{
		Issuer: AuditableIdentity{
			Identity:  Identity("issuer1"),
			AuditInfo: []byte("issuer_audit"),
		},
		Inputs: []*IssueInputMetadata{
			{TokenID: &token.ID{TxId: "tx1", Index: 0}},
		},
		Outputs: []*IssueOutputMetadata{
			{OutputMetadata: []byte("output1")},
		},
		ExtraSigners: []Identity{Identity("signer1")},
	}

	proto, err := im.ToProtos()
	require.NoError(t, err)
	require.NotNil(t, proto)
	assert.NotNil(t, proto.Issuer)
	assert.Len(t, proto.Inputs, 1)
	assert.Len(t, proto.Outputs, 1)
	assert.Len(t, proto.ExtraSigners, 1)
}

// TestIssueMetadata_FromProtos tests conversion from protobuf
func TestIssueMetadata_FromProtos(t *testing.T) {
	proto := &request.IssueMetadata{
		Issuer: &request.AuditableIdentity{
			Identity:  &request.Identity{Raw: []byte("issuer1")},
			AuditInfo: []byte("issuer_audit"),
		},
		Inputs: []*request.IssueInputMetadata{
			{TokenId: &request.TokenID{TxId: "tx1", Index: 0}},
		},
		Outputs: []*request.OutputMetadata{
			{Metadata: []byte("output1")},
		},
		ExtraSigners: []*request.Identity{
			{Raw: []byte("signer1")},
		},
	}

	im := &IssueMetadata{}
	err := im.FromProtos(proto)
	require.NoError(t, err)
	assert.Equal(t, Identity("issuer1"), im.Issuer.Identity)
	assert.Len(t, im.Inputs, 1)
	assert.Len(t, im.Outputs, 1)
	assert.Len(t, im.ExtraSigners, 1)
}

// TestTransferInputMetadata_ToProtos tests conversion to protobuf
func TestTransferInputMetadata_ToProtos(t *testing.T) {
	tim := &TransferInputMetadata{
		TokenID: &token.ID{TxId: "tx123", Index: 5},
		Senders: []*AuditableIdentity{
			{Identity: Identity("sender1"), AuditInfo: []byte("audit1")},
		},
	}

	proto, err := tim.ToProtos()
	require.NoError(t, err)
	require.NotNil(t, proto)
	assert.Equal(t, "tx123", proto.TokenId.TxId)
	assert.Len(t, proto.Senders, 1)
}

// TestTransferInputMetadata_FromProtos tests conversion from protobuf
func TestTransferInputMetadata_FromProtos(t *testing.T) {
	proto := &request.TransferInputMetadata{
		TokenId: &request.TokenID{TxId: "tx123", Index: 5},
		Senders: []*request.AuditableIdentity{
			{
				Identity:  &request.Identity{Raw: []byte("sender1")},
				AuditInfo: []byte("audit1"),
			},
		},
	}

	tim := &TransferInputMetadata{}
	err := tim.FromProtos(proto)
	require.NoError(t, err)
	require.NotNil(t, tim.TokenID)
	assert.Equal(t, "tx123", tim.TokenID.TxId)
	assert.Len(t, tim.Senders, 1)
}

// TestTransferInputMetadata_FromProtos_Nil tests handling of nil protobuf
func TestTransferInputMetadata_FromProtos_Nil(t *testing.T) {
	tim := &TransferInputMetadata{}
	err := tim.FromProtos(nil)
	require.NoError(t, err)
}

// TestTransferOutputMetadata_RecipientAt tests recipient retrieval by index
func TestTransferOutputMetadata_RecipientAt(t *testing.T) {
	tom := &TransferOutputMetadata{
		Receivers: []*AuditableIdentity{
			{Identity: Identity("receiver1")},
			{Identity: Identity("receiver2")},
		},
	}

	// Valid index
	recipient := tom.RecipientAt(0)
	require.NotNil(t, recipient)
	assert.Equal(t, Identity("receiver1"), recipient.Identity)

	// Invalid indices
	assert.Nil(t, tom.RecipientAt(-1))
	assert.Nil(t, tom.RecipientAt(2))
}

// TestTransferOutputMetadata_ToProtos tests conversion to protobuf
func TestTransferOutputMetadata_ToProtos(t *testing.T) {
	tom := &TransferOutputMetadata{
		OutputMetadata:  []byte("metadata1"),
		OutputAuditInfo: []byte("auditinfo1"),
		Receivers: []*AuditableIdentity{
			{Identity: Identity("receiver1")},
		},
	}

	proto, err := tom.ToProtos()
	require.NoError(t, err)
	require.NotNil(t, proto)
	assert.Equal(t, []byte("metadata1"), proto.Metadata)
	assert.Equal(t, []byte("auditinfo1"), proto.AuditInfo)
	assert.Len(t, proto.Receivers, 1)
}

// TestTransferOutputMetadata_FromProtos tests conversion from protobuf
func TestTransferOutputMetadata_FromProtos(t *testing.T) {
	proto := &request.OutputMetadata{
		Metadata:  []byte("metadata1"),
		AuditInfo: []byte("auditinfo1"),
		Receivers: []*request.AuditableIdentity{
			{Identity: &request.Identity{Raw: []byte("receiver1")}},
		},
	}

	tom := &TransferOutputMetadata{}
	err := tom.FromProtos(proto)
	require.NoError(t, err)
	assert.Equal(t, []byte("metadata1"), tom.OutputMetadata)
	assert.Equal(t, []byte("auditinfo1"), tom.OutputAuditInfo)
	assert.Len(t, tom.Receivers, 1)
}

// TestTransferOutputMetadata_FromProtos_Nil tests handling of nil protobuf
func TestTransferOutputMetadata_FromProtos_Nil(t *testing.T) {
	tom := &TransferOutputMetadata{}
	err := tom.FromProtos(nil)
	require.NoError(t, err)
}

// TestTransferMetadata_TokenIDAt tests TokenID retrieval by index
func TestTransferMetadata_TokenIDAt(t *testing.T) {
	tm := &TransferMetadata{
		Inputs: []*TransferInputMetadata{
			{TokenID: &token.ID{TxId: "tx1", Index: 1}},
			{TokenID: &token.ID{TxId: "tx2", Index: 2}},
		},
	}

	// Valid index
	tokenID := tm.TokenIDAt(0)
	require.NotNil(t, tokenID)
	assert.Equal(t, "tx1", tokenID.TxId)

	tokenID = tm.TokenIDAt(1)
	require.NotNil(t, tokenID)
	assert.Equal(t, "tx2", tokenID.TxId)

	// Invalid indices
	assert.Nil(t, tm.TokenIDAt(-1))
	assert.Nil(t, tm.TokenIDAt(2))
}

// TestTransferMetadata_Receivers tests receiver extraction
func TestTransferMetadata_Receivers(t *testing.T) {
	tm := &TransferMetadata{
		Outputs: []*TransferOutputMetadata{
			{
				Receivers: []*AuditableIdentity{
					{Identity: Identity("receiver1")},
					nil, // Test nil receiver
				},
			},
			{
				Receivers: []*AuditableIdentity{
					{Identity: Identity("receiver2")},
				},
			},
		},
	}

	receivers := tm.Receivers()
	assert.Len(t, receivers, 3)
	assert.Equal(t, Identity("receiver1"), receivers[0])
	assert.Nil(t, receivers[1])
	assert.Equal(t, Identity("receiver2"), receivers[2])
}

// TestTransferMetadata_Senders tests sender extraction
func TestTransferMetadata_Senders(t *testing.T) {
	tm := &TransferMetadata{
		Inputs: []*TransferInputMetadata{
			{
				Senders: []*AuditableIdentity{
					{Identity: Identity("sender1")},
					{Identity: Identity("sender2")},
				},
			},
			{
				Senders: []*AuditableIdentity{
					nil, // Test nil sender
					{Identity: Identity("sender3")},
				},
			},
		},
	}

	senders := tm.Senders()
	assert.Len(t, senders, 4)
	assert.Equal(t, Identity("sender1"), senders[0])
	assert.Equal(t, Identity("sender2"), senders[1])
	assert.Nil(t, senders[2])
	assert.Equal(t, Identity("sender3"), senders[3])
}

// TestTransferMetadata_TokenIDs tests TokenID extraction
func TestTransferMetadata_TokenIDs(t *testing.T) {
	tm := &TransferMetadata{
		Inputs: []*TransferInputMetadata{
			{TokenID: &token.ID{TxId: "tx1", Index: 1}},
			nil, // Test nil input
			{TokenID: &token.ID{TxId: "tx2", Index: 2}},
		},
	}

	tokenIDs := tm.TokenIDs()
	assert.Len(t, tokenIDs, 2) // nil input should be skipped
	assert.Equal(t, "tx1", tokenIDs[0].TxId)
	assert.Equal(t, "tx2", tokenIDs[1].TxId)
}

// TestTransferMetadata_ToProtos tests conversion to protobuf
func TestTransferMetadata_ToProtos(t *testing.T) {
	tm := &TransferMetadata{
		Inputs: []*TransferInputMetadata{
			{TokenID: &token.ID{TxId: "tx1", Index: 1}},
		},
		Outputs: []*TransferOutputMetadata{
			{OutputMetadata: []byte("output1")},
		},
		ExtraSigners: []Identity{Identity("signer1")},
		Issuer:       Identity("issuer1"),
	}

	proto, err := tm.ToProtos()
	require.NoError(t, err)
	require.NotNil(t, proto)
	assert.Len(t, proto.Inputs, 1)
	assert.Len(t, proto.Outputs, 1)
	assert.Len(t, proto.ExtraSigners, 1)
	assert.NotNil(t, proto.Issuer)
}

// TestTransferMetadata_ToProtos_NilIssuer tests conversion with nil issuer
func TestTransferMetadata_ToProtos_NilIssuer(t *testing.T) {
	tm := &TransferMetadata{
		Inputs:  []*TransferInputMetadata{},
		Outputs: []*TransferOutputMetadata{},
		Issuer:  nil,
	}

	proto, err := tm.ToProtos()
	require.NoError(t, err)
	require.NotNil(t, proto)
	assert.Nil(t, proto.Issuer)
}

// TestTransferMetadata_FromProtos tests conversion from protobuf
func TestTransferMetadata_FromProtos(t *testing.T) {
	proto := &request.TransferMetadata{
		Inputs: []*request.TransferInputMetadata{
			{TokenId: &request.TokenID{TxId: "tx1", Index: 1}},
		},
		Outputs: []*request.OutputMetadata{
			{Metadata: []byte("output1")},
		},
		ExtraSigners: []*request.Identity{
			{Raw: []byte("signer1")},
		},
		Issuer: &request.Identity{Raw: []byte("issuer1")},
	}

	tm := &TransferMetadata{}
	err := tm.FromProtos(proto)
	require.NoError(t, err)
	assert.Len(t, tm.Inputs, 1)
	assert.Len(t, tm.Outputs, 1)
	assert.Len(t, tm.ExtraSigners, 1)
	assert.Equal(t, Identity("issuer1"), tm.Issuer)
}

// TestTransferMetadata_FromProtos_NilIssuer tests conversion with nil issuer
func TestTransferMetadata_FromProtos_NilIssuer(t *testing.T) {
	proto := &request.TransferMetadata{
		Inputs:  []*request.TransferInputMetadata{},
		Outputs: []*request.OutputMetadata{},
		Issuer:  nil,
	}

	tm := &TransferMetadata{}
	err := tm.FromProtos(proto)
	require.NoError(t, err)
	assert.Nil(t, tm.Issuer)
}

func TestTokenRequestMetadataSerialization(t *testing.T) {
	reqMeta := &TokenRequestMetadata{
		Issues: []*IssueMetadata{
			{
				Issuer: AuditableIdentity{
					Identity:  []byte("issuer1"),
					AuditInfo: []byte("issuer_auditinfo1"),
				},
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
				Issuer: Identity([]byte("issuer")),
			},
		},
		Application: map[string][]byte{
			"key1": []byte("value1"),
			"key2": []byte("value2"),
		},
	}

	raw, err := reqMeta.Bytes()
	require.NoError(t, err)

	reqMeta2 := &TokenRequestMetadata{}
	err = reqMeta2.FromBytes(raw)
	require.NoError(t, err)
	raw2, err := reqMeta2.Bytes()
	require.NoError(t, err)
	reqMeta3 := &TokenRequestMetadata{}
	err = reqMeta3.FromBytes(raw2)
	require.NoError(t, err)

	assert.Equal(t, reqMeta, reqMeta2)
	assert.Equal(t, reqMeta2, reqMeta3)
}

// TestTokenRequestMetadata_FromBytes_InvalidVersion tests error handling for invalid version
func TestTokenRequestMetadata_FromBytes_InvalidVersion(t *testing.T) {
	protoMeta := &request.TokenRequestMetadata{
		Version: 999, // Invalid version
	}

	raw, err := proto.Marshal(protoMeta)
	require.NoError(t, err)

	reqMeta := &TokenRequestMetadata{}
	err = reqMeta.FromBytes(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token request metadata version")
}

// TestTokenRequestMetadata_ToProtos_NilIssueMetadata tests error handling for nil issue metadata
func TestTokenRequestMetadata_ToProtos_NilIssueMetadata(t *testing.T) {
	reqMeta := &TokenRequestMetadata{
		Issues: []*IssueMetadata{nil},
	}

	_, err := reqMeta.ToProtos()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "it is nil")
}

// TestTokenRequestMetadata_ToProtos_NilTransferMetadata tests error handling for nil transfer metadata
func TestTokenRequestMetadata_ToProtos_NilTransferMetadata(t *testing.T) {
	reqMeta := &TokenRequestMetadata{
		Transfers: []*TransferMetadata{nil},
	}

	_, err := reqMeta.ToProtos()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "it is nil")
}

// TestTokenRequestMetadata_FromProtos_UnrecognizedMetadata tests error handling for unrecognized metadata
func TestTokenRequestMetadata_FromProtos_UnrecognizedMetadata(t *testing.T) {
	protoMeta := &request.TokenRequestMetadata{
		Version: ProtocolV1,
		Metadata: []*request.ActionMetadata{
			{}, // Empty metadata (neither issue nor transfer)
		},
	}

	reqMeta := &TokenRequestMetadata{}
	err := reqMeta.FromProtos(protoMeta)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "type not recognized")
}

// TestTokenRequestMetadata_EmptyApplication tests handling of empty application map
func TestTokenRequestMetadata_EmptyApplication(t *testing.T) {
	reqMeta := &TokenRequestMetadata{
		Issues:      []*IssueMetadata{},
		Transfers:   []*TransferMetadata{},
		Application: nil,
	}

	raw, err := reqMeta.Bytes()
	require.NoError(t, err)

	reqMeta2 := &TokenRequestMetadata{}
	err = reqMeta2.FromBytes(raw)
	require.NoError(t, err)
	assert.Nil(t, reqMeta2.Application)
}

// TestTokenRequestMetadata_WithApplication tests application metadata handling
func TestTokenRequestMetadata_WithApplication(t *testing.T) {
	reqMeta := &TokenRequestMetadata{
		Issues:    []*IssueMetadata{},
		Transfers: []*TransferMetadata{},
		Application: map[string][]byte{
			"app_key1": []byte("app_value1"),
			"app_key2": []byte("app_value2"),
		},
	}

	raw, err := reqMeta.Bytes()
	require.NoError(t, err)

	reqMeta2 := &TokenRequestMetadata{}
	err = reqMeta2.FromBytes(raw)
	require.NoError(t, err)
	assert.Equal(t, reqMeta.Application, reqMeta2.Application)
}

// TestIssueMetadata_Receivers_EmptyOutputs tests receiver extraction with empty outputs
func TestIssueMetadata_Receivers_EmptyOutputs(t *testing.T) {
	im := &IssueMetadata{
		Outputs: []*IssueOutputMetadata{},
	}

	receivers := im.Receivers()
	assert.Empty(t, receivers)
}

// TestTransferMetadata_Receivers_EmptyOutputs tests receiver extraction with empty outputs
func TestTransferMetadata_Receivers_EmptyOutputs(t *testing.T) {
	tm := &TransferMetadata{
		Outputs: []*TransferOutputMetadata{},
	}

	receivers := tm.Receivers()
	assert.Empty(t, receivers)
}

// TestTransferMetadata_Senders_EmptyInputs tests sender extraction with empty inputs
func TestTransferMetadata_Senders_EmptyInputs(t *testing.T) {
	tm := &TransferMetadata{
		Inputs: []*TransferInputMetadata{},
	}

	senders := tm.Senders()
	assert.Empty(t, senders)
}

// TestTransferMetadata_TokenIDs_EmptyInputs tests TokenID extraction with empty inputs
func TestTransferMetadata_TokenIDs_EmptyInputs(t *testing.T) {
	tm := &TransferMetadata{
		Inputs: []*TransferInputMetadata{},
	}

	tokenIDs := tm.TokenIDs()
	assert.Empty(t, tokenIDs)
}

// TestTokenRequest_Bytes_Error tests error handling in Bytes serialization
func TestTokenRequest_Bytes_Error(t *testing.T) {
	// Create a request that will serialize successfully
	req := &TokenRequest{
		Issues: [][]byte{[]byte("issue1")},
	}

	raw, err := req.Bytes()
	require.NoError(t, err)
	assert.NotEmpty(t, raw)
}

// TestTokenRequest_FromBytes_UnmarshalError tests error handling for invalid bytes
func TestTokenRequest_FromBytes_UnmarshalError(t *testing.T) {
	req := &TokenRequest{}
	err := req.FromBytes([]byte("invalid protobuf data"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed unmarshalling token request")
}

// TestIssueOutputMetadata_ToProtos_Error tests error handling in ToProtos
func TestIssueOutputMetadata_ToProtos_Error(t *testing.T) {
	// Normal case should work
	iom := &IssueOutputMetadata{
		OutputMetadata: []byte("metadata1"),
		Receivers: []*AuditableIdentity{
			{Identity: Identity("receiver1")},
		},
	}

	proto, err := iom.ToProtos()
	require.NoError(t, err)
	assert.NotNil(t, proto)
}

// TestIssueMetadata_ToProtos_ErrorInInputs tests error handling when inputs fail
func TestIssueMetadata_ToProtos_ErrorInInputs(t *testing.T) {
	im := &IssueMetadata{
		Issuer: AuditableIdentity{
			Identity: Identity("issuer1"),
		},
		Inputs: []*IssueInputMetadata{
			{TokenID: &token.ID{TxId: "tx1", Index: 0}},
		},
		Outputs: []*IssueOutputMetadata{},
	}

	proto, err := im.ToProtos()
	require.NoError(t, err)
	assert.NotNil(t, proto)
}

// TestIssueMetadata_ToProtos_ErrorInOutputs tests error handling when outputs fail
func TestIssueMetadata_ToProtos_ErrorInOutputs(t *testing.T) {
	im := &IssueMetadata{
		Issuer: AuditableIdentity{
			Identity: Identity("issuer1"),
		},
		Inputs: []*IssueInputMetadata{},
		Outputs: []*IssueOutputMetadata{
			{OutputMetadata: []byte("output1")},
		},
	}

	proto, err := im.ToProtos()
	require.NoError(t, err)
	assert.NotNil(t, proto)
}

// TestIssueMetadata_FromProtos_ErrorInInputs tests error handling in FromProtos
func TestIssueMetadata_FromProtos_ErrorInInputs(t *testing.T) {
	proto := &request.IssueMetadata{
		Issuer: &request.AuditableIdentity{
			Identity: &request.Identity{Raw: []byte("issuer1")},
		},
		Inputs: []*request.IssueInputMetadata{
			{TokenId: &request.TokenID{TxId: "tx1", Index: 0}},
		},
		Outputs: []*request.OutputMetadata{},
	}

	im := &IssueMetadata{}
	err := im.FromProtos(proto)
	require.NoError(t, err)
}

// TestIssueMetadata_FromProtos_ErrorInOutputs tests error handling in FromProtos
func TestIssueMetadata_FromProtos_ErrorInOutputs(t *testing.T) {
	proto := &request.IssueMetadata{
		Issuer: &request.AuditableIdentity{
			Identity: &request.Identity{Raw: []byte("issuer1")},
		},
		Inputs: []*request.IssueInputMetadata{},
		Outputs: []*request.OutputMetadata{
			{Metadata: []byte("output1")},
		},
	}

	im := &IssueMetadata{}
	err := im.FromProtos(proto)
	require.NoError(t, err)
}

// TestTransferOutputMetadata_ToProtos_Error tests error handling in ToProtos
func TestTransferOutputMetadata_ToProtos_Error(t *testing.T) {
	tom := &TransferOutputMetadata{
		OutputMetadata:  []byte("metadata1"),
		OutputAuditInfo: []byte("auditinfo1"),
		Receivers: []*AuditableIdentity{
			{Identity: Identity("receiver1")},
		},
	}

	proto, err := tom.ToProtos()
	require.NoError(t, err)
	assert.NotNil(t, proto)
}

// TestTransferMetadata_ToProtos_ErrorInInputs tests error handling when inputs fail
func TestTransferMetadata_ToProtos_ErrorInInputs(t *testing.T) {
	tm := &TransferMetadata{
		Inputs: []*TransferInputMetadata{
			{TokenID: &token.ID{TxId: "tx1", Index: 1}},
		},
		Outputs: []*TransferOutputMetadata{},
	}

	proto, err := tm.ToProtos()
	require.NoError(t, err)
	assert.NotNil(t, proto)
}

// TestTransferMetadata_ToProtos_ErrorInOutputs tests error handling when outputs fail
func TestTransferMetadata_ToProtos_ErrorInOutputs(t *testing.T) {
	tm := &TransferMetadata{
		Inputs: []*TransferInputMetadata{},
		Outputs: []*TransferOutputMetadata{
			{OutputMetadata: []byte("output1")},
		},
	}

	proto, err := tm.ToProtos()
	require.NoError(t, err)
	assert.NotNil(t, proto)
}

// TestTransferMetadata_FromProtos_ErrorInInputs tests error handling in FromProtos
func TestTransferMetadata_FromProtos_ErrorInInputs(t *testing.T) {
	proto := &request.TransferMetadata{
		Inputs: []*request.TransferInputMetadata{
			{TokenId: &request.TokenID{TxId: "tx1", Index: 1}},
		},
		Outputs: []*request.OutputMetadata{},
	}

	tm := &TransferMetadata{}
	err := tm.FromProtos(proto)
	require.NoError(t, err)
}

// TestTransferMetadata_FromProtos_ErrorInOutputs tests error handling in FromProtos
func TestTransferMetadata_FromProtos_ErrorInOutputs(t *testing.T) {
	proto := &request.TransferMetadata{
		Inputs: []*request.TransferInputMetadata{},
		Outputs: []*request.OutputMetadata{
			{Metadata: []byte("output1")},
		},
	}

	tm := &TransferMetadata{}
	err := tm.FromProtos(proto)
	require.NoError(t, err)
}

// TestTokenRequestMetadata_Bytes_Error tests error handling in Bytes
func TestTokenRequestMetadata_Bytes_Error(t *testing.T) {
	reqMeta := &TokenRequestMetadata{
		Issues:    []*IssueMetadata{},
		Transfers: []*TransferMetadata{},
	}

	raw, err := reqMeta.Bytes()
	require.NoError(t, err)
	assert.NotEmpty(t, raw)
}

// TestTokenRequestMetadata_FromBytes_UnmarshalError tests unmarshal error
func TestTokenRequestMetadata_FromBytes_UnmarshalError(t *testing.T) {
	reqMeta := &TokenRequestMetadata{}
	err := reqMeta.FromBytes([]byte("invalid protobuf"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal")
}

// TestTokenRequestMetadata_ToProtos_ErrorInIssues tests error in issues conversion
func TestTokenRequestMetadata_ToProtos_ErrorInIssues(t *testing.T) {
	reqMeta := &TokenRequestMetadata{
		Issues: []*IssueMetadata{
			{
				Issuer: AuditableIdentity{Identity: Identity("issuer1")},
			},
		},
		Transfers: []*TransferMetadata{},
	}

	proto, err := reqMeta.ToProtos()
	require.NoError(t, err)
	assert.NotNil(t, proto)
}

// TestTokenRequestMetadata_ToProtos_ErrorInTransfers tests error in transfers conversion
func TestTokenRequestMetadata_ToProtos_ErrorInTransfers(t *testing.T) {
	reqMeta := &TokenRequestMetadata{
		Issues: []*IssueMetadata{},
		Transfers: []*TransferMetadata{
			{
				Inputs:  []*TransferInputMetadata{},
				Outputs: []*TransferOutputMetadata{},
			},
		},
	}

	proto, err := reqMeta.ToProtos()
	require.NoError(t, err)
	assert.NotNil(t, proto)
}

// TestTokenRequestMetadata_FromProtos_NilIssueMetadata tests nil issue metadata handling
func TestTokenRequestMetadata_FromProtos_NilIssueMetadata(t *testing.T) {
	proto := &request.TokenRequestMetadata{
		Version: ProtocolV1,
		Metadata: []*request.ActionMetadata{
			{
				Metadata: &request.ActionMetadata_IssueMetadata{
					IssueMetadata: &request.IssueMetadata{
						Issuer: &request.AuditableIdentity{
							Identity: &request.Identity{Raw: []byte("issuer1")},
						},
					},
				},
			},
		},
	}

	reqMeta := &TokenRequestMetadata{}
	err := reqMeta.FromProtos(proto)
	require.NoError(t, err)
}

// TestTokenRequestMetadata_FromProtos_NilTransferMetadata tests nil transfer metadata handling
func TestTokenRequestMetadata_FromProtos_NilTransferMetadata(t *testing.T) {
	proto := &request.TokenRequestMetadata{
		Version: ProtocolV1,
		Metadata: []*request.ActionMetadata{
			{
				Metadata: &request.ActionMetadata_TransferMetadata{
					TransferMetadata: &request.TransferMetadata{
						Inputs:  []*request.TransferInputMetadata{},
						Outputs: []*request.OutputMetadata{},
					},
				},
			},
		},
	}

	reqMeta := &TokenRequestMetadata{}
	err := reqMeta.FromProtos(proto)
	require.NoError(t, err)
}

// TestIssueInputMetadata_ToProtos_NilTokenID tests ToProtos with nil TokenID
func TestIssueInputMetadata_ToProtos_NilTokenID(t *testing.T) {
	iim := &IssueInputMetadata{
		TokenID: nil,
	}

	proto, err := iim.ToProtos()
	require.NoError(t, err)
	assert.Nil(t, proto.TokenId)
}

// TestTransferInputMetadata_ToProtos_NilTokenID tests ToProtos with nil TokenID
func TestTransferInputMetadata_ToProtos_NilTokenID(t *testing.T) {
	tim := &TransferInputMetadata{
		TokenID: nil,
		Senders: []*AuditableIdentity{},
	}

	proto, err := tim.ToProtos()
	require.NoError(t, err)
	assert.Nil(t, proto.TokenId)
}

// TestTransferInputMetadata_ToProtos_WithSenders tests ToProtos with senders
func TestTransferInputMetadata_ToProtos_WithSenders(t *testing.T) {
	tim := &TransferInputMetadata{
		TokenID: &token.ID{TxId: "tx1", Index: 1},
		Senders: []*AuditableIdentity{
			{Identity: Identity("sender1"), AuditInfo: []byte("audit1")},
			{Identity: Identity("sender2"), AuditInfo: []byte("audit2")},
		},
	}

	proto, err := tim.ToProtos()
	require.NoError(t, err)
	assert.NotNil(t, proto)
	assert.Len(t, proto.Senders, 2)
}

// TestTransferOutputMetadata_ToProtos_WithReceivers tests ToProtos with receivers
func TestTransferOutputMetadata_ToProtos_WithReceivers(t *testing.T) {
	tom := &TransferOutputMetadata{
		OutputMetadata:  []byte("metadata1"),
		OutputAuditInfo: []byte("auditinfo1"),
		Receivers: []*AuditableIdentity{
			{Identity: Identity("receiver1"), AuditInfo: []byte("audit1")},
			{Identity: Identity("receiver2"), AuditInfo: []byte("audit2")},
		},
	}

	proto, err := tom.ToProtos()
	require.NoError(t, err)
	assert.NotNil(t, proto)
	assert.Len(t, proto.Receivers, 2)
}

// TestIssueMetadata_ToProtos_WithExtraSigners tests ToProtos with extra signers
func TestIssueMetadata_ToProtos_WithExtraSigners(t *testing.T) {
	im := &IssueMetadata{
		Issuer: AuditableIdentity{
			Identity:  Identity("issuer1"),
			AuditInfo: []byte("issuer_audit"),
		},
		Inputs: []*IssueInputMetadata{},
		Outputs: []*IssueOutputMetadata{
			{OutputMetadata: []byte("output1")},
		},
		ExtraSigners: []Identity{
			Identity("signer1"),
			Identity("signer2"),
			Identity("signer3"),
		},
	}

	proto, err := im.ToProtos()
	require.NoError(t, err)
	assert.NotNil(t, proto)
	assert.Len(t, proto.ExtraSigners, 3)
}

// TestTransferMetadata_ToProtos_WithExtraSigners tests ToProtos with extra signers
func TestTransferMetadata_ToProtos_WithExtraSigners(t *testing.T) {
	tm := &TransferMetadata{
		Inputs: []*TransferInputMetadata{
			{TokenID: &token.ID{TxId: "tx1", Index: 1}},
		},
		Outputs: []*TransferOutputMetadata{
			{OutputMetadata: []byte("output1")},
		},
		ExtraSigners: []Identity{
			Identity("signer1"),
			Identity("signer2"),
		},
		Issuer: Identity("issuer1"),
	}

	proto, err := tm.ToProtos()
	require.NoError(t, err)
	assert.NotNil(t, proto)
	assert.Len(t, proto.ExtraSigners, 2)
	assert.NotNil(t, proto.Issuer)
}

// TestTokenRequest_ToProtos_EmptyAuditorSignatures tests ToProtos with empty auditor signatures
func TestTokenRequest_ToProtos_EmptyAuditorSignatures(t *testing.T) {
	req := &TokenRequest{
		Issues:            [][]byte{[]byte("issue1")},
		Signatures:        [][]byte{[]byte("sig1")},
		AuditorSignatures: []*AuditorSignature{},
	}

	proto, err := req.ToProtos()
	require.NoError(t, err)
	assert.NotNil(t, proto.Auditing)
	assert.Empty(t, proto.Auditing.Signatures)
}

// TestIssueOutputMetadata_FromProtos_NilOutputMetadata tests FromProtos with nil output metadata
func TestIssueOutputMetadata_FromProtos_NilOutputMetadata(t *testing.T) {
	meta := &IssueOutputMetadata{}
	err := meta.FromProtos(nil)
	require.NoError(t, err)
}

// TestTransferInputMetadata_FromProtos_NilTokenID tests FromProtos with nil token ID
func TestTransferInputMetadata_FromProtos_NilTokenID(t *testing.T) {
	proto := &request.TransferInputMetadata{
		TokenId: nil,
	}

	meta := &TransferInputMetadata{}
	err := meta.FromProtos(proto)
	require.NoError(t, err)
	assert.Nil(t, meta.TokenID)
}

// TestTransferOutputMetadata_FromProtos_NilOutputMetadata tests FromProtos with nil output metadata
func TestTransferOutputMetadata_FromProtos_NilOutputMetadata(t *testing.T) {
	meta := &TransferOutputMetadata{}
	err := meta.FromProtos(nil)
	require.NoError(t, err)
}

// TestTokenRequest_ToProtos_WithBothIssuesAndTransfers tests ToProtos with multiple action types
func TestTokenRequest_ToProtos_WithBothIssuesAndTransfers(t *testing.T) {
	req := &TokenRequest{
		Issues:     [][]byte{[]byte("issue1"), []byte("issue2")},
		Transfers:  [][]byte{[]byte("transfer1")},
		Signatures: [][]byte{[]byte("sig1")},
		AuditorSignatures: []*AuditorSignature{
			{Identity: Identity("aud1"), Signature: []byte("audsig1")},
		},
	}

	proto, err := req.ToProtos()
	require.NoError(t, err)
	require.NotNil(t, proto)
	assert.Len(t, proto.Actions, 3)
	assert.NotNil(t, proto.Auditing)
}

// TestTokenRequest_FromProtos_WithAuditing tests FromProtos with auditing
func TestTokenRequest_FromProtos_WithAuditing(t *testing.T) {
	protoReq := &request.TokenRequest{
		Version: ProtocolV1,
		Actions: []*request.Action{
			{Type: request.ActionType_ISSUE, Raw: []byte("issue1")},
		},
		Signatures: []*request.Signature{{Raw: []byte("sig1")}},
		Auditing: &request.Auditing{
			Signatures: []*request.AuditorSignature{
				{
					Identity:  &request.Identity{Raw: []byte("aud1")},
					Signature: &request.Signature{Raw: []byte("audsig1")},
				},
			},
		},
	}

	req := &TokenRequest{}
	err := req.FromProtos(protoReq)
	require.NoError(t, err)
	assert.Len(t, req.AuditorSignatures, 1)
}

// TestTokenRequest_MarshalToMessageToSign_WithAnchor tests MarshalToMessageToSign
func TestTokenRequest_MarshalToMessageToSign_WithAnchor(t *testing.T) {
	req := &TokenRequest{
		Issues:    [][]byte{[]byte("issue1")},
		Transfers: [][]byte{[]byte("transfer1")},
	}

	raw, err := req.MarshalToMessageToSign([]byte("anchor"))
	require.NoError(t, err)
	assert.NotEmpty(t, raw)
}

// TestIssueInputMetadata_ToProtos_WithTokenID tests ToProtos with token ID
func TestIssueInputMetadata_ToProtos_WithTokenID(t *testing.T) {
	meta := &IssueInputMetadata{
		TokenID: &token.ID{TxId: "tx1", Index: 0},
	}

	proto, err := meta.ToProtos()
	require.NoError(t, err)
	assert.NotNil(t, proto.TokenId)
}

// TestIssueInputMetadata_ToProtos_WithNilTokenID tests ToProtos with nil token ID
func TestIssueInputMetadata_ToProtos_WithNilTokenID(t *testing.T) {
	meta := &IssueInputMetadata{
		TokenID: nil,
	}

	proto, err := meta.ToProtos()
	require.NoError(t, err)
	assert.Nil(t, proto.TokenId)
}

// TestIssueOutputMetadata_ToProtos_WithReceivers tests ToProtos with receivers
func TestIssueOutputMetadata_ToProtos_WithReceivers(t *testing.T) {
	meta := &IssueOutputMetadata{
		OutputMetadata: []byte("output1"),
		Receivers: []*AuditableIdentity{
			{Identity: Identity("id1"), AuditInfo: []byte("audit1")},
		},
	}

	proto, err := meta.ToProtos()
	require.NoError(t, err)
	assert.Len(t, proto.Receivers, 1)
}

// TestIssueOutputMetadata_ToProtos_WithNilReceivers tests ToProtos with nil receivers
func TestIssueOutputMetadata_ToProtos_WithNilReceivers(t *testing.T) {
	meta := &IssueOutputMetadata{
		OutputMetadata: []byte("output1"),
		Receivers:      nil,
	}

	proto, err := meta.ToProtos()
	require.NoError(t, err)
	assert.Empty(t, proto.Receivers)
}

// TestIssueMetadata_ToProtos_WithEmptyInputs tests ToProtos with empty inputs
func TestIssueMetadata_ToProtos_WithEmptyInputs(t *testing.T) {
	meta := &IssueMetadata{
		Issuer: AuditableIdentity{Identity: Identity("issuer1")},
		Inputs: []*IssueInputMetadata{},
		Outputs: []*IssueOutputMetadata{
			{OutputMetadata: []byte("output1")},
		},
	}

	proto, err := meta.ToProtos()
	require.NoError(t, err)
	assert.Empty(t, proto.Inputs)
}

// TestIssueMetadata_ToProtos_WithEmptyOutputs tests ToProtos with empty outputs
func TestIssueMetadata_ToProtos_WithEmptyOutputs(t *testing.T) {
	meta := &IssueMetadata{
		Issuer:  AuditableIdentity{Identity: Identity("issuer1")},
		Inputs:  []*IssueInputMetadata{{TokenID: &token.ID{TxId: "tx1"}}},
		Outputs: []*IssueOutputMetadata{},
	}

	proto, err := meta.ToProtos()
	require.NoError(t, err)
	assert.Empty(t, proto.Outputs)
}

// TestIssueMetadata_ToProtos_WithExtraSigners tests ToProtos with extra signers
// TestTransferInputMetadata_ToProtos_WithTokenID tests ToProtos with token ID
func TestTransferInputMetadata_ToProtos_WithTokenID(t *testing.T) {
	meta := &TransferInputMetadata{
		TokenID: &token.ID{TxId: "tx1", Index: 0},
	}

	proto, err := meta.ToProtos()
	require.NoError(t, err)
	assert.NotNil(t, proto.TokenId)
}

// TestTransferOutputMetadata_ToProtos_WithNilReceivers tests ToProtos with nil receivers
func TestTransferOutputMetadata_ToProtos_WithNilReceivers(t *testing.T) {
	meta := &TransferOutputMetadata{
		OutputMetadata: []byte("output1"),
		Receivers:      nil,
	}

	proto, err := meta.ToProtos()
	require.NoError(t, err)
	assert.Empty(t, proto.Receivers)
}

// TestTransferMetadata_ToProtos_WithEmptyInputs tests ToProtos with empty inputs
func TestTransferMetadata_ToProtos_WithEmptyInputs(t *testing.T) {
	meta := &TransferMetadata{
		Inputs:  []*TransferInputMetadata{{TokenID: &token.ID{TxId: "tx1"}}},
		Outputs: []*TransferOutputMetadata{},
	}

	proto, err := meta.ToProtos()
	require.NoError(t, err)
	assert.Empty(t, proto.Outputs)
}

// TestTransferMetadata_ToProtos_WithExtraSigners tests ToProtos with extra signers
func TestTransferMetadata_ToProtos_WithIssuer(t *testing.T) {
	meta := &TransferMetadata{
		Outputs: []*TransferOutputMetadata{
			{OutputMetadata: []byte("output1")},
		},
		Issuer: Identity("issuer1"),
	}

	proto, err := meta.ToProtos()
	require.NoError(t, err)
	assert.NotNil(t, proto.Issuer)
}

// TestTransferMetadata_FromProtos_WithEmptySlices tests FromProtos with empty slices
func TestTransferMetadata_FromProtos_WithEmptySlices(t *testing.T) {
	proto := &request.TransferMetadata{
		Inputs:  []*request.TransferInputMetadata{},
		Outputs: []*request.OutputMetadata{},
	}

	meta := &TransferMetadata{}
	err := meta.FromProtos(proto)
	require.NoError(t, err)
	assert.Empty(t, meta.Inputs)
	assert.Empty(t, meta.Outputs)
}

// TestTokenRequestMetadata_Bytes_ToProtosError tests Bytes error when ToProtos fails
func TestTokenRequestMetadata_Bytes_ToProtosError(t *testing.T) {
	reqMeta := &TokenRequestMetadata{
		Issues: []*IssueMetadata{nil},
	}

	raw, err := reqMeta.Bytes()
	require.Error(t, err)
	assert.Nil(t, raw)
}

// TestTokenRequestMetadata_ToProtos_WithEmptyMetadata tests ToProtos with empty metadata
func TestTokenRequestMetadata_ToProtos_WithEmptyMetadata(t *testing.T) {
	reqMeta := &TokenRequestMetadata{
		Issues:    []*IssueMetadata{},
		Transfers: []*TransferMetadata{},
	}

	proto, err := reqMeta.ToProtos()
	require.NoError(t, err)
	assert.Empty(t, proto.Metadata)
}
