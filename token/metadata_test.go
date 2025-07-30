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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
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
	t.Helper()
	// Case:
	// - Two transfers. One from Alice to Bob and one from Charlie to Dave.

	// From Alice to Bob
	aliceToBob := &driver.TransferMetadata{
		Inputs: []*driver.TransferInputMetadata{
			{
				TokenID: &token2.ID{
					TxId:  "pineapple",
					Index: 0,
				},
				Senders: []*driver.AuditableIdentity{
					{
						Identity:  token.Identity("Alice"),
						AuditInfo: []byte("Alice"),
					},
				},
			},
		},
		Outputs: []*driver.TransferOutputMetadata{
			{
				OutputMetadata:  []byte("Bob's output's token metadata"),
				OutputAuditInfo: []byte("Bob's output's token audit info"),
				Receivers: []*driver.AuditableIdentity{
					{
						Identity:  token.Identity("Bob"),
						AuditInfo: []byte("Bob"),
					},
				},
			},
		},
		ExtraSigners: nil,
	}
	// From Charlie to Dave
	charlieToDave := &driver.TransferMetadata{
		Inputs: []*driver.TransferInputMetadata{
			{
				TokenID: &token2.ID{
					TxId:  "avocado",
					Index: 0,
				},
				Senders: []*driver.AuditableIdentity{
					{
						Identity:  token.Identity("Charlie"),
						AuditInfo: []byte("Charlie"),
					},
				},
			},
		},
		Outputs: []*driver.TransferOutputMetadata{
			{
				OutputMetadata:  []byte("Dave's output's token metadata"),
				OutputAuditInfo: []byte("Dave's output's token audit info"),
				Receivers: []*driver.AuditableIdentity{
					{
						Identity:  token.Identity("Dave"),
						AuditInfo: []byte("Dave"),
					},
				},
			},
		},
		ExtraSigners: nil,
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
			Transfers: []*driver.TransferMetadata{
				aliceToBob,
				charlieToDave,
			},
			Application: map[string][]byte{
				"application": []byte("application"),
			},
		},
		Logger: logging.MustGetLogger(),
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
	assertEqualTransferMetadata(t, aliceToBob, filteredMetadata.TokenRequestMetadata.Transfers[0], true)
	assertEmptyTransferMetadata(t, charlieToDave, filteredMetadata.TokenRequestMetadata.Transfers[1])

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
	assertEmptyTransferMetadata(t, aliceToBob, filteredMetadata.TokenRequestMetadata.Transfers[0])
	assertEqualTransferMetadata(t, charlieToDave, filteredMetadata.TokenRequestMetadata.Transfers[1], true)

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
	assertEmptyTransferMetadata(t, aliceToBob, filteredMetadata.TokenRequestMetadata.Transfers[0])
	assertEmptyTransferMetadata(t, charlieToDave, filteredMetadata.TokenRequestMetadata.Transfers[1])

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
	assertEqualTransferMetadata(t, aliceToBob, filteredMetadata.TokenRequestMetadata.Transfers[0], true)
	assertEqualTransferMetadata(t, charlieToDave, filteredMetadata.TokenRequestMetadata.Transfers[1], true)

	// No Filter
	filteredMetadata, err = metadata.FilterBy()
	assert.NoError(t, err)
	// assert the calls to the TMS
	assert.Equal(t, 8, ws.GetEnrollmentIDCallCount())
	// Check no issues were returned
	assert.Len(t, filteredMetadata.TokenRequestMetadata.Issues, 0)
	// Check that the application metadata is not filtered
	assert.Equal(t, metadata.TokenRequestMetadata.Application, filteredMetadata.TokenRequestMetadata.Application)
	// Check that the transfers are filtered
	assert.Len(t, filteredMetadata.TokenRequestMetadata.Transfers, 2)
	assertEqualTransferMetadata(t, aliceToBob, filteredMetadata.TokenRequestMetadata.Transfers[0], false)
	assertEqualTransferMetadata(t, charlieToDave, filteredMetadata.TokenRequestMetadata.Transfers[1], false)
}

func testFilterByCase1(t *testing.T) {
	t.Helper()
	// Case:
	// - Two issues. One for Alice and one for Bob.

	// Alice's issue
	aliceIssue := &driver.IssueMetadata{
		Issuer: driver.AuditableIdentity{
			Identity:  token.Identity("Issuer"),
			AuditInfo: []byte("Issuer Audit Info"),
		},
		Inputs: nil,
		Outputs: []*driver.IssueOutputMetadata{
			{
				OutputMetadata: []byte("Alice's output's token info"),
				Receivers: []*driver.AuditableIdentity{
					{
						Identity:  token.Identity("Alice"),
						AuditInfo: []byte("Alice"),
					},
				},
			},
		},
		ExtraSigners: nil,
	}
	// Bob's issue
	bobIssue := &driver.IssueMetadata{
		Issuer: driver.AuditableIdentity{
			Identity:  token.Identity("Issuer"),
			AuditInfo: []byte("Issuer Audit Info"),
		},
		Inputs: nil,
		Outputs: []*driver.IssueOutputMetadata{
			{
				OutputMetadata: []byte("Bob's output's token info"),
				Receivers: []*driver.AuditableIdentity{
					{
						Identity:  token.Identity("Bob"),
						AuditInfo: []byte("Bob"),
					},
				},
			},
		},
		ExtraSigners: nil,
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
			Issues: []*driver.IssueMetadata{
				aliceIssue,
				bobIssue,
			},
		},
		Logger: logging.MustGetLogger(),
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
	assertEqualIssueMetadata(t, aliceIssue, filteredMetadata.TokenRequestMetadata.Issues[0])
	assertEmptyIssueMetadata(t, bobIssue, filteredMetadata.TokenRequestMetadata.Issues[1])

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
	assertEmptyIssueMetadata(t, aliceIssue, filteredMetadata.TokenRequestMetadata.Issues[0])
	assertEqualIssueMetadata(t, bobIssue, filteredMetadata.TokenRequestMetadata.Issues[1])

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
	assertEmptyIssueMetadata(t, aliceIssue, filteredMetadata.TokenRequestMetadata.Issues[0])
	assertEmptyIssueMetadata(t, bobIssue, filteredMetadata.TokenRequestMetadata.Issues[1])

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
	assertEqualIssueMetadata(t, aliceIssue, filteredMetadata.TokenRequestMetadata.Issues[0])
	assertEqualIssueMetadata(t, bobIssue, filteredMetadata.TokenRequestMetadata.Issues[1])

	// No Filter
	filteredMetadata, err = metadata.FilterBy()
	assert.NoError(t, err)
	// assert the calls to the TMS
	assert.Equal(t, 8, ws.GetEnrollmentIDCallCount())
	// Check no transfers were returned
	assert.Len(t, filteredMetadata.TokenRequestMetadata.Transfers, 0)
	// Check that the application metadata is not filtered
	assert.Equal(t, metadata.TokenRequestMetadata.Application, filteredMetadata.TokenRequestMetadata.Application)
	// Check that the issues are filtered
	assert.Len(t, filteredMetadata.TokenRequestMetadata.Issues, 2)
	assertEqualIssueMetadata(t, aliceIssue, filteredMetadata.TokenRequestMetadata.Issues[0])
	assertEqualIssueMetadata(t, bobIssue, filteredMetadata.TokenRequestMetadata.Issues[1])
}

func assertEqualIssueMetadata(t *testing.T, original, filtered *driver.IssueMetadata) {
	t.Helper()
	assert.Equal(t, original, filtered)
}

func assertEmptyIssueMetadata(t *testing.T, original, filtered *driver.IssueMetadata) {
	t.Helper()
	// check equal issuer
	assert.Equal(t, original.Issuer, filtered.Issuer)
	// assert that the lengths are the same
	assert.Len(t, original.Outputs, len(filtered.Outputs))

	// assert that the token info is empty
	for i := range len(original.Outputs) {
		assert.Empty(t, filtered.Outputs[i])
	}
}

func assertEmptyTransferMetadata(t *testing.T, original, filtered *driver.TransferMetadata) {
	t.Helper()
	// assert tokenIDs, senders and senderAuditInfos are empty
	for _, input := range filtered.Inputs {
		assert.Nil(t, input.TokenID)
		assert.Nil(t, input.Senders)
	}

	// assert that the lengths are the same
	assert.Len(t, original.Outputs, len(filtered.Outputs))
	for i := range len(original.Outputs) {
		assert.Empty(t, filtered.Outputs[i])
	}
}

func assertEqualTransferMetadata(t *testing.T, original, filtered *driver.TransferMetadata, noInputs bool) {
	t.Helper()
	for i, input := range original.Inputs {
		if noInputs {
			assert.Nil(t, filtered.Inputs[i].TokenID)
		} else {
			assert.Equal(t, input.TokenID, filtered.Inputs[i].TokenID)
		}
		assert.Equal(t, input.Senders, filtered.Inputs[i].Senders)
	}
	assert.Equal(t, original.Outputs, filtered.Outputs)
	assert.Equal(t, original.ExtraSigners, filtered.ExtraSigners)
}

func TestMetadata_TestMatchTransferAction(t *testing.T) {
	transferActionWithIssuer := &mock.TransferAction{}
	mockIssuer := identity.Identity{0x1, 0x2, 0x3}
	transferActionWithIssuer.GetIssuerReturns(mockIssuer)

	tests := []struct {
		name          string
		action        *token.TransferAction
		meta          *token.TransferMetadata
		wantErr       bool
		expectedError string
	}{
		{
			name:   "action and meta with issuer",
			action: &token.TransferAction{transferActionWithIssuer},
			meta: &token.TransferMetadata{
				&driver.TransferMetadata{
					Issuer: mockIssuer,
				},
			},

			wantErr: false,
		},
		{
			name:   "error: meta with no issuer",
			action: &token.TransferAction{transferActionWithIssuer},
			meta: &token.TransferMetadata{
				&driver.TransferMetadata{
					Issuer: nil,
				},
			},

			wantErr:       true,
			expectedError: "expected issuer [<empty>] but got [\x01\x02\x03]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.meta.Match(tt.action)
			if tt.wantErr {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
