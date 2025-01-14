/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/test-go/testify/assert"
)

func TestMetadata_TestFilterBy(t *testing.T) {
	testCases := map[string]func(*testing.T){
		"case0": testFilterByCase0,
		"case1": testFilterByCase1,
	}
	for key, tc := range testCases {
		t.Run(key, tc)
	}
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
		Senders:            []token.Identity{token.Identity("Alice")},
		SenderAuditInfos:   [][]byte{[]byte("Alice")},
		OutputsMetadata:    [][]byte{[]byte("Bob's output's token metadata")},
		OutputsAuditInfo:   [][]byte{[]byte("Bob's output's token audit info")},
		Receivers:          []token.Identity{token.Identity("Bob")},
		ReceiverAuditInfos: [][]byte{[]byte("Bob")},
	}
	// From Charlie to Dave
	charlieToDave := driver.TransferMetadata{
		TokenIDs: []*token2.ID{
			{
				TxId:  "avocado",
				Index: 0,
			},
		},
		Senders:            []token.Identity{token.Identity("Charlie")},
		SenderAuditInfos:   [][]byte{[]byte("Charlie")},
		OutputsMetadata:    [][]byte{[]byte("Dave's output's token metadata")},
		OutputsAuditInfo:   [][]byte{[]byte("Dave's output's token audit info")},
		Receivers:          []token.Identity{token.Identity("Dave")},
		ReceiverAuditInfos: [][]byte{[]byte("Dave")},
	}
	ws := &mock.WalletService{}
	ws.GetEnrollmentIDReturnsOnCall(0, "Bob", nil)
	ws.GetEnrollmentIDReturnsOnCall(1, "Charlie", nil)
	ws.GetEnrollmentIDReturnsOnCall(2, "Bob", nil)
	ws.GetEnrollmentIDReturnsOnCall(3, "Charlie", nil)
	ws.GetEnrollmentIDReturnsOnCall(4, "Bob", nil)
	ws.GetEnrollmentIDReturnsOnCall(5, "Charlie", nil)
	ws.GetEnrollmentIDReturnsOnCall(6, "Bob", nil)
	ws.GetEnrollmentIDReturnsOnCall(7, "Charlie", nil)

	metadata := &token.Metadata{
		WalletService: ws,
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
		Logger: logging.MustGetLogger("test"),
	}
	// Filter by Bob
	filteredMetadata, err := metadata.FilterBy("Bob")
	assert.NoError(t, err)
	// assert the calls to the TMS
	assert.Equal(t, 2, ws.GetEnrollmentIDCallCount())
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
	assert.Equal(t, 4, ws.GetEnrollmentIDCallCount())
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
	assert.Equal(t, 6, ws.GetEnrollmentIDCallCount())
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
	assert.Equal(t, 8, ws.GetEnrollmentIDCallCount())
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
		Issuer:              token.Identity("Issuer"),
		OutputsMetadata:     [][]byte{[]byte("Alice's output's token info")},
		Receivers:           []token.Identity{token.Identity("Alice")},
		ReceiversAuditInfos: [][]byte{[]byte("Alice")},
	}

	// Bob's issue
	bobIssue := driver.IssueMetadata{
		Issuer:              token.Identity("Issuer"),
		OutputsMetadata:     [][]byte{[]byte("Bob's output's token info")},
		Receivers:           []token.Identity{token.Identity("Bob")},
		ReceiversAuditInfos: [][]byte{[]byte("Bob")},
	}

	ws := &mock.WalletService{}
	ws.GetEnrollmentIDReturnsOnCall(0, "Alice", nil)
	ws.GetEnrollmentIDReturnsOnCall(1, "Bob", nil)
	ws.GetEnrollmentIDReturnsOnCall(2, "Alice", nil)
	ws.GetEnrollmentIDReturnsOnCall(3, "Bob", nil)
	ws.GetEnrollmentIDReturnsOnCall(4, "Alice", nil)
	ws.GetEnrollmentIDReturnsOnCall(5, "Bob", nil)
	ws.GetEnrollmentIDReturnsOnCall(6, "Alice", nil)
	ws.GetEnrollmentIDReturnsOnCall(7, "Bob", nil)

	metadata := &token.Metadata{
		WalletService: ws,
		TokenRequestMetadata: &driver.TokenRequestMetadata{
			Issues: []driver.IssueMetadata{
				aliceIssue,
				bobIssue,
			},
		},
		Logger: logging.MustGetLogger("test"),
	}

	// Filter by Alice
	filteredMetadata, err := metadata.FilterBy("Alice")
	assert.NoError(t, err)
	// assert the calls to the TMS
	assert.Equal(t, 2, ws.GetEnrollmentIDCallCount())
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
	assert.Equal(t, 4, ws.GetEnrollmentIDCallCount())
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
	assert.Equal(t, 6, ws.GetEnrollmentIDCallCount())
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
	assert.Equal(t, 8, ws.GetEnrollmentIDCallCount())
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
	assert.Equal(t, original.OutputsMetadata, filtered.OutputsMetadata)
	assert.Equal(t, original.Receivers, filtered.Receivers)
}

func assertEmptyIssueMetadata(t *testing.T, original, filtered *driver.IssueMetadata) {
	// check equal issuer
	assert.Equal(t, original.Issuer, filtered.Issuer)
	// assert that the lengths are the same
	assert.Len(t, original.OutputsMetadata, len(filtered.OutputsMetadata))
	assert.Len(t, original.Receivers, len(filtered.Receivers))
	assert.Len(t, original.ReceiversAuditInfos, len(filtered.ReceiversAuditInfos))

	// assert that the token info is empty
	for i := 0; i < len(original.OutputsMetadata); i++ {
		assert.Empty(t, filtered.OutputsMetadata[i])
	}
	// assert that the receivers are empty
	for i := 0; i < len(original.Receivers); i++ {
		assert.Equal(t, original.Receivers[i], filtered.Receivers[i])
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
	assert.Len(t, original.OutputsMetadata, len(filtered.OutputsMetadata))
	assert.Len(t, original.Receivers, len(filtered.Receivers))
	assert.Len(t, original.ReceiverAuditInfos, len(filtered.ReceiverAuditInfos))
	// assert each token info is empty
	for i := 0; i < len(original.OutputsMetadata); i++ {
		assert.Nil(t, filtered.OutputsMetadata[i])
	}
	// assert each receiver is empty
	for i := 0; i < len(original.Receivers); i++ {
		assert.NotNil(t, filtered.Receivers[i])
	}
	// assert each receiver audit info is empty
	for i := 0; i < len(original.ReceiverAuditInfos); i++ {
		assert.Nil(t, filtered.ReceiverAuditInfos[i])
	}
}

func assertEqualTransferMetadata(t *testing.T, original, filtered *driver.TransferMetadata) {
	assert.Nil(t, filtered.TokenIDs)
	assert.Equal(t, original.Senders, filtered.Senders)
	assert.Equal(t, original.SenderAuditInfos, filtered.SenderAuditInfos)
	assert.Equal(t, original.OutputsMetadata, filtered.OutputsMetadata)
	assert.Equal(t, original.Receivers, filtered.Receivers)
	assert.Equal(t, original.ReceiverAuditInfos, filtered.ReceiverAuditInfos)
}
