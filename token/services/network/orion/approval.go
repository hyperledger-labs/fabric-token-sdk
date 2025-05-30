/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	session2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/json/session"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TxStatusResponseCache interface {
	Get(key string) (*TxStatusResponse, bool)
	GetOrLoad(key string, loader func() (*TxStatusResponse, error)) (*TxStatusResponse, bool, error)
	Add(key string, value *TxStatusResponse)
}

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
	logger.DebugfContext(context.Context(), "Send approval request")
	if err := session.SendWithContext(context.Context(), request); err != nil {
		return nil, errors.Wrapf(err, "failed to send request to custodian [%s]", sm.CustodianID)
	}
	response := &ApprovalResponse{}
	logger.DebugfContext(context.Context(), "Receive approval response")
	if err := session.ReceiveWithTimeout(response, 30*time.Second); err != nil {
		logger.ErrorfContext(context.Context(), "failed to receive response from custodian [%s]", sm.CustodianID)
		return nil, errors.Wrapf(err, "failed to receive response from custodian [%s]", sm.CustodianID)
	}
	logger.DebugfContext(context.Context(), "Read tx envelope")
	env := sm.Orion.TransactionManager().NewEnvelope()
	if err := env.FromBytes(response.Envelope); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal transaction")
	}
	return env, nil
}

type RequestApprovalResponderView struct {
	dbManager     *DBManager
	statusCache   TxStatusResponseCache
	keyTranslator translator.KeyTranslator
}

func (r *RequestApprovalResponderView) Call(context view.Context) (interface{}, error) {
	// receive request
	session := session2.JSON(context)
	logger.DebugfContext(context.Context(), "Receive approval request")
	request := &ApprovalRequest{}
	if err := session.Receive(request); err != nil {
		return nil, errors.Wrapf(err, "failed to receive request")
	}
	logger.Debugf("request: %+v", request)

	txRaw, err := r.process(context, request)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to process request")
	}
	logger.DebugfContext(context.Context(), "Send approval response")
	if err := session.SendWithContext(context.Context(), &ApprovalResponse{Envelope: txRaw}); err != nil {
		return nil, errors.Wrapf(err, "failed to send response")
	}
	return nil, nil
}

func (r *RequestApprovalResponderView) process(context view.Context, request *ApprovalRequest) ([]byte, error) {
	tms := token.GetManagementService(context, token.WithNetwork(request.Network), token.WithNamespace(request.Namespace))
	if tms == nil {
		return nil, errors.Errorf("failed to get token management service for network [%s:%s]", request.Network, request.Namespace)
	}
	validator, err := tms.Validator()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create validator")
	}

	// commit
	txStatusFetcher := &RequestTxStatusResponderView{
		dbManager:     r.dbManager,
		statusCache:   r.statusCache,
		keyTranslator: r.keyTranslator,
	}

	runner := common2.NewRetryRunner(5, time.Second, true)

	var envelopeRaw []byte
	validateErr := runner.RunWithErrors(func() (bool, error) {
		logger.DebugfContext(context.Context(), "Try validate")
		var retry bool
		envelopeRaw, retry, err = r.validate(context, request, validator)
		if err == nil {
			return true, nil
		}

		if !retry {
			logger.Errorf("failed to commit transaction [%s], no more retry after this", err)
			return true, errors.Wrapf(err, "failed to commit transaction [%s]", request.TxID)
		}
		logger.Errorf("failed to commit transaction [%s], retry", err)
		// was the transaction committed, by any chance?
		logger.DebugfContext(context.Context(), "Fetc tx status")
		status, err := txStatusFetcher.process(&TxStatusRequest{
			Network:   request.Network,
			Namespace: request.Namespace,
			TxID:      request.TxID,
		})
		if err != nil {
			logger.Errorf("failed to ask transaction status [%s], retry", err)
			return false, nil
		}

		if status.Status == network.Valid {
			return true, nil
		}
		if status.Status == network.Invalid {
			return true, errors.New("invalid transaction status")
		}
		logger.Debugf("transaction [%s] status [%d], retry, wait a bit and resubmit", request.TxID, status)
		return false, nil
	})

	if validateErr != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction [%s]", request.TxID)
	}
	return envelopeRaw, nil
}

func (r *RequestApprovalResponderView) validate(context view.Context, request *ApprovalRequest, validator *token.Validator) ([]byte, bool, error) {
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
	actions, attributes, err := validator.UnmarshallAndVerifyWithMetadata(
		context.Context(),
		&LedgerWrapper{qe: qe, keyTranslator: &translator.HashedKeyTranslator{KT: &keys.Translator{}}},
		request.TxID,
		request.Request,
	)
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to unmarshall and verify request")
	}

	// Write
	tx, err := sm.Orion.TransactionManager().NewTransactionFromSession(oSession, request.TxID)
	if err != nil {
		return nil, true, errors.Wrapf(err, "failed to create transaction [%s]", request.TxID)
	}
	rws := &TxRWSWrapper{
		me: sm.CustodianID,
		db: request.Namespace,
		tx: tx,
	}
	t := translator.New(request.TxID, translator.NewRWSetWrapper(rws, "", request.TxID), &translator.HashedKeyTranslator{KT: &keys.Translator{}})
	for _, action := range actions {
		err = t.Write(action)
		if err != nil {
			return nil, false, errors.Wrapf(err, "failed to write action")
		}
	}
	logger.DebugfContext(context.Context(), "Commit token request")
	h, err := t.CommitTokenRequest(attributes[common.TokenRequestToSign], true)
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to commit token request")
	}

	// close transaction
	envelopeRaw, err := tx.SignAndClose()
	if err != nil {
		return nil, true, errors.Wrapf(err, "failed to sign and close transaction [%s]", request.TxID)
	}
	// update the cache
	r.statusCache.Add(request.TxID, &TxStatusResponse{
		Status:                driver2.Busy,
		TokenRequestReference: h,
	})
	return envelopeRaw, false, nil
}

type LedgerWrapper struct {
	qe            *orion.SessionQueryExecutor
	keyTranslator translator.KeyTranslator
}

func (l *LedgerWrapper) GetState(id token2.ID) ([]byte, error) {
	key, err := l.keyTranslator.CreateOutputKey(id.TxId, id.Index)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting token key for [%v]", id)
	}
	return l.qe.Get(orionKey(key))
}
