/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	jsession "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/json/session"
	utilsession "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/session"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/session/mock"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/require"
)

func TestVersionedRecipientRequestRoundTrip(t *testing.T) {
	original := &RecipientRequest{
		TMSID:    token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"},
		WalletID: []byte("wallet"),
		MultiSig: true,
	}
	roundTripTTXMessage(t, jsession.TypeRecipientRequest, original, &RecipientRequest{})
}

func TestVersionedRecipientResponseRoundTrip(t *testing.T) {
	original := &RecipientData{
		Identity:  []byte("recipient"),
		AuditInfo: []byte("audit"),
	}
	roundTripTTXMessage(t, jsession.TypeRecipientResponse, original, &RecipientData{})
}

func TestVersionedExchangeRecipientRoundTrip(t *testing.T) {
	original := &ExchangeRecipientRequest{
		TMSID:    token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"},
		WalletID: []byte("wallet"),
	}
	roundTripTTXMessage(t, jsession.TypeExchangeRecipientRequest, original, &ExchangeRecipientRequest{})
}

func TestVersionedMultisigRecipientDataRoundTrip(t *testing.T) {
	original := &MultisigRecipientData{
		RecipientData: &token.RecipientData{Identity: []byte("ms")},
		Nodes:         []view.Identity{view.Identity("a")},
	}
	roundTripTTXMessage(t, jsession.TypeMultisigRecipientData, original, &MultisigRecipientData{})
}

func TestVersionedPolicyRecipientDataRoundTrip(t *testing.T) {
	original := &PolicyRecipientData{
		RecipientData: &token.RecipientData{Identity: []byte("pol")},
	}
	roundTripTTXMessage(t, jsession.TypePolicyRecipientData, original, &PolicyRecipientData{})
}

func TestVersionedWithdrawalRequestRoundTrip(t *testing.T) {
	original := &WithdrawalRequest{
		TMSID:     token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"},
		TokenType: "TOK",
		Amount:    42,
	}
	roundTripTTXMessage(t, jsession.TypeWithdrawalRequest, original, &WithdrawalRequest{})
}

func TestVersionedUpgradeAgreementRoundTrip(t *testing.T) {
	original := &UpgradeTokensAgreement{
		Challenge: []byte("challenge"),
		TMSID:     token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"},
	}
	roundTripTTXMessage(t, jsession.TypeUpgradeAgreement, original, &UpgradeTokensAgreement{})
}

func TestVersionedUpgradeRequestRoundTrip(t *testing.T) {
	original := &UpgradeTokensRequest{
		TMSID: token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"},
		Tokens: []token2.LedgerToken{
			{ID: token2.ID{TxId: "tx1"}},
		},
	}
	roundTripTTXMessage(t, jsession.TypeUpgradeRequest, original, &UpgradeTokensRequest{})
}

func roundTripTTXMessage(t *testing.T, msgType string, sent, received any) {
	t.Helper()

	var wire []byte
	mockSession := &mock.Session{}
	mockSession.SendWithContextStub = func(_ context.Context, payload []byte) error {
		wire = append([]byte(nil), payload...)

		return nil
	}

	s := utilsession.New(mockSession, t.Context(), jsession.JSONMarshaller{})
	require.NoError(t, jsession.SendTyped(s, t.Context(), sent, msgType))

	var env jsession.Envelope
	require.NoError(t, json.Unmarshal(wire, &env))
	require.Equal(t, jsession.CurrentVersion, env.Version)
	require.Equal(t, msgType, env.Type)

	ch := make(chan *view.Message, 1)
	ch <- &view.Message{Payload: wire, Status: int32(view.OK)}
	mockSession.ReceiveReturns(ch)

	require.NoError(t, jsession.ReceiveTypedWithTimeout(s, msgType, received, time.Second))

	sentBytes, err := json.Marshal(sent)
	require.NoError(t, err)
	recvBytes, err := json.Marshal(received)
	require.NoError(t, err)
	require.JSONEq(t, string(sentBytes), string(recvBytes))
}
