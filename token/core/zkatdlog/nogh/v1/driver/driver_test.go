/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	math3 "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	mock2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/common/driver/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	tdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	dmock "github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	imock "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver/mock"
	idmock "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

// testingHelper is a test helper for creating test data.
func testingHelper(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "setup", "testdata", "idemix", "msp", "IssuerPublicKey")
	issuerPK, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotEmpty(t, issuerPK)

	return issuerPK
}

// TestNewDriver tests the creation of a new zkatdlog driver.
func TestNewDriver(t *testing.T) {
	metricsProvider := &disabled.Provider{}
	configService := &mock2.ConfigService{}
	storageProvider := &imock.StorageProvider{}
	identityProvider := &mock2.IdentityProvider{}
	endpointService := &idmock.NetworkBinderService{}
	networkProvider := &mock2.NetworkProvider{}
	vaultProvider := &mock2.VaultProvider{}

	factory := driver.NewDriver(
		metricsProvider,
		noop.NewTracerProvider(),
		configService,
		storageProvider,
		identityProvider,
		endpointService,
		networkProvider,
		vaultProvider,
	)

	assert.NotNil(t, factory.Driver)
	assert.Equal(t, core.DriverIdentifier(setup.DLogNoGHDriverName, setup.ProtocolV1), factory.Name)
}

// TestNewTokenService tests the creation of a new zkatdlog token manager service, covering success and various error paths.
func TestNewTokenService(t *testing.T) {
	metricsProvider := &disabled.Provider{}
	configService := &mock2.ConfigService{}
	storageProvider := &imock.StorageProvider{}
	identityProvider := &mock2.IdentityProvider{}
	endpointService := &idmock.NetworkBinderService{}
	networkProvider := &mock2.NetworkProvider{}
	vaultProvider := &mock2.VaultProvider{}

	d := driver.NewDriver(
		metricsProvider,
		noop.NewTracerProvider(),
		configService,
		storageProvider,
		identityProvider,
		endpointService,
		networkProvider,
		vaultProvider,
	).Driver.(*driver.Driver)

	tmsID := tdriver.TMSID{Network: "n1", Channel: "c1", Namespace: "ns1"}
	issuerPK := testingHelper(t)
	pp, err := setup.NewWith(setup.DLogNoGHDriverName, setup.ProtocolV1, 32, issuerPK, math3.FP256BN_AMCL)
	require.NoError(t, err)
	pp.AddIssuer([]byte("issuer-1"))
	pp.AddAuditor([]byte("auditor-1"))
	ppBytes, err := pp.Serialize()
	require.NoError(t, err)

	// Case 1: Empty public parameters
	ts, err := d.NewTokenService(tmsID, nil)
	require.Error(t, err)
	assert.Nil(t, ts)
	assert.Contains(t, err.Error(), "empty public parameters")

	// Case 2: Failed getting network
	networkProvider.GetNetworkReturns(nil, errors.New("network-error"))
	ts, err = d.NewTokenService(tmsID, ppBytes)
	require.Error(t, err)
	assert.Nil(t, ts)
	assert.Contains(t, err.Error(), "failed getting network [network-error]")

	// Case 3: Failed getting vault
	networkProvider.GetNetworkReturns(&network.Network{}, nil)
	vaultProvider.VaultReturns(nil, errors.New("vault-error"))
	ts, err = d.NewTokenService(tmsID, ppBytes)
	require.Error(t, err)
	assert.Nil(t, ts)
	assert.Contains(t, err.Error(), "failed getting vault [vault-error]")

	// Case 4: Failed to get config
	vaultProvider.VaultReturns(&dmock.Vault{}, nil)
	configService.ConfigurationForReturns(nil, errors.New("config-error"))
	ts, err = d.NewTokenService(tmsID, ppBytes)
	require.Error(t, err)
	assert.Nil(t, ts)
	assert.Contains(t, err.Error(), "failed to get config for token service")

	// Case 5: Failed to initialize public params manager (invalid bytes)
	configService.ConfigurationForReturns(&dmock.Configuration{}, nil)
	ts, err = d.NewTokenService(tmsID, []byte("invalid-pp"))
	require.Error(t, err)
	assert.Nil(t, ts)
	assert.Contains(t, err.Error(), "failed to initiliaze public params manager")

	// Case 6: Failed to initialize wallet service
	nw := &mock2.Network{}
	lm := &mock2.LocalMembership{}
	nw.LocalMembershipReturns(lm)
	networkProvider.GetNetworkReturns(network.NewNetwork(nw, network.NewLocalMembership(lm)), nil)
	vault := &dmock.Vault{}
	qe := &dmock.QueryEngine{}
	vault.QueryEngineReturns(qe)
	vaultProvider.VaultReturns(vault, nil)

	storageProvider.IdentityStoreReturns(nil, errors.New("identity-store-error"))
	ts, err = d.NewTokenService(tmsID, ppBytes)
	require.Error(t, err)
	assert.Nil(t, ts)
	assert.Contains(t, err.Error(), "failed to initiliaze wallet service")

	// Case 7: Success
	identityStore := &imock.IdentityStoreService{}
	identityStore.IteratorConfigurationsReturns(&mock2.IdentityConfigurationIterator{}, nil)
	keystore := &mock2.Keystore{}
	walletStore := &imock.WalletStoreService{}
	storageProvider.IdentityStoreReturns(identityStore, nil)
	storageProvider.KeystoreReturns(keystore, nil)
	storageProvider.WalletStoreReturns(walletStore, nil)
	identityProvider.DefaultIdentityReturns([]byte("fsc-identity"))
	lm.DefaultIdentityReturns([]byte("network-identity"))

	// Configuration ID should match tmsID
	config := &dmock.Configuration{}
	config.IDReturns(tmsID)
	configService.ConfigurationForReturns(config, nil)

	ts, err = d.NewTokenService(tmsID, ppBytes)
	require.NoError(t, err)
	assert.NotNil(t, ts)
}

// TestNewDefaultValidator tests the creation of a default zkatdlog validator.
func TestNewDefaultValidator(t *testing.T) {
	metricsProvider := &disabled.Provider{}
	configService := &mock2.ConfigService{}
	storageProvider := &imock.StorageProvider{}
	identityProvider := &mock2.IdentityProvider{}
	endpointService := &idmock.NetworkBinderService{}
	networkProvider := &mock2.NetworkProvider{}
	vaultProvider := &mock2.VaultProvider{}

	d := driver.NewDriver(
		metricsProvider,
		noop.NewTracerProvider(),
		configService,
		storageProvider,
		identityProvider,
		endpointService,
		networkProvider,
		vaultProvider,
	).Driver.(*driver.Driver)

	issuerPK := testingHelper(t)
	pp, _ := setup.Setup(32, issuerPK, math3.FP256BN_AMCL)

	// Case 1: Valid public parameters
	v, err := d.NewDefaultValidator(pp)
	require.NoError(t, err)
	assert.NotNil(t, v)

	// Case 2: Invalid public parameters type
	v, err = d.NewDefaultValidator(&dmock.PublicParameters{})
	require.Error(t, err)
	assert.Nil(t, v)
	assert.Contains(t, err.Error(), "invalid public parameters type")
}

// TestPublicParametersFromBytes tests the unmarshalling of public parameters from bytes.
func TestPublicParametersFromBytes(t *testing.T) {
	d := &driver.Driver{}
	issuerPK := testingHelper(t)
	pp, _ := setup.Setup(32, issuerPK, math3.FP256BN_AMCL)
	ppBytes, err := pp.Serialize()
	require.NoError(t, err)

	res, err := d.PublicParametersFromBytes(ppBytes)
	require.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, uint64(32), res.Precision())

	_, err = d.PublicParametersFromBytes([]byte("invalid"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal public parameters")
}

// TestPPMFactory tests the zkatdlog public parameters manager factory.
func TestPPMFactory(t *testing.T) {
	factory := driver.NewPPMFactory()
	assert.Equal(t, core.DriverIdentifier(setup.DLogNoGHDriverName, setup.ProtocolV1), factory.Name)
	assert.NotNil(t, factory.Driver)

	ppmFactory := factory.Driver
	issuerPK := testingHelper(t)
	pp, _ := setup.Setup(32, issuerPK, math3.FP256BN_AMCL)

	// Success
	ppm, err := ppmFactory.NewPublicParametersManager(pp)
	require.NoError(t, err)
	assert.NotNil(t, ppm)

	// Invalid type
	_, err = ppmFactory.NewPublicParametersManager(&dmock.PublicParameters{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid public parameters type")
}

// TestWalletServiceFactory tests the zkatdlog wallet service factory, covering success and error paths.
func TestWalletServiceFactory(t *testing.T) {
	storageProvider := &imock.StorageProvider{}
	factory := driver.NewWalletServiceFactory(storageProvider)
	assert.Equal(t, core.DriverIdentifier(setup.DLogNoGHDriverName, setup.ProtocolV1), factory.Name)
	assert.NotNil(t, factory.Driver)

	wsFactory := factory.Driver
	tmsConfig := &dmock.Configuration{}
	tmsConfig.IDReturns(tdriver.TMSID{Network: "n1", Channel: "c1", Namespace: "ns1"})

	identityStore := &imock.IdentityStoreService{}
	identityStore.IteratorConfigurationsReturns(&mock2.IdentityConfigurationIterator{}, nil)
	keystore := &mock2.Keystore{}
	walletStore := &imock.WalletStoreService{}
	storageProvider.IdentityStoreReturns(identityStore, nil)
	storageProvider.KeystoreReturns(keystore, nil)
	storageProvider.WalletStoreReturns(walletStore, nil)

	issuerPK := testingHelper(t)
	pp, _ := setup.Setup(32, issuerPK, math3.FP256BN_AMCL)
	pp.AddIssuer([]byte("issuer-1"))
	pp.AddAuditor([]byte("auditor-1"))

	// Success
	ws, err := wsFactory.NewWalletService(tmsConfig, pp)
	require.NoError(t, err)
	assert.NotNil(t, ws)

	// Error path: IdentityStore failure
	storageProvider.IdentityStoreReturns(nil, errors.New("identity store error"))
	_, err = wsFactory.NewWalletService(tmsConfig, pp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open identity db")

	// Error path: Keystore failure
	storageProvider.IdentityStoreReturns(identityStore, nil)
	storageProvider.KeystoreReturns(nil, errors.New("keystore error"))
	_, err = wsFactory.NewWalletService(tmsConfig, pp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open keystore")

	// Error path: WalletStore failure
	storageProvider.KeystoreReturns(keystore, nil)
	storageProvider.WalletStoreReturns(nil, errors.New("wallet store error"))
	_, err = wsFactory.NewWalletService(tmsConfig, pp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get identity storage provider")
}

// TestDeserializers tests the zkatdlog deserializer and EIDRH deserializer creation.
func TestDeserializers(t *testing.T) {
	issuerPK := testingHelper(t)
	pp, _ := setup.Setup(32, issuerPK, math3.FP256BN_AMCL)
	d, err := driver.NewDeserializer(pp)
	require.NoError(t, err)
	assert.NotNil(t, d)

	ppd := &driver.PublicParamsDeserializer{}
	ppBytes, _ := pp.Serialize()
	res, err := ppd.DeserializePublicParams(ppBytes, setup.DLogNoGHDriverName, setup.ProtocolV1)
	require.NoError(t, err)
	assert.NotNil(t, res)

	eidrh := driver.NewEIDRHDeserializer()
	assert.NotNil(t, eidrh)
}
