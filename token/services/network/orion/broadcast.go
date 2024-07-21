/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"fmt"
	"strings"
	"time"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	session2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type BroadcastRequest struct {
	Network string
	Blob    []byte
}

type BroadcastResponse struct {
	Err string
}

type BroadcastView struct {
	DBManager *DBManager
	Network   string
	Blob      interface{}
}

func NewBroadcastView(dbManager *DBManager, network string, blob interface{}) *BroadcastView {
	return &BroadcastView{DBManager: dbManager, Network: network, Blob: blob}
}

func (r *BroadcastView) Call(context view.Context) (interface{}, error) {
	span := context.StartSpan("broadcast_view")
	defer span.End()
	sm, err := r.DBManager.GetSessionManager(r.Network)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting session manager for network [%s]", r.Network)
	}
	custodian := sm.CustodianID
	session, err := session2.NewJSON(context, context.Initiator(), view2.GetIdentityProvider(context).Identity(custodian))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get session to custodian [%s]", custodian)
	}

	var blob []byte
	switch b := r.Blob.(type) {
	case driver.Envelope:
		var err error
		blob, err = b.Bytes()
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal envelope")
		}
	default:
		return nil, errors.Errorf("unsupported blob type [%T]", b)
	}

	// TODO: Should we sign the broadcast request?
	request := &BroadcastRequest{
		Network: r.Network,
		Blob:    blob,
	}
	span.AddEvent("send_broadcast_request")
	if err := session.SendWithContext(context.Context(), request); err != nil {
		return nil, errors.Wrapf(err, "failed to send request to custodian [%s]", custodian)
	}
	response := &BroadcastResponse{}
	span.AddEvent("receive_broadcast_response")
	if err := session.ReceiveWithTimeout(response, 30*time.Second); err != nil {
		return nil, errors.Wrapf(err, "failed to receive response from custodian [%s]", custodian)
	}
	if len(response.Err) != 0 {
		return nil, errors.Errorf("failed to broadcast with response err [%s]", response.Err)
	}
	return nil, nil
}

type BroadcastResponderView struct {
	dbManager *DBManager
}

func (r *BroadcastResponderView) Call(context view.Context) (interface{}, error) {
	span := context.StartSpan("broadcast_responder_view")
	defer span.End()
	// receive request
	session := session2.JSON(context)
	request := &BroadcastRequest{}
	span.AddEvent("receive_request")
	if err := session.Receive(request); err != nil {
		return nil, errors.Wrapf(err, "failed to receive request")
	}
	logger.Debugf("request: %+v", request)

	// commit
	sm, err := r.dbManager.GetSessionManager(request.Network)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get session manager for [%s]", request.Network)
	}

	done := false
	err = nil
	txStatusFetcher := &RequestTxStatusResponderView{r.dbManager}
	numRetries := 5
	sleepDuration := 1 * time.Second
	for i := 0; i < numRetries; i++ {
		span.AddEvent("try_broadcast")
		if _, txID, err2 := r.broadcast(context, sm, request); err2 != nil {
			span.RecordError(err2)
			logger.Errorf("failed to broadcast to [%s], txID [%s] with err [%s], retry [%d]", sm.CustodianID, txID, err2, i)
			if strings.Contains(err2.Error(), "is not valid") {
				err = err2
				break
			}
			if len(txID) != 0 {
				// was the transaction committed, by any chance?
				logger.Errorf("check transaction [%s] status on [%s], retry [%d]", txID, sm.CustodianID, i)
				span.AddEvent("fetch_tx_status")
				status, err := txStatusFetcher.process(context, &TxStatusRequest{
					Network: request.Network,
					TxID:    txID,
				})
				if err != nil {
					logger.Errorf("failed to ask transaction status [%s][%s], retry [%d]", txID, err, i)
				}
				if status != nil {
					if status.Status == network.Valid {
						done = true
						break
					}
					if status.Status == network.Invalid {
						break
					}
					logger.Debugf("transaction [%s] status [%d], retry [%d], wait a bit and resubmit", txID, status, i)
				} else {
					logger.Errorf("failed to ask transaction status [%s], got a nil answert, retry [%d]", txID, i)
				}
			}
			time.Sleep(sleepDuration)
			sleepDuration = sleepDuration * 2
			continue
		}
		done = true
		break
	}
	var broadcastError string
	if !done {
		broadcastError = fmt.Sprintf("failed to broadcast to [%s] with err [%s]", sm.CustodianID, err)
	}
	if err := session.SendWithContext(context.Context(), &BroadcastResponse{Err: broadcastError}); err != nil {
		return nil, errors.Wrapf(err, "failed to send response")
	}
	return nil, nil
}

func (r *BroadcastResponderView) broadcast(context view.Context, sm *SessionManager, request *BroadcastRequest) (interface{}, string, error) {
	oSession, err := sm.GetSession()
	if err != nil {
		return nil, "", errors.Wrapf(err, "failed to create session to orion network [%s]", request.Network)
	}
	tm := sm.Orion.TransactionManager()
	env := tm.NewEnvelope()
	if err := env.FromBytes(request.Blob); err != nil {
		return nil, "", errors.Wrap(err, "failed to unmarshal envelope")
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("commit envelope... [%s][%s]", env.TxID(), env.String())
	}
	if err := sm.Orion.TransactionManager().CommitEnvelope(oSession, env); err != nil {
		return nil, env.TxID(), err
	}
	return nil, env.TxID(), nil
}
