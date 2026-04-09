/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interactive

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: We use counterfeiter-generated mocks that are in the same package (not a subpackage).
// This avoids import cycles that would occur if mocks were in interactive/mock, since the
// Backend interface references *CertificationRequest from the interactive package.

// TestNewCertificationService verifies construction of a new CertificationService.
func TestNewCertificationService(t *testing.T) {
	responderRegistry := &ResponderRegistryMock{}
	backend := &BackendMock{}
	mp := &disabled.Provider{}

	service := NewCertificationService(responderRegistry, mp, backend)

	assert.NotNil(t, service)
	assert.Equal(t, responderRegistry, service.ResponderRegistry)
	assert.Equal(t, backend, service.backend)
	assert.NotNil(t, service.wallets)
	assert.NotNil(t, service.metrics)
	assert.Empty(t, service.wallets)
}

// TestCertificationService_Start_Success verifies that Start succeeds when
// RegisterResponder succeeds.
func TestCertificationService_Start_Success(t *testing.T) {
	responderRegistry := &ResponderRegistryMock{}
	backend := &BackendMock{}
	mp := &disabled.Provider{}

	service := NewCertificationService(responderRegistry, mp, backend)

	err := service.Start()
	require.NoError(t, err)
	assert.Equal(t, 1, responderRegistry.RegisterResponderCallCount())
}

// TestCertificationService_Start_RegistrationError verifies that Start propagates
// errors returned by RegisterResponder.
func TestCertificationService_Start_RegistrationError(t *testing.T) {
	expectedErr := errors.New("registration failed")
	responderRegistry := &ResponderRegistryMock{}
	responderRegistry.RegisterResponderReturns(expectedErr)

	backend := &BackendMock{}
	mp := &disabled.Provider{}

	service := NewCertificationService(responderRegistry, mp, backend)

	err := service.Start()
	require.ErrorIs(t, err, expectedErr)
	assert.Equal(t, 1, responderRegistry.RegisterResponderCallCount())
}

// TestCertificationService_Start_OnlyOnce verifies that repeated Start calls only
// register the responder once.
func TestCertificationService_Start_OnlyOnce(t *testing.T) {
	responderRegistry := &ResponderRegistryMock{}
	backend := &BackendMock{}
	mp := &disabled.Provider{}

	service := NewCertificationService(responderRegistry, mp, backend)

	require.NoError(t, service.Start())
	require.NoError(t, service.Start())
	require.NoError(t, service.Start())

	// RegisterResponder must have been called exactly once.
	assert.Equal(t, 1, responderRegistry.RegisterResponderCallCount())
}

// TestCertificationRequest_String verifies the String summary.
func TestCertificationRequest_String(t *testing.T) {
	cr := &CertificationRequest{
		Network:   "test-network",
		Channel:   "test-channel",
		Namespace: "test-namespace",
		IDs:       nil,
		Request:   []byte("test-request"),
	}

	str := cr.String()
	assert.NotEmpty(t, str, "String() should return non-empty string")
	assert.Contains(t, str, "CertificationRequest")
	assert.Contains(t, str, "test-channel")
	assert.Contains(t, str, "test-namespace")
}

// TestNewCertificationRequestView verifies construction of a CertificationRequestView.
func TestNewCertificationRequestView(t *testing.T) {
	channel := "test-channel"
	namespace := "test-namespace"
	certifier := view.Identity([]byte("certifier-identity"))
	ids := []*token.ID{
		{TxId: "tx1", Index: 0},
		{TxId: "tx2", Index: 1},
	}

	network := "test-network"
	v := NewCertificationRequestView(network, channel, namespace, certifier, ids...)

	assert.NotNil(t, v)
	assert.Equal(t, network, v.network)
	assert.Equal(t, channel, v.channel)
	assert.Equal(t, namespace, v.ns)
	assert.Equal(t, certifier, v.certifier)
	assert.Equal(t, ids, v.ids)
}

// ---------------------------------------------------------------------------
// BackendMock helper methods
// ---------------------------------------------------------------------------

// TestBackendMock_LoadHelpers exercises the counterfeiter-generated helper
// methods on BackendMock that are not exercised by higher-level tests.
func TestBackendMock_LoadHelpers(t *testing.T) {
	cr := &CertificationRequest{Channel: "ch", Namespace: "ns"}

	// LoadReturns + Load + LoadCallCount + LoadArgsForCall + Invocations
	backend := &BackendMock{}
	backend.LoadReturns([][]byte{[]byte("cert")}, nil)

	result, err := backend.Load(nil, cr)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 1, backend.LoadCallCount())

	gotCtx, gotCR := backend.LoadArgsForCall(0)
	assert.Nil(t, gotCtx)
	assert.Equal(t, cr, gotCR)

	inv := backend.Invocations()
	assert.Contains(t, inv, "Load")

	// LoadCalls — installs a stub that takes over from the default return value.
	backend2 := &BackendMock{}
	stubCalled := false
	backend2.LoadCalls(func(_ view.Context, _ *CertificationRequest) ([][]byte, error) {
		stubCalled = true

		return [][]byte{[]byte("from-stub")}, nil
	})
	res, err2 := backend2.Load(nil, cr)
	require.NoError(t, err2)
	assert.True(t, stubCalled, "stub should have been called")
	assert.Len(t, res, 1)

	// LoadReturnsOnCall — first call succeeds, second returns an error.
	backend3 := &BackendMock{}
	backend3.LoadReturnsOnCall(0, [][]byte{[]byte("first")}, nil)
	backend3.LoadReturnsOnCall(1, nil, errors.New("second call error"))

	res0, err0 := backend3.Load(nil, cr)
	require.NoError(t, err0)
	assert.Len(t, res0, 1)

	_, err1 := backend3.Load(nil, cr)
	require.Error(t, err1)
	assert.Equal(t, 2, backend3.LoadCallCount())
}

// ---------------------------------------------------------------------------
// ResponderRegistryMock helper methods
// ---------------------------------------------------------------------------

// TestResponderRegistryMock_HelperMethods exercises the counterfeiter helper
// methods that are not covered by the Start tests.
func TestResponderRegistryMock_HelperMethods(t *testing.T) {
	mock := &ResponderRegistryMock{}

	// RegisterResponderCalls — installs a stub.
	stubCalled := false
	mock.RegisterResponderCalls(func(_ view.View, _ interface{}) error {
		stubCalled = true

		return nil
	})
	err := mock.RegisterResponder(nil, nil)
	require.NoError(t, err)
	assert.True(t, stubCalled, "stub should have been invoked")

	// RegisterResponderArgsForCall
	gotView, gotInitiatedBy := mock.RegisterResponderArgsForCall(0)
	assert.Nil(t, gotView)
	assert.Nil(t, gotInitiatedBy)

	// Invocations
	inv := mock.Invocations()
	assert.Contains(t, inv, "RegisterResponder")

	// RegisterResponderReturnsOnCall — first call returns a specific error.
	mock2 := &ResponderRegistryMock{}
	customErr := errors.New("per-call error")
	mock2.RegisterResponderReturnsOnCall(0, customErr)

	err2 := mock2.RegisterResponder(nil, nil)
	require.ErrorIs(t, err2, customErr)
}
