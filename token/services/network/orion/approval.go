/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	session2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/pkg/errors"
)

type ApprovalRequest struct {
	Network   string
	Namespace string
	TxID      string
	Request   []byte
}

type ApprovalResponse struct {
	Envelope []byte
}

type RequestApprovalView struct {
	DBManager  *DBManager
	Network    string
	Namespace  string
	RequestRaw []byte
	Signer     view.Identity
	TxID       string
}

func NewRequestApprovalView(
	dbManager *DBManager,
	network string,
	namespace string,
	requestRaw []byte,
	signer view.Identity,
	txID string,
) *RequestApprovalView {
	return &RequestApprovalView{
		DBManager:  dbManager,
		Network:    network,
		Namespace:  namespace,
		RequestRaw: requestRaw,
		Signer:     signer,
		TxID:       txID,
	}
}

func (r *RequestApprovalView) Call(context view.Context) (interface{}, error) {
	sm, err := r.DBManager.GetSessionManager(r.Network)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting session manager for network [%s]", r.Network)
	}
	session, err := session2.NewJSON(context, context.Initiator(), view2.GetIdentityProvider(context).Identity(sm.CustodianID))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get session to custodian [%s]", sm.CustodianID)
	}
	// TODO: Should we sign the approval request?
	request := &ApprovalRequest{
		Network:   r.Network,
		Namespace: r.Namespace,
		TxID:      r.TxID,
		Request:   r.RequestRaw,
	}
	if err := session.Send(request); err != nil {
		return nil, errors.Wrapf(err, "failed to send request to custodian [%s]", sm.CustodianID)
	}
	response := &ApprovalResponse{}
	if err := session.ReceiveWithTimeout(response, 30*time.Second); err != nil {
		return nil, errors.Wrapf(err, "failed to receive response from custodian [%s]", sm.CustodianID)
	}
	env := sm.Orion.TransactionManager().NewEnvelope()
	if err := env.FromBytes(response.Envelope); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal transaction")
	}
	return env, nil
}

type RequestApprovalResponderView struct {
	dbManager *DBManager
}

func (r *RequestApprovalResponderView) Call(context view.Context) (interface{}, error) {
	// receive request
	session := session2.JSON(context)
	request := &ApprovalRequest{}
	if err := session.Receive(request); err != nil {
		return nil, errors.Wrapf(err, "failed to receive request")
	}
	logger.Debugf("request: %+v", request)

	txRaw, err := r.process(context, request)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to process request")
	}
	if err := session.Send(&ApprovalResponse{Envelope: txRaw}); err != nil {
		return nil, errors.Wrapf(err, "failed to send response")
	}
	return nil, nil
}

func (r *RequestApprovalResponderView) process(context view.Context, request *ApprovalRequest) ([]byte, error) {
	ds, err := driver.GetTokenDriverService(context)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get token driver")
	}
	sm, err := r.dbManager.GetSessionManager(request.Network)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get session manager for network [%s]", request.Network)
	}
	pp, err := sm.PublicParameters(ds, request.Namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get public parameters for network [%s]", request.Network)
	}
	validator, err := ds.NewValidator(token.TMSID{Network: request.Network, Namespace: request.Namespace}, pp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create validator")
	}

	// commit
	txStatusFetcher := &RequestTxStatusResponderView{r.dbManager}
	numRetries := 5
	sleepDuration := 1 * time.Second
	for i := 0; i < numRetries; i++ {
		envelopeRaw, retry, err := r.validate(context, request, validator)
		if err != nil {
			if !retry {
				logger.Errorf("failed to commit transaction [%s], no more retry after this [%d]", err, i)
				return nil, errors.Wrapf(err, "failed to commit transaction [%s]", request.TxID)
			}
			logger.Errorf("failed to commit transaction [%s], retry [%d]", err, i)
			// was the transaction committed, by any chance?
			status, err := txStatusFetcher.process(context, &TxStatusRequest{
				Network:   request.Network,
				Namespace: request.Namespace,
				TxID:      request.TxID,
			})
			if err != nil {
				logger.Errorf("failed to ask transaction status [%s], retry [%d]", err, i)
			}
			if status != nil {
				if status.Status == network.Valid {
					return nil, nil
				}
				if status.Status == network.Invalid {
					break
				}
				logger.Debugf("transaction [%s] status [%s], retry [%d], wait a bit and resubmit", request.TxID, status, i)
			} else {
				logger.Errorf("failed to ask transaction status [%s], got a nil answert, retry [%d]", request.TxID, i)
			}
			time.Sleep(sleepDuration)
			sleepDuration = sleepDuration * 2
			continue
		}
		return envelopeRaw, nil
	}

	return nil, errors.Wrapf(err, "failed to commit transaction [%s]", request.TxID)
}

func (r *RequestApprovalResponderView) validate(context view.Context, request *ApprovalRequest, validator driver.Validator) ([]byte, bool, error) {
	sm, err := r.dbManager.GetSessionManager(request.Network)
	if err != nil {
		return nil, true, errors.Wrapf(err, "failed to get session manager for network [%s]", request.Network)
	}
	oSession, err := sm.GetSession()
	if err != nil {
		return nil, true, errors.Wrapf(err, "failed to create session to orion network [%s]", request.Network)
	}
	qe, err := oSession.QueryExecutor(request.Namespace)
	if err != nil {
		return nil, true, errors.Wrapf(err, "failed to get query executor for orion network [%s]", request.Network)
	}
	actions, attributes, err := token.NewValidator(validator).UnmarshallAndVerifyWithMetadata(
		context.Context(),
		&LedgerWrapper{qe: qe},
		request.TxID,
		request.Request,
	)
	if err != nil {
		return nil, true, errors.Wrapf(err, "failed to unmarshall and verify request")
	}

	// Write
	tx, err := sm.Orion.TransactionManager().NewTransaction(request.TxID, sm.CustodianID)
	if err != nil {
		return nil, true, errors.Wrapf(err, "failed to create transaction [%s]", request.TxID)
	}
	rws := &TxRWSWrapper{
		me: sm.CustodianID,
		db: request.Namespace,
		tx: tx,
	}
	t := translator.New(request.TxID, rws, "")
	for _, action := range actions {
		err = t.Write(action)
		if err != nil {
			return nil, false, errors.Wrapf(err, "failed to write action")
		}
	}
	err = t.CommitTokenRequest(attributes[common.TokenRequestToSign], true)
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to commit token request")
	}

	// close transaction
	envelopeRaw, err := tx.SignAndClose()
	if err != nil {
		return nil, true, errors.Wrapf(err, "failed to sign and close transaction [%s]", request.TxID)
	}
	return envelopeRaw, false, nil
}

type LedgerWrapper struct {
	qe *orion.SessionQueryExecutor
}

func (l *LedgerWrapper) GetState(key string) ([]byte, error) {
	return l.qe.Get(orionKey(key))
}
