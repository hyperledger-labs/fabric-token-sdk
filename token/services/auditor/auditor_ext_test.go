package auditor

import (
	"context"
	"errors"
	"testing"
	"time"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	drivermock "github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	tokenmock "github.com/hyperledger-labs/fabric-token-sdk/token/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	netdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubDriverNetwork implements netdriver.Network with no-op methods.
// Only AddFinalityListener is needed for the Append success path.
type stubDriverNetwork struct{}

func (s *stubDriverNetwork) Name() string    { return "stub" }
func (s *stubDriverNetwork) Channel() string { return "stubch" }
func (s *stubDriverNetwork) Normalize(opt *token.ServiceOptions) (*token.ServiceOptions, error) {
	return opt, nil
}
func (s *stubDriverNetwork) Connect(ns string) ([]token.ServiceOption, error) { return nil, nil }
func (s *stubDriverNetwork) Broadcast(ctx context.Context, blob interface{}) error {
	return nil
}
func (s *stubDriverNetwork) NewEnvelope() netdriver.Envelope { return nil }
func (s *stubDriverNetwork) RequestApproval(context view2.Context, tms *token.ManagementService, requestRaw []byte, signer view2.Identity, txID netdriver.TxID) (netdriver.Envelope, error) {
	return nil, nil
}
func (s *stubDriverNetwork) ComputeTxID(id *netdriver.TxID) string { return "" }
func (s *stubDriverNetwork) FetchPublicParameters(namespace string) ([]byte, error) {
	return nil, nil
}
func (s *stubDriverNetwork) QueryTokens(ctx context.Context, namespace string, IDs []*token2.ID) ([][]byte, error) {
	return nil, nil
}
func (s *stubDriverNetwork) AreTokensSpent(ctx context.Context, namespace string, tokenIDs []*token2.ID, meta []string) ([]bool, error) {
	return nil, nil
}
func (s *stubDriverNetwork) LocalMembership() netdriver.LocalMembership { return nil }
func (s *stubDriverNetwork) AddFinalityListener(namespace string, txID string, listener netdriver.FinalityListener) error {
	return nil
}
func (s *stubDriverNetwork) LookupTransferMetadataKey(namespace string, key string, timeout time.Duration) ([]byte, error) {
	return nil, nil
}
func (s *stubDriverNetwork) Ledger() (netdriver.Ledger, error) { return nil, nil }

// newStubNetwork creates a *network.Network backed by our stubDriverNetwork.
func newStubNetwork() *network.Network {
	return network.NewNetwork(&stubDriverNetwork{}, nil)
}

// stubDriverNetworkWithErr embeds stubDriverNetwork but overrides AddFinalityListener.
type stubDriverNetworkWithErr struct {
	stubDriverNetwork
	err error
}

func (s *stubDriverNetworkWithErr) AddFinalityListener(namespace string, txID string, listener netdriver.FinalityListener) error {
	return s.err
}

func TestService_Validate(t *testing.T) {
	svc := &Service{}
	assert.Panics(t, func() {
		_ = svc.Validate(context.Background(), &token.Request{})
	})
}

func TestService_Audit_AuditRecordError(t *testing.T) {
	// Use a TMS with nil PublicParameters to cause inputsAndOutputs to fail.
	mockTMS := &drivermock.TokenManagerService{}
	mockPPM := &drivermock.PublicParamsManager{}
	mockPPM.PublicParametersReturns(nil) // nil PP causes AuditRecord error
	mockTMS.PublicParamsManagerReturns(mockPPM)
	mockTMS.ValidatorReturns(&drivermock.Validator{}, nil)
	mockTMS.TokensServiceReturns(&drivermock.TokensService{})
	mockTMS.WalletServiceReturns(&drivermock.WalletService{})

	mockVP := &tokenmock.VaultProvider{}
	mockV := &drivermock.Vault{}
	mockV.QueryEngineReturns(&drivermock.QueryEngine{})
	mockVP.VaultReturns(mockV, nil)

	badTMS, err := token.NewManagementService(
		token.TMSID{}, mockTMS, logging.MustGetLogger("test"), mockVP, nil, nil,
	)
	require.NoError(t, err)

	svc := &Service{
		metrics: newMetrics(nil),
		auditDB: newTestStoreService(t, &stubAuditTransactionStore{}),
	}
	tx := &mockTransaction{anchor: "tx-err", tms: badTMS}

	_, _, err = svc.Audit(context.Background(), tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed getting transaction audit record")
}

func TestService_Audit_Success(t *testing.T) {
	svc := &Service{
		metrics: newMetrics(nil),
		auditDB: newTestStoreService(t, &stubAuditTransactionStore{}),
	}
	tx := &mockTransaction{
		anchor: "tx-audit-ok",
		tms:    newTestManagementService(t),
	}

	inputs, outputs, err := svc.Audit(context.Background(), tx)
	assert.NoError(t, err)
	assert.NotNil(t, inputs)
	assert.NotNil(t, outputs)
}

func TestService_Append_Error_TMSProvider(t *testing.T) {
	svc := &Service{
		metrics: newMetrics(nil),
		tmsProvider: &mockTokenManagementServiceProvider{err: errors.New("tms err")},
	}
	tx := &mockTransaction{anchor: "tx-app", tms: newTestManagementService(t)}
	
	svc.auditDB = newTestStoreService(t, &stubAuditTransactionStore{})
	
	err := svc.Append(context.Background(), tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tms err")
}

func TestRequestWrapper_PublicParamsHash(t *testing.T) {
	rw := newRequestWrapper(minimalRequest("tx-pph"), nil)
	assert.Panics(t, func() {
		rw.PublicParamsHash()
	})
}

func TestService_Append_Success(t *testing.T) {
	svc := &Service{
		metrics:         newMetrics(nil),
		tmsProvider:     &mockTokenManagementServiceProvider{},
		auditDB:         newTestStoreService(t, &stubAuditTransactionStore{}),
		networkProvider: &mockNetworkProvider{net: newStubNetwork()},
	}
	tx := &mockTransaction{
		anchor: "tx-app-success",
		tms:    newTestManagementService(t),
	}

	err := svc.Append(context.Background(), tx)
	assert.NoError(t, err)
}

func TestService_Append_GetNetworkError(t *testing.T) {
	svc := &Service{
		metrics:         newMetrics(nil),
		tmsProvider:     &mockTokenManagementServiceProvider{},
		auditDB:         newTestStoreService(t, &stubAuditTransactionStore{}),
		networkProvider: &mockNetworkProvider{err: errors.New("network unavailable")},
	}
	tx := &mockTransaction{
		anchor: "tx-net-err",
		tms:    newTestManagementService(t),
	}

	err := svc.Append(context.Background(), tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed getting network instance")
}

func TestService_Append_AddFinalityListenerError(t *testing.T) {
	// Create a stub network where AddFinalityListener returns an error.
	stubNet := &stubDriverNetworkWithErr{err: errors.New("listener fail")}
	svc := &Service{
		metrics:         newMetrics(nil),
		tmsProvider:     &mockTokenManagementServiceProvider{},
		auditDB:         newTestStoreService(t, &stubAuditTransactionStore{}),
		networkProvider: &mockNetworkProvider{net: network.NewNetwork(stubNet, nil)},
	}
	tx := &mockTransaction{
		anchor: "tx-listener-err",
		tms:    newTestManagementService(t),
	}

	err := svc.Append(context.Background(), tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed listening to network")
}
func TestRequestWrapper_CompleteInputsWithEmptyEID_Shortcut(t *testing.T) {
	// Use the default TMS (empty token list) for the shortcut test.
	tx := &mockTransaction{anchor: "tx-cid", tms: newTestManagementService(t)}
	rw := newRequestWrapper(tx.Request(), newTestManagementService(t))

	// Empty inputs → shortcut, returns nil immediately.
	record := &token.AuditRecord{
		Inputs: token.NewInputStream(nil, []*token.Input{}, 0),
	}
	err := rw.completeInputsWithEmptyEID(context.Background(), record)
	assert.NoError(t, err)
}

func TestRequestWrapper_CompleteInputsWithEmptyEID_WithInputs(t *testing.T) {
	// This test needs a TMS whose ListAuditTokens returns a token
	// matching the single input in the record below.
	tmsWithToken := newTestManagementServiceWithTokens(t, []*token2.Token{
		{Type: "USD", Quantity: "100", Owner: []byte("owner1")},
	})
	tx := &mockTransaction{anchor: "tx-cid2", tms: tmsWithToken}
	rw := newRequestWrapper(tx.Request(), tmsWithToken)

	recordWithInputs := &token.AuditRecord{
		Inputs:  token.NewInputStream(nil, []*token.Input{{Id: &token2.ID{TxId: "123"}}}, 0),
		Outputs: token.NewOutputStream([]*token.Output{{EnrollmentID: "target"}}, 0),
	}
	err := rw.completeInputsWithEmptyEID(context.Background(), recordWithInputs)
	assert.NoError(t, err)
}

func TestRequestWrapper_AuditRecord(t *testing.T) {
	tx := &mockTransaction{anchor: "tx-ar", tms: newTestManagementService(t)}
	rw := newRequestWrapper(tx.Request(), newTestManagementService(t))
	record, err := rw.AuditRecord(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, record)
}


func TestService_Append_AuditError(t *testing.T) {
	svc := &Service{
		metrics:         newMetrics(nil),
		auditDB:         newTestStoreService(t, &stubAuditTransactionStore{appendErr: errors.New("db append err")}),
		tmsProvider:     &mockTokenManagementServiceProvider{},
		networkProvider: &mockNetworkProvider{net: newStubNetwork()},
	}
	tx := &mockTransaction{anchor: "tx-app-err", tms: newTestManagementService(t)}

	err := svc.Append(context.Background(), tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed appending request")
}

func TestService_Audit_DBCleanSuccess(t *testing.T) {
	// Even with a DB status error, Audit succeeds because it doesn't check status.
	svc := &Service{
		metrics: newMetrics(nil),
		auditDB: newTestStoreService(t, &stubAuditTransactionStore{getStatusErr: errors.New("db status err")}),
	}
	tx := &mockTransaction{anchor: "tx-aud-err", tms: newTestManagementService(t)}

	inputs, outputs, err := svc.Audit(context.Background(), tx)
	assert.NoError(t, err)
	assert.NotNil(t, inputs)
	assert.NotNil(t, outputs)
}

func TestService_Audit_NotUnknown(t *testing.T) {
	svc := &Service{
		metrics:     newMetrics(nil),
		auditDB:     newTestStoreService(t, &stubAuditTransactionStore{getStatusResult: dbdriver.Pending}),
	}
	tx := &mockTransaction{anchor: "tx-aud-not-unknown", tms: newTestManagementService(t)}

	// Audit succeeds because AcquireLocks doesn't check transaction status.
	inputs, outputs, err := svc.Audit(context.Background(), tx)
	assert.NoError(t, err)
	assert.NotNil(t, inputs)
	assert.NotNil(t, outputs)
}

func TestService_Audit_TMSProviderIrrelevant(t *testing.T) {
	// tmsProvider error does NOT affect Audit — only affects Append.
	svc := &Service{
		metrics:     newMetrics(nil),
		auditDB:     newTestStoreService(t, &stubAuditTransactionStore{}),
		tmsProvider: &mockTokenManagementServiceProvider{err: errors.New("tms err")},
	}
	tx := &mockTransaction{anchor: "tx-aud-tms-err", tms: newTestManagementService(t)}

	inputs, outputs, err := svc.Audit(context.Background(), tx)
	assert.NoError(t, err)
	assert.NotNil(t, inputs)
	assert.NotNil(t, outputs)
}
