/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/token"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/stretchr/testify/assert"
)

func TestIssueAction_Serialize(t *testing.T) {
	mockIssueAction := &mock.IssueAction{}
	mockIssueAction.SerializeReturns([]byte{1, 2, 3}, nil)
	issueAction := &IssueAction{a: mockIssueAction}
	serialized, err := issueAction.Serialize()
	assert.NoError(t, err)
	assert.Equal(t, []byte{1, 2, 3}, serialized)
}

func TestIssueAction_NumOutputs(t *testing.T) {
	mockIssueAction := &mock.IssueAction{}
	mockIssueAction.NumOutputsReturns(5)
	issueAction := &IssueAction{a: mockIssueAction}
	numOutputs := issueAction.NumOutputs()
	assert.Equal(t, 5, numOutputs)
}

func TestIssueAction_GetSerializedOutputs(t *testing.T) {
	mockIssueAction := &mock.IssueAction{}
	mockSerializedOutputs := [][]byte{{1, 2}, {3, 4}}
	mockIssueAction.GetSerializedOutputsReturns(mockSerializedOutputs, nil)
	issueAction := &IssueAction{a: mockIssueAction}
	serializedOutputs, err := issueAction.GetSerializedOutputs()
	assert.NoError(t, err)
	assert.Equal(t, mockSerializedOutputs, serializedOutputs)
}

func TestIssueAction_IsAnonymous(t *testing.T) {
	mockIssueAction := &mock.IssueAction{}
	mockIssueAction.IsAnonymousReturns(true)
	issueAction := &IssueAction{a: mockIssueAction}
	isAnonymous := issueAction.IsAnonymous()
	assert.True(t, isAnonymous)
}

func TestIssueAction_GetIssuer(t *testing.T) {
	mockIssueAction := &mock.IssueAction{}
	mockIssuer := []byte{1, 2, 3}
	mockIssueAction.GetIssuerReturns(mockIssuer)
	issueAction := &IssueAction{a: mockIssueAction}
	issuer := issueAction.GetIssuer()
	assert.Equal(t, mockIssuer, issuer)
}

func TestIssueAction_GetMetadata(t *testing.T) {
	mockIssueAction := &mock.IssueAction{}
	mockMetadata := map[string][]byte{"key1": {1, 2, 3}, "key2": {4, 5, 6}}
	mockIssueAction.GetMetadataReturns(mockMetadata)
	issueAction := &IssueAction{a: mockIssueAction}
	metadata := issueAction.GetMetadata()
	assert.Equal(t, mockMetadata, metadata)
}

func TestTransferAction_Serialize(t *testing.T) {
	mockTransferAction := &mock.TransferAction{}
	mockTransferAction.SerializeReturns([]byte{1, 2, 3}, nil)
	transferAction := &TransferAction{a: mockTransferAction}
	serialized, err := transferAction.Serialize()
	assert.NoError(t, err)
	assert.Equal(t, []byte{1, 2, 3}, serialized)
}

func TestTransferAction_NumOutputs(t *testing.T) {
	mockTransferAction := &mock.TransferAction{}
	mockTransferAction.NumOutputsReturns(5)
	transferAction := &TransferAction{a: mockTransferAction}
	numOutputs := transferAction.NumOutputs()
	assert.Equal(t, 5, numOutputs)
}

func TestTransferAction_GetSerializedOutputs(t *testing.T) {
	mockTransferAction := &mock.TransferAction{}
	mockSerializedOutputs := [][]byte{{1, 2}, {3, 4}}
	mockTransferAction.GetSerializedOutputsReturns(mockSerializedOutputs, nil)
	transferAction := &TransferAction{a: mockTransferAction}
	serializedOutputs, err := transferAction.GetSerializedOutputs()
	assert.NoError(t, err)
	assert.Equal(t, mockSerializedOutputs, serializedOutputs)
}

func TestTransferAction_IsRedeemAt(t *testing.T) {
	mockTransferAction := &mock.TransferAction{}
	mockTransferAction.IsRedeemAtReturns(true)
	transferAction := &TransferAction{a: mockTransferAction}
	isRedeemAt := transferAction.IsRedeemAt(0)
	assert.True(t, isRedeemAt)
}

func TestTransferAction_SerializeOutputAt(t *testing.T) {
	mockTransferAction := &mock.TransferAction{}
	mockSerializedOutput := []byte{1, 2, 3}
	mockTransferAction.SerializeOutputAtReturns(mockSerializedOutput, nil)
	transferAction := &TransferAction{a: mockTransferAction}
	serializedOutput, err := transferAction.SerializeOutputAt(0)
	assert.NoError(t, err)
	assert.Equal(t, mockSerializedOutput, serializedOutput)
}

func TestTransferAction_GetInputs(t *testing.T) {
	mockTransferAction := &mock.TransferAction{}
	mockInputs := []*token.ID{{TxId: "input1"}, {TxId: "input2"}}
	mockTransferAction.GetInputsReturns(mockInputs, nil)
	transferAction := &TransferAction{a: mockTransferAction}
	inputs, err := transferAction.GetInputs()
	assert.NoError(t, err)
	assert.Equal(t, mockInputs, inputs)
}

func TestTransferAction_IsGraphHiding(t *testing.T) {
	mockTransferAction := &mock.TransferAction{}
	mockTransferAction.IsGraphHidingReturns(true)
	transferAction := &TransferAction{a: mockTransferAction}
	isGraphHiding := transferAction.IsGraphHiding()
	assert.True(t, isGraphHiding)
}
