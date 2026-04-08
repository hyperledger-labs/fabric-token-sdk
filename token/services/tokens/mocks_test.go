package tokens

import (
	"context"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	netdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	storage "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/tokendb"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/test-go/testify/require"
)

type mockWalletService struct {
	driver.WalletService
}

func (m *mockWalletService) SpendIDs(ids ...*token2.ID) ([]string, error) {
	result := make([]string, len(ids))
	for i, id := range ids {
		result[i] = id.TxId
	}
	return result, nil
}

// buildTestTMS creates a *token.ManagementService with fully wired mocks for isolated testing.
func buildTestTMS(t *testing.T, tmsID token.TMSID, mockVaultInst *mockVault, mockAuth driver.Authorization) *token.ManagementService {
	t.Helper()
	mockPPM := &mockPublicParamsManager{
		PublicParametersReturns: &mockPublicParameters{},
	}
	mockTMS := &mockTokenManagerService{
		AuthorizationReturns:       mockAuth,
		PublicParamsManagerReturns: mockPPM,
		VaultReturns:               mockVaultInst,
		WalletServiceReturns:       &mockWalletService{},
	}
	tms, err := token.NewManagementService(
		tmsID, mockTMS, logger,
		&mockVaultProvider{VaultReturns: mockVaultInst},
		&mockCertificationClientProvider{},
		&mockSelectorManagerProvider{},
	)
	require.NoError(t, err)
	return tms
}

type mockVault struct {
	driver.Vault
	QueryEngineReturns driver.QueryEngine
}

func (m *mockVault) QueryEngine() driver.QueryEngine {
	return m.QueryEngineReturns
}

func (m *mockVault) CertificationStorage() driver.CertificationStorage {
	return nil
}

type mockQueryEngine struct {
	driver.QueryEngine
	UnspentTokensIteratorReturns driver.UnspentTokensIterator
	UnspentTokensIteratorError   error
}

func (m *mockQueryEngine) UnspentTokensIterator(ctx context.Context) (driver.UnspentTokensIterator, error) {
	return m.UnspentTokensIteratorReturns, m.UnspentTokensIteratorError
}

type mockUnspentTokensIterator struct {
	driver.UnspentTokensIterator
	NextReturnsTokens []*token2.UnspentToken
	NextError         error
	NextCalls         int
	CloseCalls        int
}

func (m *mockUnspentTokensIterator) Next() (*token2.UnspentToken, error) {
	if m.NextCalls >= len(m.NextReturnsTokens) {
		return nil, m.NextError
	}
	res := m.NextReturnsTokens[m.NextCalls]
	m.NextCalls++
	return res, m.NextError
}

func (m *mockUnspentTokensIterator) Close() {
	m.CloseCalls++
}

type mockPublicParamsManager struct {
	driver.PublicParamsManager
	PublicParametersReturns driver.PublicParameters
}

func (m *mockPublicParamsManager) PublicParameters() driver.PublicParameters {
	return m.PublicParametersReturns
}

func (m *mockPublicParamsManager) PublicParamsHash() driver.PPHash {
	return nil
}

type mockPublicParameters struct {
	driver.PublicParameters
	GraphHidingReturns bool
	PrecisionReturns   uint64
}

func (m *mockPublicParameters) GraphHiding() bool {
	return m.GraphHidingReturns
}

func (m *mockPublicParameters) Precision() uint64 {
	return m.PrecisionReturns
}

type mockAuthorization struct {
	driver.Authorization
	AmIAnAuditorReturns bool
	IsMineReturnsWallet string
	IsMineReturnsIDs    []string
	IsMineReturnsMine   bool
	IssuedReturns       bool
	OwnerTypeReturns    driver.IdentityType
	OwnerTypeReturnsID  []byte
	OwnerTypeReturnsErr error
}

func (m *mockAuthorization) AmIAnAuditor() bool {
	return m.AmIAnAuditorReturns
}

func (m *mockAuthorization) IsMine(ctx context.Context, tok *token2.Token) (string, []string, bool) {
	return m.IsMineReturnsWallet, m.IsMineReturnsIDs, m.IsMineReturnsMine
}

func (m *mockAuthorization) Issued(ctx context.Context, issuer driver.Identity, tok *token2.Token) bool {
	return m.IssuedReturns
}

func (m *mockAuthorization) OwnerType(raw []byte) (driver.IdentityType, []byte, error) {
	return m.OwnerTypeReturns, m.OwnerTypeReturnsID, m.OwnerTypeReturnsErr
}

type mockTMSProvider struct {
	GetManagementServiceReturns *token.ManagementService
	GetManagementServiceError   error
}

func (m *mockTMSProvider) GetManagementService(opts ...token.ServiceOption) (*token.ManagementService, error) {
	return m.GetManagementServiceReturns, m.GetManagementServiceError
}

type mockNetwork struct {
	netdriver.Network
	NameReturns           string
	ChannelReturns        string
	AreTokensSpentReturns []bool
	AreTokensSpentError   error
}

func (m *mockNetwork) Name() string { return m.NameReturns }

func (m *mockNetwork) Channel() string { return m.ChannelReturns }

func (m *mockNetwork) AreTokensSpent(ctx context.Context, namespace string, tokenIDs []*token2.ID, meta []string) ([]bool, error) {
	return m.AreTokensSpentReturns, m.AreTokensSpentError
}

func (m *mockNetwork) LocalMembership() netdriver.LocalMembership {
	return nil
}



type mockStoreServiceManager struct {
	StoreServiceByTMSIdReturns *tokendb.StoreService
	StoreServiceByTMSIdError   error
}

func (m *mockStoreServiceManager) StoreServiceByTMSId(tmsID token.TMSID) (*tokendb.StoreService, error) {
	return m.StoreServiceByTMSIdReturns, m.StoreServiceByTMSIdError
}

type mockNetworkProvider struct {
	GetNetworkReturns *network.Network
	GetNetworkError   error
}

func (m *mockNetworkProvider) GetNetwork(net, channel string) (*network.Network, error) {
	return m.GetNetworkReturns, m.GetNetworkError
}

// buildTestNetwork creates a *network.Network wrapping a mock driver.
func buildTestNetwork(mockDriver *mockNetwork) *network.Network {
	return network.NewNetwork(mockDriver, nil)
}

type mockPublisher struct {
	PublishCalls int
}

func (m *mockPublisher) Publish(event events.Event) {
	m.PublishCalls++
}

type mockTokenDB struct {
	storage.TokenStore
	TransactionExistsReturns     bool
	TransactionExistsError       error
	NewTransactionReturns        storage.TokenStoreTransaction
	NewTransactionError          error
	StorePublicParamsError       error
	UnspentTokensIteratorReturns driver.UnspentTokensIterator
}

func (m *mockTokenDB) TransactionExists(ctx context.Context, txID string) (bool, error) {
	return m.TransactionExistsReturns, m.TransactionExistsError
}

func (m *mockTokenDB) NewTokenDBTransaction() (storage.TokenStoreTransaction, error) {
	return m.NewTransactionReturns, m.NewTransactionError
}

func (m *mockTokenDB) StorePublicParams(ctx context.Context, publicParams []byte) error {
	return m.StorePublicParamsError
}

func (m *mockTokenDB) UnspentTokensIterator(ctx context.Context) (driver.UnspentTokensIterator, error) {
	return m.UnspentTokensIteratorReturns, nil
}

func (m *mockTokenDB) DeleteTokens(ctx context.Context, deletedBy string, toDelete ...*token2.ID) error {
	return nil
}

func (m *mockTokenDB) IsMine(ctx context.Context, txID string, index uint64) (bool, error) {
	return false, nil
}

func (m *mockTokenDB) SetSupportedTokenFormats(formats []token2.Format) error {
	return nil
}

func (m *mockTokenDB) UnsupportedTokensIteratorBy(ctx context.Context, walletID string, tokenType token2.Type) (driver.UnsupportedTokensIterator, error) {
	return nil, nil
}

func (m *mockTokenDB) ExistsCertification(ctx context.Context, id *token2.ID) bool { return false }
func (m *mockTokenDB) Close() error                                                { return nil }

type mockTokenDBTransaction struct {
	storage.TokenStoreTransaction
	CommitError                              error
	RollbackError                            error
	StoreTokenError                          error
	DeleteError                              error
	SetSpendableError                        error
	SetSpendableBySupportedTokenFormatsError error
	GetTokenValue                            *token2.Token
	GetTokenOwners                           []string
	GetTokenError                            error
}

func (m *mockTokenDBTransaction) Commit() error {
	return m.CommitError
}

func (m *mockTokenDBTransaction) Rollback() error {
	return m.RollbackError
}

func (m *mockTokenDBTransaction) StoreToken(ctx context.Context, tr storage.TokenRecord, owners []string) error {
	return m.StoreTokenError
}

func (m *mockTokenDBTransaction) Delete(ctx context.Context, tokenID token2.ID, deletedBy string) error {
	return m.DeleteError
}

func (m *mockTokenDBTransaction) SetSpendable(ctx context.Context, tokenID token2.ID, spendable bool) error {
	return m.SetSpendableError
}

func (m *mockTokenDBTransaction) SetSpendableBySupportedTokenFormats(ctx context.Context, formats []token2.Format) error {
	return m.SetSpendableBySupportedTokenFormatsError
}

func (m *mockTokenDBTransaction) GetToken(ctx context.Context, tokenID token2.ID, includeDeleted bool) (*token2.Token, []string, error) {
	return m.GetTokenValue, m.GetTokenOwners, m.GetTokenError
}

type mockCache struct {
	GetReturns  *CacheEntry
	GetFound    bool
	AddCalls    int
	DeleteCalls int
}

func (m *mockCache) Get(key string) (*CacheEntry, bool) {
	return m.GetReturns, m.GetFound
}

func (m *mockCache) Add(key string, value *CacheEntry) {
	m.AddCalls++
}

func (m *mockCache) Delete(key string) {
	m.DeleteCalls++
}

type mockServiceProvider struct {
	GetServiceReturns interface{}
	GetServiceError   error
}

func (m *mockServiceProvider) GetService(v interface{}) (interface{}, error) {
	return m.GetServiceReturns, m.GetServiceError
}

type mockTokenManagerService struct {
	driver.TokenManagerService
	AuthorizationReturns        driver.Authorization
	PublicParamsManagerReturns  driver.PublicParamsManager
	VaultReturns                driver.Vault
	TokensServiceReturns        driver.TokensService
	TokensUpgradeServiceReturns driver.TokensUpgradeService
	WalletServiceReturns        driver.WalletService
	IdentityProviderReturns     driver.IdentityProvider
	DeserializerReturns         driver.Deserializer
	ValidatorReturns            driver.Validator
	ValidatorError              error
}

func (m *mockTokenManagerService) Vault() driver.Vault {
	return m.VaultReturns
}

func (m *mockTokenManagerService) IssueService() driver.IssueService {
	return nil
}

func (m *mockTokenManagerService) TransferService() driver.TransferService {
	return nil
}

func (m *mockTokenManagerService) AuditorService() driver.AuditorService {
	return nil
}

func (m *mockTokenManagerService) CertificationService() driver.CertificationService {
	return nil
}

func (m *mockTokenManagerService) Configuration() driver.Configuration {
	return nil
}

func (m *mockTokenManagerService) Done() error {
	return nil
}

func (m *mockTokenManagerService) Authorization() driver.Authorization {
	return m.AuthorizationReturns
}

func (m *mockTokenManagerService) PublicParamsManager() driver.PublicParamsManager {
	return m.PublicParamsManagerReturns
}

func (m *mockTokenManagerService) TokensService() driver.TokensService {
	return m.TokensServiceReturns
}

func (m *mockTokenManagerService) TokensUpgradeService() driver.TokensUpgradeService {
	return m.TokensUpgradeServiceReturns
}

func (m *mockTokenManagerService) WalletService() driver.WalletService {
	return m.WalletServiceReturns
}

func (m *mockTokenManagerService) IdentityProvider() driver.IdentityProvider {
	return m.IdentityProviderReturns
}

func (m *mockTokenManagerService) Deserializer() driver.Deserializer {
	return m.DeserializerReturns
}

func (m *mockTokenManagerService) Validator() (driver.Validator, error) {
	return m.ValidatorReturns, m.ValidatorError
}

type mockVaultProvider struct {
	token.VaultProvider
	VaultReturns driver.Vault
	VaultError   error
}

func (m *mockVaultProvider) Vault(network, channel, namespace string) (driver.Vault, error) {
	return m.VaultReturns, m.VaultError
}

type mockCertificationClientProvider struct {
	token.CertificationClientProvider
}

type mockSelectorManagerProvider struct {
	token.SelectorManagerProvider
}
