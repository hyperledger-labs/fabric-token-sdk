/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"runtime/debug"
	"strings"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	jsession "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/json/session"
)

type ReceiveTransactionView struct {
	opts []TxOption
}

func NewReceiveTransactionView(opts ...TxOption) *ReceiveTransactionView {
	return &ReceiveTransactionView{
		opts: opts,
	}
}

func (f *ReceiveTransactionView) Call(context view.Context) (interface{}, error) {
	// options
	options, err := CompileOpts(f.opts...)
	if err != nil {
		return nil, errors.Join(err, ErrFailedCompilingOptions)
	}
	if options.Timeout == 0 {
		options.Timeout = time.Minute * 4
	}

	// run
	jsonSession := jsession.JSON(context)
	msg, err := jsonSession.ReceiveRawWithTimeout(options.Timeout)
	if err != nil {
		// TODO: replace this with a check of a typed error
		if strings.Contains(err.Error(), "time out reached") {
			return nil, errors.Join(err, ErrTimeout)
		}
		logger.ErrorfContext(context.Context(), err.Error())
		return nil, err
	}
	logger.DebugfContext(context.Context(), "ReceiveTransactionView: received transaction, len [%d][%s]", len(msg), utils.Hashable(msg))

	if len(msg) == 0 {
		info := context.Session().Info()
		logger.ErrorfContext(context.Context(), "received empty message, session closed [%s:%v]: [%s]", info.ID, info.Closed, string(debug.Stack()))
		return nil, errors.Errorf("received empty message, session closed [%s:%v]", info.ID, info.Closed)
	}
	tx, err := NewTransactionFromBytes(context, msg)
	if err != nil {
		logger.WarnfContext(context.Context(), "failed creating transaction from bytes: [%v], try to unmarshal as signature request...", err)
		// try to unmarshal as SignatureRequest
		var err2 error
		tx, err2 = f.unmarshalAsSignatureRequest(context, msg)
		if err2 != nil {
			return nil, errors.Wrap(errors.Join(err, err2), "failed to receive transaction")
		}
	}
	return tx, nil
}

func (f *ReceiveTransactionView) unmarshalAsSignatureRequest(context view.Context, raw []byte) (*Transaction, error) {
	signatureRequest := &SignatureRequest{}
	err := Unmarshal(raw, signatureRequest)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling signature request, got [%s]", string(raw))
	}
	if len(signatureRequest.TX) == 0 {
		return nil, errors.Wrap(err, "no transaction received")
	}
	tx, err := NewTransactionFromSignatureRequest(context, signatureRequest)
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive transaction")
	}
	return tx, nil
}
