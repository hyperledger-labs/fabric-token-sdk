/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interactive

import (
	"bytes"
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
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
// the provided fakeJsonSession instead of the real comm-stack session.
func newServiceWithSession(fs *fakeJsonSession) *CertificationService {
	svc := NewCertificationService(&ResponderRegistryMock{}, &disabled.Provider{}, &BackendMock{})
	svc.sessionFactory = func(_ view.Context) session.JsonSession { return fs }

	return svc
}

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

// ---------------------------------------------------------------------------
// Wire-size guard — Call() must reject oversized messages before JSON decode.
// These tests validate the pre-deserialization size check added to address the
// reviewer's concern about allocate-then-reject memory exhaustion (PR #1498).
// ---------------------------------------------------------------------------

// TestCall_WireMessageTooLarge verifies that Call() rejects a message whose raw
// byte length exceeds MaxWireMessageBytes without attempting JSON decoding.
func TestCall_WireMessageTooLarge(t *testing.T) {
	oversized := bytes.Repeat([]byte("x"), MaxWireMessageBytes+1)
	svc := newServiceWithSession(&fakeJsonSession{raw: oversized})

	_, err := svc.Call(&fakeViewContext{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "wire message too large")
	assert.Contains(t, err.Error(), strconv.Itoa(MaxWireMessageBytes))
}

// TestCall_WireMessageAtLimit verifies that a message exactly at MaxWireMessageBytes
// is not rejected by the size guard (it may fail for other reasons, e.g. JSON parse).
func TestCall_WireMessageAtLimit(t *testing.T) {
	atLimit := bytes.Repeat([]byte("x"), MaxWireMessageBytes)
	svc := newServiceWithSession(&fakeJsonSession{raw: atLimit})

	_, err := svc.Call(&fakeViewContext{})

	require.Error(t, err)
	// Must NOT be a wire-size error — the guard must pass.
	assert.NotContains(t, err.Error(), "wire message too large")
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
	assert.Equal(t, MaxRequestBytes*2, MaxWireMessageBytes,
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
	v := NewCertificationRequestView(network, channel, namespace, certifier, ids...)

	assert.NotNil(t, v)
	assert.Equal(t, network, v.network)
	assert.Equal(t, channel, v.channel)
	assert.Equal(t, namespace, v.ns)
	assert.Equal(t, certifier, v.certifier)
	assert.Equal(t, ids, v.ids)
}
