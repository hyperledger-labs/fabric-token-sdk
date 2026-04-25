/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interactive_test

import (
	"bytes"
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/interactive"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier/interactive/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/json/session"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

// ---------------------------------------------------------------------------
// fakeJsonSession — minimal session.JsonSession for unit-testing Call().
// Only ReceiveRaw and Send are used in the paths exercised by these tests.
// ---------------------------------------------------------------------------

type fakeJsonSession struct {
	raw    []byte // bytes returned by ReceiveRaw
	rawErr error  // error returned by ReceiveRaw
}

func (f *fakeJsonSession) ReceiveRaw() ([]byte, error) { return f.raw, f.rawErr }
func (f *fakeJsonSession) ReceiveRawWithTimeout(_ time.Duration) ([]byte, error) {
	return f.raw, f.rawErr
}
func (f *fakeJsonSession) Receive(_ interface{}) error                             { return nil }
func (f *fakeJsonSession) ReceiveWithTimeout(_ interface{}, _ time.Duration) error { return nil }
func (f *fakeJsonSession) Send(_ interface{}) error                                { return nil }
func (f *fakeJsonSession) SendRaw(_ context.Context, _ []byte) error               { return nil }
func (f *fakeJsonSession) SendWithContext(_ context.Context, _ interface{}) error  { return nil }
func (f *fakeJsonSession) SendError(_ string) error                                { return nil }
func (f *fakeJsonSession) SendErrorWithContext(_ context.Context, _ string) error  { return nil }
func (f *fakeJsonSession) Info() view.SessionInfo                                  { return view.SessionInfo{} }
func (f *fakeJsonSession) Session() view.Session                                   { return nil }

var _ session.JsonSession = (*fakeJsonSession)(nil)

// fakeViewContext satisfies view.Context for tests that only need ID() and the
// session factory — all other methods panic if called unexpectedly.
type fakeViewContext struct{}

func (f *fakeViewContext) ID() string { return "test-context-id" }

// Unused methods — panic to surface accidental calls in tests.
func (f *fakeViewContext) StartSpanFrom(_ context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
	panic("StartSpanFrom called unexpectedly")
}
func (f *fakeViewContext) GetService(_ interface{}) (interface{}, error) {
	panic("GetService called unexpectedly")
}
func (f *fakeViewContext) RunView(_ view.View, _ ...view.RunViewOption) (interface{}, error) {
	panic("RunView called unexpectedly")
}
func (f *fakeViewContext) Me() view.Identity         { return nil }
func (f *fakeViewContext) IsMe(_ view.Identity) bool { return false }
func (f *fakeViewContext) Initiator() view.View      { return nil }
func (f *fakeViewContext) GetSession(_ view.View, _ view.Identity, _ ...view.View) (view.Session, error) {
	return nil, nil
}
func (f *fakeViewContext) GetSessionByID(_ string, _ view.Identity) (view.Session, error) {
	return nil, nil
}
func (f *fakeViewContext) Session() view.Session    { return nil }
func (f *fakeViewContext) Context() context.Context { return context.Background() }
func (f *fakeViewContext) OnError(_ func())         {}

// newServiceWithSession builds a CertificationService whose Call() will use
// the provided fakeJsonSession wrapped in a SizeLimitedJsonSession so that
// the wire-size guard fires exactly as in production.
func newServiceWithSession(fs *fakeJsonSession) *interactive.CertificationService {
	svc := interactive.NewCertificationService(&mock.ResponderRegistryMock{}, &disabled.Provider{}, &mock.BackendMock{})
	interactive.SetSessionFactory(svc, func(_ view.Context) session.JsonSession {
		return session.NewSizeLimitedSession(fs, interactive.MaxWireMessageBytes)
	})

	return svc
}

// TestNewCertificationService verifies construction of a new CertificationService.
func TestNewCertificationService(t *testing.T) {
	responderRegistry := &mock.ResponderRegistryMock{}
	backend := &mock.BackendMock{}
	mp := &disabled.Provider{}

	service := interactive.NewCertificationService(responderRegistry, mp, backend)

	assert.NotNil(t, service)
	assert.Equal(t, responderRegistry, service.ResponderRegistry)
	assert.Equal(t, backend, interactive.ServiceBackend(service))
	assert.NotNil(t, interactive.ServiceWallets(service))
	assert.NotNil(t, interactive.ServiceMetrics(service))
	assert.Empty(t, interactive.ServiceWallets(service))
}

// TestCertificationService_Start_Success verifies that Start succeeds when
// RegisterResponder succeeds.
func TestCertificationService_Start_Success(t *testing.T) {
	responderRegistry := &mock.ResponderRegistryMock{}
	backend := &mock.BackendMock{}
	mp := &disabled.Provider{}

	service := interactive.NewCertificationService(responderRegistry, mp, backend)

	err := service.Start()
	require.NoError(t, err)
	assert.Equal(t, 1, responderRegistry.RegisterResponderCallCount())
}

// TestCertificationService_Start_RegistrationError verifies that Start propagates
// errors returned by RegisterResponder.
func TestCertificationService_Start_RegistrationError(t *testing.T) {
	expectedErr := errors.New("registration failed")
	responderRegistry := &mock.ResponderRegistryMock{}
	responderRegistry.RegisterResponderReturns(expectedErr)

	backend := &mock.BackendMock{}
	mp := &disabled.Provider{}

	service := interactive.NewCertificationService(responderRegistry, mp, backend)

	err := service.Start()
	require.ErrorIs(t, err, expectedErr)
	assert.Equal(t, 1, responderRegistry.RegisterResponderCallCount())
}

// TestCertificationService_Start_OnlyOnce verifies that repeated Start calls only
// register the responder once.
func TestCertificationService_Start_OnlyOnce(t *testing.T) {
	responderRegistry := &mock.ResponderRegistryMock{}
	backend := &mock.BackendMock{}
	mp := &disabled.Provider{}

	service := interactive.NewCertificationService(responderRegistry, mp, backend)

	require.NoError(t, service.Start())
	require.NoError(t, service.Start())
	require.NoError(t, service.Start())

	// RegisterResponder must have been called exactly once.
	assert.Equal(t, 1, responderRegistry.RegisterResponderCallCount())
}

// TestCertificationRequest_String verifies the String summary.
func TestCertificationRequest_String(t *testing.T) {
	cr := &interactive.CertificationRequest{
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

// ---------------------------------------------------------------------------
// Wire-size guard — Call() must reject oversized messages before JSON decode.
// The guard lives in session.SizeLimitedJsonSession; these tests verify it is
// wired up correctly through the service's sessionFactory.
// ---------------------------------------------------------------------------

// TestCall_WireMessageTooLarge verifies that Call() rejects a message whose raw
// byte length exceeds MaxWireMessageBytes without attempting JSON decoding.
func TestCall_WireMessageTooLarge(t *testing.T) {
	oversized := bytes.Repeat([]byte("x"), interactive.MaxWireMessageBytes+1)
	svc := newServiceWithSession(&fakeJsonSession{raw: oversized})

	_, err := svc.Call(&fakeViewContext{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "message too large")
	assert.Contains(t, err.Error(), strconv.Itoa(interactive.MaxWireMessageBytes))
}

// TestCall_WireMessageAtLimit verifies that a message exactly at MaxWireMessageBytes
// is not rejected by the size guard (it may fail for other reasons, e.g. JSON parse).
func TestCall_WireMessageAtLimit(t *testing.T) {
	atLimit := bytes.Repeat([]byte("x"), interactive.MaxWireMessageBytes)
	svc := newServiceWithSession(&fakeJsonSession{raw: atLimit})

	_, err := svc.Call(&fakeViewContext{})

	require.Error(t, err)
	// Must NOT be a size-guard error — the guard must pass.
	assert.NotContains(t, err.Error(), "message too large")
}

// TestCall_ReceiveRawError verifies that a transport error from ReceiveRaw is
// propagated with a descriptive message.
func TestCall_ReceiveRawError(t *testing.T) {
	transportErr := errors.New("connection reset by peer")
	svc := newServiceWithSession(&fakeJsonSession{rawErr: transportErr})

	_, err := svc.Call(&fakeViewContext{})

	require.Error(t, err)
	require.ErrorIs(t, err, transportErr)
	assert.Contains(t, err.Error(), "failed receiving certification request")
}

// TestMaxWireMessageBytes_IsDoubleMaxRequestBytes documents and enforces the
// expected relationship between the two size constants.
func TestMaxWireMessageBytes_IsDoubleMaxRequestBytes(t *testing.T) {
	assert.Equal(t, interactive.MaxRequestBytes*2, interactive.MaxWireMessageBytes,
		"MaxWireMessageBytes should be 2× MaxRequestBytes to accommodate base64 overhead and ID encoding")
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
	v := interactive.NewCertificationRequestView(network, channel, namespace, certifier, ids...)

	assert.NotNil(t, v)
	assert.Equal(t, network, interactive.CRVNetwork(v))
	assert.Equal(t, channel, interactive.CRVChannel(v))
	assert.Equal(t, namespace, interactive.CRVNamespace(v))
	assert.Equal(t, certifier, interactive.CRVCertifier(v))
	assert.Equal(t, ids, interactive.CRVIDs(v))
}

// ---------------------------------------------------------------------------
// BackendMock helper methods
// ---------------------------------------------------------------------------

// TestBackendMock_LoadHelpers exercises the counterfeiter-generated helper
// methods on BackendMock that are not exercised by higher-level tests.
func TestBackendMock_LoadHelpers(t *testing.T) {
	cr := &interactive.CertificationRequest{Channel: "ch", Namespace: "ns"}

	// LoadReturns + Load + LoadCallCount + LoadArgsForCall + Invocations
	backend := &mock.BackendMock{}
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
	backend2 := &mock.BackendMock{}
	stubCalled := false
	backend2.LoadCalls(func(_ view.Context, _ *interactive.CertificationRequest) ([][]byte, error) {
		stubCalled = true

		return [][]byte{[]byte("from-stub")}, nil
	})
	res, err2 := backend2.Load(nil, cr)
	require.NoError(t, err2)
	assert.True(t, stubCalled, "stub should have been called")
	assert.Len(t, res, 1)

	// LoadReturnsOnCall — first call succeeds, second returns an error.
	backend3 := &mock.BackendMock{}
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
	m := &mock.ResponderRegistryMock{}

	// RegisterResponderCalls — installs a stub.
	stubCalled := false
	m.RegisterResponderCalls(func(_ view.View, _ interface{}) error {
		stubCalled = true

		return nil
	})
	err := m.RegisterResponder(nil, nil)
	require.NoError(t, err)
	assert.True(t, stubCalled, "stub should have been invoked")

	// RegisterResponderArgsForCall
	gotView, gotInitiatedBy := m.RegisterResponderArgsForCall(0)
	assert.Nil(t, gotView)
	assert.Nil(t, gotInitiatedBy)

	// Invocations
	inv := m.Invocations()
	assert.Contains(t, inv, "RegisterResponder")

	// RegisterResponderReturnsOnCall — first call returns a specific error.
	mock2 := &mock.ResponderRegistryMock{}
	customErr := errors.New("per-call error")
	mock2.RegisterResponderReturnsOnCall(0, customErr)

	err2 := mock2.RegisterResponder(nil, nil)
	require.ErrorIs(t, err2, customErr)
}
