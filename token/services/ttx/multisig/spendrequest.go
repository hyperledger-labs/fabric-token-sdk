/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
)

type SpendRequest struct {
	Token *token.UnspentToken
}

func NewSpendRequestFromBytes(msg []byte) (*SpendRequest, error) {
	request := &SpendRequest{}
	err := json.Unmarshal(msg, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed unmarshalling spendRequest")
	}
	return request, nil
}

func ReceiveSpendRequest(context view.Context, opts ...ttx.TxOption) (*SpendRequest, error) {
	logger.Debugf("receive a new spendRequest...")
	requestBoxed, err := context.RunView(NewReceiveSpendRequestView(""), view.WithSameContext())
	if err != nil {
		return nil, err
	}
	request, ok := requestBoxed.(*SpendRequest)
	if !ok {
		return nil, errors.Errorf("received spendRequest of wrong type [%T]", request)
	}
	return request, nil
}

type ReceiveSpendRequestView struct {
	network string
}

func NewReceiveSpendRequestView(network string) *ReceiveSpendRequestView {
	return &ReceiveSpendRequestView{network: network}
}

func (f *ReceiveSpendRequestView) Call(context view.Context) (interface{}, error) {
	span := trace.SpanFromContext(context.Context())
	span.AddEvent("start_receive_spendRequest_view")
	defer span.AddEvent("end_receive_spendRequest_view")

	msg, err := ttx.ReadMessage(context.Session(), time.Minute*4)
	if err != nil {
		span.RecordError(err)
	}
	span.AddEvent("receive_tx")

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("ReceiveSpendRequestView: received spendRequest, len [%d][%s]", len(msg), hash.Hashable(msg))
	}
	if len(msg) == 0 {
		info := context.Session().Info()
		return nil, errors.Errorf("received empty message, session closed [%s:%v]", info.ID, info.Closed)
	}
	tx, err := NewSpendRequestFromBytes(msg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive spendRequest")
	}
	return tx, nil
}

type RequestSpendView struct {
	UnspentToken *token.UnspentToken
	CoOwners     []token2.Identity
}

func NewRequestSpendView(unspentToken *token.UnspentToken) *RequestSpendView {
	return &RequestSpendView{UnspentToken: unspentToken, CoOwners: nil}
}

func (r *RequestSpendView) Call(context view.Context) (interface{}, error) {
	panic("implement me")
}

func ApproveSpend(context view.Context, request *SpendRequest) (*Transaction, error) {
	panic("implement me")
}
