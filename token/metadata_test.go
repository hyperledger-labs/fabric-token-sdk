/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/mock"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/test-go/testify/assert"
)

//go:generate counterfeiter -o mock/tms.go -fake-name TMS . TMS

type TMS interface {
	token.TMS
}

func TestFilterBy(t *testing.T) {
	testFilterByCase0(t)
	testFilterByCase1(t)
}

func testFilterByCase0(t *testing.T) {
	// Case:
	// - Two transfers. One from Alice to Bob and one from Charlie to Dave.

	// From Alice to Bob
	aliceToBob := driver.TransferMetadata{
		TokenIDs: []*token2.ID{
			{
				TxId:  "pineapple",
				Index: 0,
			},
		},
		Senders:            []view.Identity{view.Identity("Alice")},
		SenderAuditInfos:   [][]byte{[]byte("Alice")},
		Outputs:            [][]byte{[]byte("Bob's output")},
		TokenInfo:          [][]byte{[]byte("Bob's output's token info")},
		Receivers:          []view.Identity{view.Identity("Bob")},
		ReceiverAuditInfos: [][]byte{[]byte("Bob")},
		ReceiverIsSender:   []bool{false},
	}
	// From Charlie to Dave
	charlieToDave := driver.TransferMetadata{
		TokenIDs: []*token2.ID{
			{
				TxId:  "avocado",
				Index: 0,
			},
		},
		Senders:            []view.Identity{view.Identity("Charlie")},
		SenderAuditInfos:   [][]byte{[]byte("Charlie")},
		Outputs:            [][]byte{[]byte("Dave's output")},
		TokenInfo:          [][]byte{[]byte("Dave's output's token info")},
		Receivers:          []view.Identity{view.Identity("Dave")},
		ReceiverAuditInfos: [][]byte{[]byte("Dave")},
		ReceiverIsSender:   []bool{false},
	}
	tms := &mock.TMS{}
	tms.GetEnrollmentIDReturnsOnCall(0, "Bob", nil)
	tms.GetEnrollmentIDReturnsOnCall(1, "Charlie", nil)
	tms.GetEnrollmentIDReturnsOnCall(2, "Bob", nil)
	tms.GetEnrollmentIDReturnsOnCall(3, "Charlie", nil)
	tms.GetEnrollmentIDReturnsOnCall(4, "Bob", nil)
	tms.GetEnrollmentIDReturnsOnCall(5, "Charlie", nil)
	tms.GetEnrollmentIDReturnsOnCall(6, "Bob", nil)
	tms.GetEnrollmentIDReturnsOnCall(7, "Charlie", nil)

	metadata := &token.Metadata{
		TMS: tms,
		TokenRequestMetadata: &driver.TokenRequestMetadata{
			Issues: nil,
			Transfers: []driver.TransferMetadata{
				aliceToBob,
				charlieToDave,
			},
			Application: map[string][]byte{
				"application": []byte("application"),
			},
		},
	}
	// Filter by Bob
	filteredMetadata, err := metadata.FilterBy("Bob")
	assert.NoError(t, err)
	// assert the calls to the TMS
	assert.Equal(t, 2, tms.GetEnrollmentIDCallCount())
	// Check no issues were returned
	assert.Len(t, filteredMetadata.TokenRequestMetadata.Issues, 0)
	// Check that the application metadata is not filtered
	assert.Equal(t, metadata.TokenRequestMetadata.Application, filteredMetadata.TokenRequestMetadata.Application)
	// Check that the transfers are filtered
	assert.Len(t, filteredMetadata.TokenRequestMetadata.Transfers, 2)
	assertEqualTransferMetadata(t, &aliceToBob, &filteredMetadata.TokenRequestMetadata.Transfers[0])
	assertEmptyTransferMetadata(t, &charlieToDave, &filteredMetadata.TokenRequestMetadata.Transfers[1])

	// Filter by Charlie
	filteredMetadata, err = metadata.FilterBy("Charlie")
	assert.NoError(t, err)
	// assert the calls to the TMS
	assert.Equal(t, 4, tms.GetEnrollmentIDCallCount())
	// Check no issues were returned
	assert.Len(t, filteredMetadata.TokenRequestMetadata.Issues, 0)
	// Check that the application metadata is not filtered
	assert.Equal(t, metadata.TokenRequestMetadata.Application, filteredMetadata.TokenRequestMetadata.Application)
	// Check that the transfers are filtered
	assert.Len(t, filteredMetadata.TokenRequestMetadata.Transfers, 2)
	assertEmptyTransferMetadata(t, &aliceToBob, &filteredMetadata.TokenRequestMetadata.Transfers[0])
	assertEqualTransferMetadata(t, &charlieToDave, &filteredMetadata.TokenRequestMetadata.Transfers[1])

	// Filter by Eve
	filteredMetadata, err = metadata.FilterBy("Eve")
	assert.NoError(t, err)
	// assert the calls to the TMS
	assert.Equal(t, 6, tms.GetEnrollmentIDCallCount())
	// Check no issues were returned
	assert.Len(t, filteredMetadata.TokenRequestMetadata.Issues, 0)
	// Check that the application metadata is not filtered
	assert.Equal(t, metadata.TokenRequestMetadata.Application, filteredMetadata.TokenRequestMetadata.Application)
	// Check that the transfers are filtered
	assert.Len(t, filteredMetadata.TokenRequestMetadata.Transfers, 2)
	assertEmptyTransferMetadata(t, &aliceToBob, &filteredMetadata.TokenRequestMetadata.Transfers[0])
	assertEmptyTransferMetadata(t, &charlieToDave, &filteredMetadata.TokenRequestMetadata.Transfers[1])

	// Filter by Bob and Charlie
	filteredMetadata, err = metadata.FilterBy("Bob", "Charlie")
	assert.NoError(t, err)
	// assert the calls to the TMS
	assert.Equal(t, 8, tms.GetEnrollmentIDCallCount())
	// Check no issues were returned
	assert.Len(t, filteredMetadata.TokenRequestMetadata.Issues, 0)
	// Check that the application metadata is not filtered
	assert.Equal(t, metadata.TokenRequestMetadata.Application, filteredMetadata.TokenRequestMetadata.Application)
	// Check that the transfers are filtered
	assert.Len(t, filteredMetadata.TokenRequestMetadata.Transfers, 2)
	assertEqualTransferMetadata(t, &aliceToBob, &filteredMetadata.TokenRequestMetadata.Transfers[0])
	assertEqualTransferMetadata(t, &charlieToDave, &filteredMetadata.TokenRequestMetadata.Transfers[1])
}

func testFilterByCase1(t *testing.T) {
	// Case:
	// - Two issues. One for Alice and one for Bob.

	// Alice's issue
	aliceIssue := driver.IssueMetadata{
		Issuer:              view.Identity("Issuer"),
		Outputs:             [][]byte{[]byte("Alice's output")},
		TokenInfo:           [][]byte{[]byte("Alice's output's token info")},
		Receivers:           []view.Identity{view.Identity("Alice")},
		ReceiversAuditInfos: [][]byte{[]byte("Alice")},
	}

	// Bob's issue
	bobIssue := driver.IssueMetadata{
		Issuer:              view.Identity("Issuer"),
		Outputs:             [][]byte{[]byte("Bob's output")},
		TokenInfo:           [][]byte{[]byte("Bob's output's token info")},
		Receivers:           []view.Identity{view.Identity("Bob")},
		ReceiversAuditInfos: [][]byte{[]byte("Bob")},
	}

	tms := &mock.TMS{}
	tms.GetEnrollmentIDReturnsOnCall(0, "Alice", nil)
	tms.GetEnrollmentIDReturnsOnCall(1, "Bob", nil)
	tms.GetEnrollmentIDReturnsOnCall(2, "Alice", nil)
	tms.GetEnrollmentIDReturnsOnCall(3, "Bob", nil)
	tms.GetEnrollmentIDReturnsOnCall(4, "Alice", nil)
	tms.GetEnrollmentIDReturnsOnCall(5, "Bob", nil)
	tms.GetEnrollmentIDReturnsOnCall(6, "Alice", nil)
	tms.GetEnrollmentIDReturnsOnCall(7, "Bob", nil)

	metadata := &token.Metadata{
		TMS: tms,
		TokenRequestMetadata: &driver.TokenRequestMetadata{
			Issues: []driver.IssueMetadata{
				aliceIssue,
				bobIssue,
			},
		},
	}

	// Filter by Alice
	filteredMetadata, err := metadata.FilterBy("Alice")
	assert.NoError(t, err)
	// assert the calls to the TMS
	assert.Equal(t, 2, tms.GetEnrollmentIDCallCount())
	// Check no transfers were returned
	assert.Len(t, filteredMetadata.TokenRequestMetadata.Transfers, 0)
	// Check that the application metadata is not filtered
	assert.Equal(t, metadata.TokenRequestMetadata.Application, filteredMetadata.TokenRequestMetadata.Application)
	// Check that the issues are filtered
	assert.Len(t, filteredMetadata.TokenRequestMetadata.Issues, 2)
	assertEqualIssueMetadata(t, &aliceIssue, &filteredMetadata.TokenRequestMetadata.Issues[0])
	assertEmptyIssueMetadata(t, &bobIssue, &filteredMetadata.TokenRequestMetadata.Issues[1])

	// Filter by Bob
	filteredMetadata, err = metadata.FilterBy("Bob")
	assert.NoError(t, err)
	// assert the calls to the TMS
	assert.Equal(t, 4, tms.GetEnrollmentIDCallCount())
	// Check no transfers were returned
	assert.Len(t, filteredMetadata.TokenRequestMetadata.Transfers, 0)
	// Check that the application metadata is not filtered
	assert.Equal(t, metadata.TokenRequestMetadata.Application, filteredMetadata.TokenRequestMetadata.Application)
	// Check that the issues are filtered
	assert.Len(t, filteredMetadata.TokenRequestMetadata.Issues, 2)
	assertEmptyIssueMetadata(t, &aliceIssue, &filteredMetadata.TokenRequestMetadata.Issues[0])
	assertEqualIssueMetadata(t, &bobIssue, &filteredMetadata.TokenRequestMetadata.Issues[1])

	// Filter by Charlie
	filteredMetadata, err = metadata.FilterBy("Charlie")
	assert.NoError(t, err)
	// assert the calls to the TMS
	assert.Equal(t, 6, tms.GetEnrollmentIDCallCount())
	// Check no transfers were returned
	assert.Len(t, filteredMetadata.TokenRequestMetadata.Transfers, 0)
	// Check that the application metadata is not filtered
	assert.Equal(t, metadata.TokenRequestMetadata.Application, filteredMetadata.TokenRequestMetadata.Application)
	// Check that the issues are filtered
	assert.Len(t, filteredMetadata.TokenRequestMetadata.Issues, 2)
	assertEmptyIssueMetadata(t, &aliceIssue, &filteredMetadata.TokenRequestMetadata.Issues[0])
	assertEmptyIssueMetadata(t, &bobIssue, &filteredMetadata.TokenRequestMetadata.Issues[1])

	// Filter by Alice and Bob
	filteredMetadata, err = metadata.FilterBy("Alice", "Bob")
	assert.NoError(t, err)
	// assert the calls to the TMS
	assert.Equal(t, 8, tms.GetEnrollmentIDCallCount())
	// Check no transfers were returned
	assert.Len(t, filteredMetadata.TokenRequestMetadata.Transfers, 0)
	// Check that the application metadata is not filtered
	assert.Equal(t, metadata.TokenRequestMetadata.Application, filteredMetadata.TokenRequestMetadata.Application)
	// Check that the issues are filtered
	assert.Len(t, filteredMetadata.TokenRequestMetadata.Issues, 2)
	assertEqualIssueMetadata(t, &aliceIssue, &filteredMetadata.TokenRequestMetadata.Issues[0])
	assertEqualIssueMetadata(t, &bobIssue, &filteredMetadata.TokenRequestMetadata.Issues[1])
}

func assertEqualIssueMetadata(t *testing.T, original, filtered *driver.IssueMetadata) {
	assert.Equal(t, original.Issuer, filtered.Issuer)
	assert.Equal(t, original.Outputs, filtered.Outputs)
	assert.Equal(t, original.TokenInfo, filtered.TokenInfo)
	assert.Equal(t, original.Receivers, filtered.Receivers)
}

func assertEmptyIssueMetadata(t *testing.T, original, filtered *driver.IssueMetadata) {
	// check equal issuer
	assert.Equal(t, original.Issuer, filtered.Issuer)
	// assert that the lengths are the same
	assert.Len(t, original.Outputs, len(filtered.Outputs))
	assert.Len(t, original.TokenInfo, len(filtered.TokenInfo))
	assert.Len(t, original.Receivers, len(filtered.Receivers))
	assert.Len(t, original.ReceiversAuditInfos, len(filtered.ReceiversAuditInfos))

	// assert that the outputs are empty
	for i := 0; i < len(original.Outputs); i++ {
		assert.Empty(t, filtered.Outputs[i])
	}
	// assert that the token info is empty
	for i := 0; i < len(original.TokenInfo); i++ {
		assert.Empty(t, filtered.TokenInfo[i])
	}
	// assert that the receivers are empty
	for i := 0; i < len(original.Receivers); i++ {
		assert.Empty(t, filtered.Receivers[i])
	}
	// assert that the receivers audit infos are empty
	for i := 0; i < len(original.ReceiversAuditInfos); i++ {
		assert.Empty(t, filtered.ReceiversAuditInfos[i])
	}
}

func assertEmptyTransferMetadata(t *testing.T, original, filtered *driver.TransferMetadata) {
	// assert tokenIDs, senders and senderAuditInfos are empty
	assert.Nil(t, filtered.TokenIDs)
	assert.Nil(t, filtered.Senders)
	assert.Nil(t, filtered.SenderAuditInfos)

	// assert that the lengths are the same
	assert.Len(t, original.Outputs, len(filtered.Outputs))
	assert.Len(t, original.TokenInfo, len(filtered.TokenInfo))
	assert.Len(t, original.Receivers, len(filtered.Receivers))
	assert.Len(t, original.ReceiverAuditInfos, len(filtered.ReceiverAuditInfos))
	assert.Len(t, original.ReceiverIsSender, len(filtered.ReceiverIsSender))
	// assert each output is empty
	for i := 0; i < len(original.Outputs); i++ {
		assert.Nil(t, filtered.Outputs[i])
	}
	// assert each token info is empty
	for i := 0; i < len(original.TokenInfo); i++ {
		assert.Nil(t, filtered.TokenInfo[i])
	}
	// assert each receiver is empty
	for i := 0; i < len(original.Receivers); i++ {
		assert.Nil(t, filtered.Receivers[i])
	}
	// assert each receiver audit info is empty
	for i := 0; i < len(original.ReceiverAuditInfos); i++ {
		assert.Nil(t, filtered.ReceiverAuditInfos[i])
	}
	// assert each receiver is sender is false
	for i := 0; i < len(original.ReceiverIsSender); i++ {
		assert.False(t, filtered.ReceiverIsSender[i])
	}
}

func assertEqualTransferMetadata(t *testing.T, original, filtered *driver.TransferMetadata) {
	assert.Nil(t, filtered.TokenIDs)
	assert.Equal(t, original.Senders, filtered.Senders)
	assert.Equal(t, original.SenderAuditInfos, filtered.SenderAuditInfos)
	assert.Equal(t, original.Outputs, filtered.Outputs)
	assert.Equal(t, original.TokenInfo, filtered.TokenInfo)
	assert.Equal(t, original.Receivers, filtered.Receivers)
	assert.Equal(t, original.ReceiverAuditInfos, filtered.ReceiverAuditInfos)
}
