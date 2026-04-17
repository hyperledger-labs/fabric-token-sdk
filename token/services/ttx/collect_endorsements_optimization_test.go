/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"context"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	drivermock "github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	tokenmock "github.com/hyperledger-labs/fabric-token-sdk/token/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

// fakeViewContext is a minimal view.Context for unit tests.
// Only the methods actually exercised by CollectEndorsementsView.Call are implemented.
type fakeViewContext struct {
	goCtx      context.Context
	servicesFn func(v interface{}) (interface{}, error)
	sessionErr error
}

func (f *fakeViewContext) Context() context.Context { return f.goCtx }
func (f *fakeViewContext) GetService(v interface{}) (interface{}, error) {
	return f.servicesFn(v)
}
func (f *fakeViewContext) GetSession(_ view.View, _ view.Identity, _ ...view.View) (view.Session, error) {
	return nil, f.sessionErr
}
func (f *fakeViewContext) GetSessionByID(_ string, _ view.Identity) (view.Session, error) {
	return nil, nil
}
func (f *fakeViewContext) ID() string                { return "" }
func (f *fakeViewContext) Me() view.Identity         { return nil }
func (f *fakeViewContext) IsMe(_ view.Identity) bool { return false }
func (f *fakeViewContext) Initiator() view.View      { return nil }
func (f *fakeViewContext) Session() view.Session     { return nil }
func (f *fakeViewContext) RunView(_ view.View, _ ...view.RunViewOption) (interface{}, error) {
	return nil, nil
}
func (f *fakeViewContext) OnError(_ func()) {}
func (f *fakeViewContext) StartSpanFrom(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
	return ctx, trace.SpanFromContext(ctx)
}

// testTMS wraps token.ManagementService to satisfy dep.TokenManagementServiceWithExtensions
// without importing the wrapper package (which imports ttx, causing a cycle).
type testTMS struct {
	*token.ManagementService
}

func (t *testTMS) SetTokenManagementService(req *token.Request) error {
	if req == nil {
		return errors.New("request cannot be nil")
	}
	req.SetTokenService(t.ManagementService)

	return nil
}

var _ dep.TokenManagementServiceWithExtensions = (*testTMS)(nil)

// newMockedManagementService creates a token.ManagementService backed by the
// given identity provider and wallet service mocks.
func newMockedManagementService(t *testing.T, tmsID token.TMSID, mockIP *drivermock.IdentityProvider, mockWS *drivermock.WalletService) *token.ManagementService {
	t.Helper()

	mockDriverTMS := &drivermock.TokenManagerService{}
	mockDriverTMS.IdentityProviderReturns(mockIP)
	mockDriverTMS.WalletServiceReturns(mockWS)

	ppm := &drivermock.PublicParamsManager{}
	ppm.PublicParametersReturns(&drivermock.PublicParameters{})
	mockDriverTMS.PublicParamsManagerReturns(ppm)

	mockVP := &tokenmock.VaultProvider{}
	mockVault := &drivermock.Vault{}
	mockVault.QueryEngineReturns(&drivermock.QueryEngine{})
	mockVP.VaultReturns(mockVault, nil)

	ms, err := token.NewManagementService(tmsID, mockDriverTMS, nil, mockVP, nil, nil)
	require.NoError(t, err)

	return ms
}

// TestRequestSignatures_RemoteIdentity_SkipsGetSigner verifies the optimization
// introduced in issue #1226: when SigService.IsMe() returns false for a signer,
// GetSigner is never invoked, avoiding the expensive idemix sign-and-verify
// deserialization that was previously triggered unconditionally.
func TestRequestSignatures_RemoteIdentity_SkipsGetSigner(t *testing.T) {
	tmsID := token.TMSID{Network: "network", Channel: "channel", Namespace: "namespace"}

	// Use a properly typed identity so that multisig.Unwrap succeeds (returns ok=false).
	remoteParty, err := identity.WrapWithType(driver.X509IdentityType, []byte("remote_party_key"))
	require.NoError(t, err)

	// IdentityProvider: IsMe returns false (identity is not ours).
	// GetSigner is intentionally not configured; zero-value return would indicate an unintended call.
	mockIP := &drivermock.IdentityProvider{}
	mockIP.IsMeReturns(false)

	// WalletService: no local wallet exists for the remote signer.
	mockWS := &drivermock.WalletService{}
	mockWS.OwnerWalletReturns(nil, errors.New("no wallet for remote party"))

	ms := newMockedManagementService(t, tmsID, mockIP, mockWS)

	req := token.NewRequest(nil, "an_anchor")
	req.Metadata.Transfers = []*driver.TransferMetadata{
		{
			Inputs: []*driver.TransferInputMetadata{
				{
					Senders: []*driver.AuditableIdentity{{Identity: remoteParty}},
				},
			},
		},
	}

	tx := &Transaction{
		Payload: &Payload{
			tmsID:        tmsID,
			TokenRequest: req,
			ID:           "an_anchor",
		},
		TMS: &testTMS{ManagementService: ms},
	}

	cev := NewCollectEndorsementsView(tx,
		WithSkipAuditing(),
		WithSkipApproval(),
		WithSkipDistributeEnv(),
	)

	metrics := NewMetrics(&disabled.Provider{})
	callCount := 0
	ctx := &fakeViewContext{
		goCtx: t.Context(),
		servicesFn: func(v interface{}) (interface{}, error) {
			callCount++
			if callCount == 1 {
				return metrics, nil
			}

			return nil, errors.New("unexpected GetService call")
		},
		sessionErr: errors.New("no session available"),
	}

	_, callErr := cev.Call(ctx)

	assert.Error(t, callErr, "Call should fail because remote signing cannot proceed")
	assert.Equal(t, 0, mockIP.GetSignerCallCount(),
		"GetSigner must not be called when IsMe() returns false for a remote party")
	assert.GreaterOrEqual(t, mockIP.IsMeCallCount(), 1,
		"IsMe must be called to determine whether the signer is local")
}
