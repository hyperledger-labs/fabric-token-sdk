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
	"github.com/stretchr/testify/require"
)

// migratedProtocolCase exercises the wire format each Phase-2 view uses via SendTyped/ReceiveTyped.
type migratedProtocolCase struct {
	name    string
	msgType string
	payload any
}

func TestMigratedProtocolMessages_RoundTrip(t *testing.T) {
	cases := []migratedProtocolCase{
		{
			name:    "recipient_req",
			msgType: jsession.TypeRecipientRequest,
			payload: map[string]any{"TMSID": map[string]string{"Network": "n"}, "WalletID": []byte("w")},
		},
		{
			name:    "recipient_resp",
			msgType: jsession.TypeRecipientResponse,
			payload: map[string]any{"Identity": []byte("id"), "AuditInfo": []byte("audit")},
		},
		{
			name:    "exchange_req",
			msgType: jsession.TypeExchangeRecipientRequest,
			payload: map[string]any{"WalletID": []byte("w")},
		},
		{
			name:    "exchange_resp",
			msgType: jsession.TypeExchangeRecipientResp,
			payload: map[string]any{"Identity": []byte("id")},
		},
		{
			name:    "multisig_data",
			msgType: jsession.TypeMultisigRecipientData,
			payload: map[string]any{"Nodes": []string{"a", "b"}},
		},
		{
			name:    "policy_data",
			msgType: jsession.TypePolicyRecipientData,
			payload: map[string]any{"Policy": "and"},
		},
		{
			name:    "withdrawal_req",
			msgType: jsession.TypeWithdrawalRequest,
			payload: map[string]any{"Amount": uint64(100)},
		},
		{
			name:    "upgrade_agree",
			msgType: jsession.TypeUpgradeAgreement,
			payload: map[string]any{"Challenge": []byte("c")},
		},
		{
			name:    "upgrade_req",
			msgType: jsession.TypeUpgradeRequest,
			payload: map[string]any{"NotAnonymous": true},
		},
		{
			name:    "spend_req",
			msgType: jsession.TypeSpendRequest,
			payload: map[string]any{"Token": map[string]string{"ID": "tok1"}},
		},
		{
			name:    "spend_resp",
			msgType: jsession.TypeSpendResponse,
			payload: map[string]any{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var wire []byte
			mockSession := &mock.Session{}
			mockSession.SendWithContextStub = func(_ context.Context, payload []byte) error {
				wire = append([]byte(nil), payload...)

				return nil
			}

			s := utilsession.New(mockSession, t.Context(), jsession.JSONMarshaller{})
			require.NoError(t, jsession.SendTyped(s, t.Context(), tc.payload, tc.msgType))

			ch := make(chan *view.Message, 1)
			ch <- &view.Message{Payload: wire, Status: int32(view.OK)}
			mockSession.ReceiveReturns(ch)

			var dst map[string]any
			require.NoError(t, jsession.ReceiveTypedWithTimeout(s, tc.msgType, &dst, time.Second))
		})
	}
}

func TestMigratedProtocolMessages_ErrorPaths(t *testing.T) {
	t.Run("missing_version", func(t *testing.T) {
		raw, _ := json.Marshal(jsession.Envelope{Version: 0, Type: jsession.TypeRecipientRequest, Body: json.RawMessage(`{}`)})
		assertReceiveErrorIs(t, raw, jsession.TypeRecipientRequest, jsession.ErrMissingVersion)
	})

	t.Run("future_version", func(t *testing.T) {
		raw, _ := json.Marshal(jsession.Envelope{Version: 99, Type: jsession.TypeWithdrawalRequest, Body: json.RawMessage(`{}`)})
		assertReceiveErrorIs(t, raw, jsession.TypeWithdrawalRequest, jsession.ErrVersionMismatch)
	})

	t.Run("type_mismatch", func(t *testing.T) {
		raw, _ := json.Marshal(jsession.Envelope{
			Version: jsession.CurrentVersion,
			Type:    jsession.TypeUpgradeRequest,
			Body:    json.RawMessage(`{}`),
		})
		assertReceiveErrorIs(t, raw, jsession.TypeUpgradeAgreement, jsession.ErrTypeMismatch)
	})

	t.Run("unversioned_legacy_body", func(t *testing.T) {
		legacy, _ := json.Marshal(map[string]string{"legacy": "true"})
		assertReceiveErrorIs(t, legacy, jsession.TypeRecipientRequest, jsession.ErrMissingVersion)
	})
}

func assertReceiveErrorIs(t *testing.T, wire []byte, expectedType string, target error) {
	t.Helper()

	mockSession := &mock.Session{}
	ch := make(chan *view.Message, 1)
	ch <- &view.Message{Payload: wire, Status: int32(view.OK)}
	mockSession.ReceiveReturns(ch)

	s := utilsession.New(mockSession, t.Context(), jsession.JSONMarshaller{})
	var dst map[string]any
	err := jsession.ReceiveTypedWithTimeout(s, expectedType, &dst, time.Second)
	require.Error(t, err)
	require.ErrorIs(t, err, target)
}
