/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

type TxOptions struct {
	Auditor                   view.Identity
	TMSID                     token.TMSID
	NoTransactionVerification bool
	Timeout                   time.Duration
	TxID                      string
	Transaction               *Transaction
}

func compile(opts ...TxOption) (*TxOptions, error) {
	txOptions := &TxOptions{}
	for _, opt := range opts {
		if err := opt(txOptions); err != nil {
			return nil, err
		}
	}
	return txOptions, nil
}

type TxOption func(*TxOptions) error

func WithAuditor(auditor view.Identity) TxOption {
	return func(o *TxOptions) error {
		o.Auditor = auditor
		return nil
	}
}

func WithNetwork(network string) TxOption {
	return func(o *TxOptions) error {
		o.TMSID.Network = network
		return nil
	}
}

func WithChannel(channel string) TxOption {
	return func(o *TxOptions) error {
		o.TMSID.Channel = channel
		return nil
	}
}

func WithNamespace(namespace string) TxOption {
	return func(o *TxOptions) error {
		o.TMSID.Namespace = namespace
		return nil
	}
}

// WithTMS filters by network, channel and namespace. Each of them can be empty
func WithTMS(network, channel, namespace string) TxOption {
	return func(o *TxOptions) error {
		o.TMSID.Network = network
		o.TMSID.Channel = channel
		o.TMSID.Namespace = namespace
		return nil
	}
}

// WithTMSID filters by TMS identifier
func WithTMSID(id token.TMSID) TxOption {
	return func(o *TxOptions) error {
		o.TMSID = id
		return nil
	}
}

func WithNoTransactionVerification() TxOption {
	return func(o *TxOptions) error {
		o.NoTransactionVerification = true
		return nil
	}
}

func WithTimeout(timeout time.Duration) TxOption {
	return func(o *TxOptions) error {
		o.Timeout = timeout
		return nil
	}
}

func WithTxID(txID string) TxOption {
	return func(o *TxOptions) error {
		o.TxID = txID
		return nil
	}
}

func WithTransactions(tx *Transaction) TxOption {
	return func(o *TxOptions) error {
		o.Transaction = tx
		return nil
	}
}
