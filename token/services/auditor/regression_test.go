/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditor_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	drivermock "github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	tokenmock "github.com/hyperledger-labs/fabric-token-sdk/token/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	tokenpkg "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockOutput is a dummy output for testing
type mockOutput struct{}

func (m *mockOutput) Serialize() ([]byte, error) { return []byte("output"), nil }
func (m *mockOutput) IsRedeem() bool             { return false }
func (m *mockOutput) GetOwner() []byte           { return []byte("owner") }

// TestMetadataRegression verifies that token requests from all supported protocol versions can be correctly unmarshalled and audited.
func TestMetadataRegression(t *testing.T) {
	testcases := []struct {
		name    string
		fixture string
		version uint32
	}{
		{name: "Protocol V1", fixture: "v1.bin", version: driver.ProtocolV1},
		{name: "Protocol V2", fixture: "v2.bin", version: driver.ProtocolV2},
		{name: "Protocol V3", fixture: "v3.bin", version: driver.ProtocolV3},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", "regression", tc.fixture))
			require.NoError(t, err)

			// Mock TMS to satisfy NewFullRequestFromBytes and AuditCheck
			mockTMS := &drivermock.TokenManagerService{}
			mockPPM := &drivermock.PublicParamsManager{}
			mockPP := &drivermock.PublicParameters{}
			mockPP.PrecisionReturns(64)
			mockPP.MaxTokenValueReturns(1000000)
			mockPPM.PublicParametersReturns(mockPP)
			mockTMS.PublicParamsManagerReturns(mockPPM)

			mockIssueService := &drivermock.IssueService{}
			mockTransferService := &drivermock.TransferService{}
			mockWalletService := &drivermock.WalletService{}
			mockTokensService := &drivermock.TokensService{}
			mockTMS.IssueServiceReturns(mockIssueService)
			mockTMS.TransferServiceReturns(mockTransferService)
			mockTMS.WalletServiceReturns(mockWalletService)
			mockTMS.TokensServiceReturns(mockTokensService)
			mockTMS.ValidatorReturns(&drivermock.Validator{}, nil)

			// Stub deserialization to avoid nil panics
			mockIssueAction := &drivermock.IssueAction{}
			mockIssueService.DeserializeIssueActionReturns(mockIssueAction, nil)
			mockTransferService.DeserializeTransferActionReturns(&drivermock.TransferAction{}, nil)

			// Stub Deobfuscate to return dummy data (will be updated inside loop)
			mockTokensService.DeobfuscateReturns(&tokenpkg.Token{Quantity: "10"}, driver.Identity("issuer"), []driver.Identity{driver.Identity("receiver")}, "format", nil)

			mockVP := &tokenmock.VaultProvider{}
			mockV := &drivermock.Vault{}
			mockV.QueryEngineReturns(&drivermock.QueryEngine{})
			mockVP.VaultReturns(mockV, nil)

			tms, err := token.NewManagementService(token.TMSID{}, mockTMS, logging.MustGetLogger("test"), mockVP, nil, nil)
			require.NoError(t, err)

			// Setup mock actions based on protocol version to pass Match()
			if tc.version == driver.ProtocolV3 {
				mockIssueAction.NumOutputsReturns(0)
				mockIssueAction.ExtraSignersReturns([]driver.Identity{driver.Identity("extra-v3")})
				mockIssueAction.GetIssuerReturns([]byte("issuer-v3"))
			} else {
				mockIssueAction.NumOutputsReturns(1)
				mockIssueAction.GetSerializedOutputsReturns([][]byte{[]byte("serialized-output")}, nil)
				mockIssueAction.GetOutputsReturns([]driver.Output{&mockOutput{}})
				if tc.fixture == "v1.bin" {
					mockIssueAction.GetIssuerReturns([]byte("issuer-v1"))
					mockTokensService.DeobfuscateReturns(&tokenpkg.Token{Quantity: "10"}, driver.Identity("issuer-v1"), []driver.Identity{driver.Identity("receiver-v1")}, "format", nil)
				} else {
					mockIssueAction.GetIssuerReturns([]byte("issuer-v2"))
					mockTokensService.DeobfuscateReturns(&tokenpkg.Token{Quantity: "10"}, driver.Identity("issuer-v2"), []driver.Identity{driver.Identity("receiver-v2")}, "format", nil)
				}
			}

			req, err := token.NewFullRequestFromBytes(tms, raw)
			require.NoError(t, err)
			require.NotNil(t, req)

			// Verify metadata was correctly restored
			require.NotNil(t, req.Metadata)

			// Verify that the metadata version matches or is at least correctly parsed.
			// The current driver.TokenRequestMetadata FromProtos doesn't store the version in the struct,
			// but we can check if it loaded the issues correctly.
			assert.NotEmpty(t, req.Metadata.Issues)

			// Check V3 specific field if it's V3
			if tc.version == driver.ProtocolV3 {
				issue := req.Metadata.Issues[0]
				assert.NotEmpty(t, issue.Issuer.AuditInfo, "V3 should have audit info")
				assert.NotEmpty(t, issue.ExtraSigners[0].AuditInfo, "V3 should have extra signer audit info")
			} else {
				issue := req.Metadata.Issues[0]
				assert.Empty(t, issue.Issuer.AuditInfo, "V1/V2 should NOT have audit info")
			}

			// Verify AuditCheck calls the auditor
			mockAuditor := &drivermock.AuditorService{}
			mockTMS.AuditorServiceReturns(mockAuditor)

			// IsValid will be called by AuditCheck
			// We need to mock Deserializers etc if we want IsValid to pass,
			// or we can just mock AuditorCheck and call it.

			// For regression, the most important is that AuditorCheck is reachable and doesn't crash
			err = req.AuditCheck(t.Context())
			// It might fail because of missing mocks for IsValid, but we want to see it reaches AuditorCheck
			require.NoError(t, err)

			assert.Equal(t, 1, mockAuditor.AuditorCheckCallCount())
		})
	}
}

func TestMetadataEdgeCases(t *testing.T) {
	// 1. Malformed metadata
	t.Run("Malformed Metadata", func(t *testing.T) {
		mockTMS := &drivermock.TokenManagerService{}
		mockPPM := &drivermock.PublicParamsManager{}
		mockPP := &drivermock.PublicParameters{}
		mockPPM.PublicParametersReturns(mockPP)
		mockTMS.PublicParamsManagerReturns(mockPPM)
		mockTMS.ValidatorReturns(&drivermock.Validator{}, nil)

		mockVP := &tokenmock.VaultProvider{}
		mockV := &drivermock.Vault{}
		mockVP.VaultReturns(mockV, nil)
		tms, err := token.NewManagementService(token.TMSID{}, mockTMS, logging.MustGetLogger("test"), mockVP, nil, nil)
		require.NoError(t, err)

		req := token.NewRequest(tms, "anchor")
		err = req.Metadata.FromBytes([]byte("not a proto"))
		assert.Error(t, err)
	})

	// 2. Missing audit info in V3 (should still pass if allowed, but here we check if it's handled)
	t.Run("V3 Missing Audit Info", func(t *testing.T) {
		trm := &driver.TokenRequestMetadata{
			Issues: []*driver.IssueMetadata{
				{
					Issuer: driver.AuditableIdentity{Identity: driver.Identity("issuer")}, // AuditInfo is nil
				},
			},
		}
		raw, err := trm.Bytes()
		require.NoError(t, err)

		trm2 := &driver.TokenRequestMetadata{}
		err = trm2.FromBytes(raw)
		require.NoError(t, err)
		assert.Equal(t, driver.Identity("issuer"), trm2.Issues[0].Issuer.Identity)
		assert.Empty(t, trm2.Issues[0].Issuer.AuditInfo)
	})
}
