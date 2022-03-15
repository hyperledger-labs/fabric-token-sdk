/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nftcc

import (
	"encoding/base64"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/state"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracker/metrics"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/nftcc/marshaller"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxcc"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var logger = flogging.MustGetLogger("token-sdk.nftcc")

type TxOption ttxcc.TxOption

func WithAuditor(auditor view.Identity) TxOption {
	return func(o *ttxcc.TxOptions) error {
		o.Auditor = auditor
		return nil
	}
}

type Transaction struct {
	*ttxcc.Transaction
}

func NewAnonymousTransaction(sp view.Context, opts ...TxOption) (*Transaction, error) {
	// convert opts to ttxcc.TxOption
	txOpts := make([]ttxcc.TxOption, len(opts))
	for i, opt := range opts {
		txOpts[i] = ttxcc.TxOption(opt)
	}
	tx, err := ttxcc.NewAnonymousTransaction(sp, txOpts...)
	if err != nil {
		return nil, err
	}

	return &Transaction{Transaction: tx}, nil
}

func Wrap(tx *ttxcc.Transaction) *Transaction {
	return &Transaction{Transaction: tx}
}

func ReceiveTransaction(context view.Context) (*Transaction, error) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("receive a new transaction...")
	}

	txBoxed, err := context.RunView(ttxcc.NewReceiveTransactionView(""), view.WithSameContext())
	if err != nil {
		return nil, err
	}

	cctx, ok := txBoxed.(*ttxcc.Transaction)
	if !ok {
		return nil, errors.Errorf("received transaction of wrong type [%T]", cctx)
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("received transaction with id [%s]", cctx.ID())
	}

	return &Transaction{Transaction: cctx}, nil
}

func (t *Transaction) Issue(wallet *token.IssuerWallet, state interface{}, recipient view.Identity, opts ...token.IssueOption) error {
	// set state id first
	_, err := t.setStateID(state)
	if err != nil {
		return err
	}
	// marshal state to json
	stateJSON, err := marshaller.Marshal(state)
	if err != nil {
		return errors.Wrap(err, "failed to marshal state")
	}
	stateJSONStr := base64.StdEncoding.EncodeToString(stateJSON)

	// Issue
	return t.Transaction.Issue(wallet, recipient, stateJSONStr, 1, opts...)
}

func (t *Transaction) Transfer(wallet *token.OwnerWallet, state interface{}, recipient view.Identity, opts ...token.TransferOption) error {
	// marshal state to json
	stateJSON, err := marshaller.Marshal(state)
	if err != nil {
		return errors.Wrap(err, "failed to marshal state")
	}
	stateJSONStr := base64.StdEncoding.EncodeToString(stateJSON)

	return t.Transaction.Transfer(wallet, stateJSONStr, []uint64{1}, []view.Identity{recipient}, opts...)
}

func (t *Transaction) Outputs() (*OutputStream, error) {
	os, err := t.Transaction.Outputs()
	if err != nil {
		return nil, err
	}
	return &OutputStream{OutputStream: os}, nil
}

func (t *Transaction) QueryExecutor() (*QueryExecutor, error) {
	qe := t.TokenService().Vault().NewQueryEngine()
	return &QueryExecutor{
		selector: NewFilter(
			qe,
			metrics.Get(t.Transaction.SP),
		),
		vault: qe,
	}, nil
}

func (t *Transaction) setStateID(s interface{}) (string, error) {
	logger.Debugf("setStateID %v...", s)
	defer logger.Debugf("setStateID...done")
	var key string
	var err error
	switch d := s.(type) {
	case AutoLinearState:
		logger.Debugf("AutoLinearState...")
		key, err = d.GetLinearID()
		if err != nil {
			return "", err
		}
	case LinearState:
		logger.Debugf("LinearState...")
		key = state.GenerateUUID()
		key = d.SetLinearID(key)
	default:
		// TODO: should we return an error here?
		return "", nil
	}
	return key, nil
}
