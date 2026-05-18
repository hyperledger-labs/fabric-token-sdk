/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package session_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	jsession "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/json/session"
	utilsession "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/session"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/session/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Envelope marshal / unmarshal ---

func TestWrapEnvelope(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	env, err := jsession.WrapEnvelope(&payload{Name: "alice"}, jsession.TypeRecipientRequest)
	require.NoError(t, err)
	assert.Equal(t, jsession.CurrentVersion, env.Version)
	assert.Equal(t, jsession.TypeRecipientRequest, env.Type)

	var p payload
	require.NoError(t, json.Unmarshal(env.Body, &p))
	assert.Equal(t, "alice", p.Name)
}

func TestWrapEnvelope_MarshalError(t *testing.T) {
	_, err := jsession.WrapEnvelope(make(chan int), jsession.TypeRecipientRequest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal envelope body")
}

func TestUnwrapEnvelope_Valid(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"key": "val"})
	raw, _ := json.Marshal(jsession.Envelope{
		Version: jsession.CurrentVersion,
		Type:    jsession.TypeRecipientRequest,
		Body:    body,
	})

	env, err := jsession.UnwrapEnvelope(raw, jsession.TypeRecipientRequest)
	require.NoError(t, err)
	assert.Equal(t, jsession.CurrentVersion, env.Version)
	assert.Equal(t, jsession.TypeRecipientRequest, env.Type)
}

func TestUnwrapEnvelope_SkipTypeCheck(t *testing.T) {
	body, _ := json.Marshal("hello")
	raw, _ := json.Marshal(jsession.Envelope{
		Version: jsession.CurrentVersion,
		Type:    jsession.TypeWithdrawalRequest,
		Body:    body,
	})

	env, err := jsession.UnwrapEnvelope(raw, "")
	require.NoError(t, err)
	assert.Equal(t, jsession.TypeWithdrawalRequest, env.Type)
}

// --- Version validation ---

func TestUnwrapEnvelope_MissingVersion(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{"t": "recipient_req", "b": "{}"})

	_, err := jsession.UnwrapEnvelope(raw, "")
	require.Error(t, err)
	require.ErrorIs(t, err, jsession.ErrMissingVersion)
}

func TestUnwrapEnvelope_FutureVersion(t *testing.T) {
	raw, _ := json.Marshal(jsession.Envelope{
		Version: 99,
		Type:    jsession.TypeRecipientRequest,
		Body:    json.RawMessage(`{}`),
	})

	_, err := jsession.UnwrapEnvelope(raw, "")
	require.Error(t, err)
	require.ErrorIs(t, err, jsession.ErrVersionMismatch)

	var ve *jsession.VersionError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, jsession.CurrentVersion, ve.Expected)
	assert.Equal(t, uint32(99), ve.Received)
}

// --- Type validation ---

func TestUnwrapEnvelope_EmptyType(t *testing.T) {
	raw, _ := json.Marshal(jsession.Envelope{
		Version: jsession.CurrentVersion,
		Type:    "",
		Body:    json.RawMessage(`{}`),
	})

	_, err := jsession.UnwrapEnvelope(raw, "")
	require.Error(t, err)
	require.ErrorIs(t, err, jsession.ErrInvalidEnvelope)
	assert.Contains(t, err.Error(), "type field is empty")
}

func TestUnwrapEnvelope_TypeMismatch(t *testing.T) {
	raw, _ := json.Marshal(jsession.Envelope{
		Version: jsession.CurrentVersion,
		Type:    jsession.TypeWithdrawalRequest,
		Body:    json.RawMessage(`{}`),
	})

	_, err := jsession.UnwrapEnvelope(raw, jsession.TypeRecipientRequest)
	require.Error(t, err)
	require.ErrorIs(t, err, jsession.ErrTypeMismatch)
}

// --- Malformed JSON ---

func TestUnwrapEnvelope_MalformedJSON(t *testing.T) {
	_, err := jsession.UnwrapEnvelope([]byte(`not json`), "")
	require.Error(t, err)
	require.ErrorIs(t, err, jsession.ErrInvalidEnvelope)
}

// --- UnwrapBody ---

func TestUnwrapBody(t *testing.T) {
	type msg struct {
		Value int `json:"value"`
	}
	body, _ := json.Marshal(msg{Value: 42})
	raw, _ := json.Marshal(jsession.Envelope{
		Version: jsession.CurrentVersion,
		Type:    jsession.TypeUpgradeRequest,
		Body:    body,
	})

	var dst msg
	require.NoError(t, jsession.UnwrapBody(raw, jsession.TypeUpgradeRequest, &dst))
	assert.Equal(t, 42, dst.Value)
}

func TestUnwrapBody_VersionMismatch(t *testing.T) {
	raw, _ := json.Marshal(jsession.Envelope{
		Version: 2,
		Type:    jsession.TypeUpgradeRequest,
		Body:    json.RawMessage(`{}`),
	})

	var dst struct{}
	err := jsession.UnwrapBody(raw, jsession.TypeUpgradeRequest, &dst)
	require.Error(t, err)
	require.ErrorIs(t, err, jsession.ErrVersionMismatch)
}

// --- Envelope.Validate ---

func TestEnvelope_Validate(t *testing.T) {
	env := &jsession.Envelope{
		Version: jsession.CurrentVersion,
		Type:    jsession.TypeSpendRequest,
	}
	require.NoError(t, env.Validate(jsession.TypeSpendRequest))
	require.Error(t, env.Validate(jsession.TypeSpendResponse))
}

// --- VersionError ---

func TestVersionError_Error(t *testing.T) {
	ve := &jsession.VersionError{Expected: 1, Received: 5}
	assert.Contains(t, ve.Error(), "expected 1")
	assert.Contains(t, ve.Error(), "received 5")
}

func TestVersionError_Is(t *testing.T) {
	ve := &jsession.VersionError{Expected: 1, Received: 2}
	require.ErrorIs(t, ve, jsession.ErrVersionMismatch)
	assert.False(t, ve.Is(jsession.ErrInvalidEnvelope))
}

// --- SendTyped / ReceiveTyped over mock session ---

func TestSendTyped(t *testing.T) {
	mockSession := &mock.Session{}
	var captured []byte
	mockSession.SendWithContextStub = func(_ context.Context, payload []byte) error {
		captured = payload

		return nil
	}

	s := utilsession.New(mockSession, t.Context(), jsession.JSONMarshaller{})

	type req struct {
		ID string `json:"id"`
	}
	err := jsession.SendTyped(s, t.Context(), &req{ID: "abc"}, jsession.TypeRecipientRequest)
	require.NoError(t, err)

	var env jsession.Envelope
	require.NoError(t, json.Unmarshal(captured, &env))
	assert.Equal(t, jsession.CurrentVersion, env.Version)
	assert.Equal(t, jsession.TypeRecipientRequest, env.Type)

	var body req
	require.NoError(t, json.Unmarshal(env.Body, &body))
	assert.Equal(t, "abc", body.ID)
}

func TestReceiveTypedWithTimeout_Success(t *testing.T) {
	type resp struct {
		Name string `json:"name"`
	}

	body, _ := json.Marshal(resp{Name: "bob"})
	raw, _ := json.Marshal(jsession.Envelope{
		Version: jsession.CurrentVersion,
		Type:    jsession.TypeRecipientResponse,
		Body:    body,
	})

	mockSession := &mock.Session{}
	ch := make(chan *view.Message, 1)
	ch <- &view.Message{Payload: raw, Status: int32(view.OK)}
	mockSession.ReceiveReturns(ch)

	s := utilsession.New(mockSession, t.Context(), jsession.JSONMarshaller{})

	var dst resp
	err := jsession.ReceiveTypedWithTimeout(s, jsession.TypeRecipientResponse, &dst, 1*time.Second)
	require.NoError(t, err)
	assert.Equal(t, "bob", dst.Name)
}

func TestReceiveTypedWithTimeout_VersionMismatch(t *testing.T) {
	raw, _ := json.Marshal(jsession.Envelope{
		Version: 99,
		Type:    jsession.TypeRecipientResponse,
		Body:    json.RawMessage(`{}`),
	})

	mockSession := &mock.Session{}
	ch := make(chan *view.Message, 1)
	ch <- &view.Message{Payload: raw, Status: int32(view.OK)}
	mockSession.ReceiveReturns(ch)

	s := utilsession.New(mockSession, t.Context(), jsession.JSONMarshaller{})

	var dst struct{}
	err := jsession.ReceiveTypedWithTimeout(s, jsession.TypeRecipientResponse, &dst, 1*time.Second)
	require.Error(t, err)
	require.ErrorIs(t, err, jsession.ErrVersionMismatch)
}

func TestReceiveTypedWithTimeout_TypeMismatch(t *testing.T) {
	raw, _ := json.Marshal(jsession.Envelope{
		Version: jsession.CurrentVersion,
		Type:    jsession.TypeWithdrawalRequest,
		Body:    json.RawMessage(`{}`),
	})

	mockSession := &mock.Session{}
	ch := make(chan *view.Message, 1)
	ch <- &view.Message{Payload: raw, Status: int32(view.OK)}
	mockSession.ReceiveReturns(ch)

	s := utilsession.New(mockSession, t.Context(), jsession.JSONMarshaller{})

	var dst struct{}
	err := jsession.ReceiveTypedWithTimeout(s, jsession.TypeRecipientRequest, &dst, 1*time.Second)
	require.Error(t, err)
	require.ErrorIs(t, err, jsession.ErrTypeMismatch)
}

func TestReceiveTypedWithTimeout_Timeout(t *testing.T) {
	mockSession := &mock.Session{}
	ch := make(chan *view.Message)
	mockSession.ReceiveReturns(ch)
	mockSession.InfoReturns(view.SessionInfo{ID: "test"})

	s := utilsession.New(mockSession, t.Context(), jsession.JSONMarshaller{})

	var dst struct{}
	err := jsession.ReceiveTypedWithTimeout(s, jsession.TypeRecipientRequest, &dst, 1*time.Millisecond)
	require.Error(t, err)
	require.ErrorIs(t, err, utilsession.ErrTimeout)
}

func TestReceiveTyped_UsesDefaultTimeout(t *testing.T) {
	mockSession := &mock.Session{}
	ch := make(chan *view.Message)
	mockSession.ReceiveReturns(ch)
	mockSession.InfoReturns(view.SessionInfo{ID: "test"})

	s := utilsession.New(mockSession, t.Context(), jsession.JSONMarshaller{})

	var dst struct{}
	err := jsession.ReceiveTyped(s, jsession.TypeRecipientRequest, &dst)
	require.Error(t, err)
	require.ErrorIs(t, err, utilsession.ErrTimeout)
}

// --- Large body ---

func TestRoundTrip_LargeBody(t *testing.T) {
	large := make([]byte, 64*1024)
	for i := range large {
		large[i] = byte(i % 256)
	}

	env, err := jsession.WrapEnvelope(large, jsession.TypeUpgradeRequest)
	require.NoError(t, err)

	raw, err := json.Marshal(env)
	require.NoError(t, err)

	var dst []byte
	require.NoError(t, jsession.UnwrapBody(raw, jsession.TypeUpgradeRequest, &dst))
	assert.Equal(t, large, dst)
}

// --- VersionError with Message ---

func TestVersionError_WithMessage(t *testing.T) {
	ve := &jsession.VersionError{Expected: 1, Received: 3, Message: "upgrade required"}
	assert.Contains(t, ve.Error(), "expected 1")
	assert.Contains(t, ve.Error(), "received 3")
	assert.Contains(t, ve.Error(), "upgrade required")
}

// --- IsCompatible ---

func TestIsCompatible(t *testing.T) {
	assert.True(t, jsession.IsCompatible(1, 1))
	assert.False(t, jsession.IsCompatible(1, 2))
	assert.False(t, jsession.IsCompatible(2, 1))
	assert.False(t, jsession.IsCompatible(0, 0))
}

// --- Concurrent send/receive ---

func TestConcurrentSendReceive(t *testing.T) {
	const goroutines = 20

	mockSession := &mock.Session{}
	mockSession.SendWithContextStub = func(_ context.Context, _ []byte) error {
		return nil
	}

	body, _ := json.Marshal(map[string]string{"k": "v"})
	raw, _ := json.Marshal(jsession.Envelope{
		Version: jsession.CurrentVersion,
		Type:    jsession.TypeRecipientRequest,
		Body:    body,
	})

	errs := make(chan error, goroutines*2)
	for range goroutines {
		go func() {
			s := utilsession.New(mockSession, t.Context(), jsession.JSONMarshaller{})
			errs <- jsession.SendTyped(s, t.Context(), map[string]string{"k": "v"}, jsession.TypeRecipientRequest)
		}()
		go func() {
			_, err := jsession.UnwrapEnvelope(raw, jsession.TypeRecipientRequest)
			errs <- err
		}()
	}
	for range goroutines * 2 {
		assert.NoError(t, <-errs)
	}
}

// --- Type constants exist ---

func TestTypeConstants(t *testing.T) {
	types := []string{
		jsession.TypeRecipientRequest,
		jsession.TypeRecipientResponse,
		jsession.TypeExchangeRecipientRequest,
		jsession.TypeExchangeRecipientResp,
		jsession.TypeMultisigRecipientData,
		jsession.TypePolicyRecipientData,
		jsession.TypeWithdrawalRequest,
		jsession.TypeUpgradeAgreement,
		jsession.TypeUpgradeRequest,
		jsession.TypeSpendRequest,
		jsession.TypeSpendResponse,
	}
	seen := make(map[string]bool, len(types))
	for _, typ := range types {
		assert.NotEmpty(t, typ)
		assert.False(t, seen[typ], "duplicate type constant: %s", typ)
		seen[typ] = true
	}
}

// --- Performance benchmarks ---

func BenchmarkWrapEnvelope(b *testing.B) {
	payload := map[string]string{"name": "alice", "role": "owner"}
	for b.Loop() {
		_, _ = jsession.WrapEnvelope(payload, jsession.TypeRecipientRequest)
	}
}

func BenchmarkUnwrapEnvelope(b *testing.B) {
	body, _ := json.Marshal(map[string]string{"name": "alice"})
	raw, _ := json.Marshal(jsession.Envelope{
		Version: jsession.CurrentVersion,
		Type:    jsession.TypeRecipientRequest,
		Body:    body,
	})
	for b.Loop() {
		_, _ = jsession.UnwrapEnvelope(raw, jsession.TypeRecipientRequest)
	}
}

func BenchmarkRoundTrip(b *testing.B) {
	type msg struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	for b.Loop() {
		env, _ := jsession.WrapEnvelope(&msg{Name: "alice", Value: 42}, jsession.TypeRecipientRequest)
		raw, _ := json.Marshal(env)
		var dst msg
		_ = jsession.UnwrapBody(raw, jsession.TypeRecipientRequest, &dst)
	}
}
