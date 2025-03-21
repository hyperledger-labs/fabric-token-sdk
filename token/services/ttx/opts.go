/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
)

type TxOptions struct {
	Auditor                   view.Identity
	TMSID                     token.TMSID
	NoTransactionVerification bool
	Timeout                   time.Duration
	TxID                      string
	Transaction               *Transaction
	NetworkTxID               network.TxID
	NoCachingRequest          bool
	AnonymousTransaction      bool
}

func CompileOpts(opts ...TxOption) (*TxOptions, error) {
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

// WithNoCachingRequest is used to tell the ordering view to not cache the token request
func WithNoCachingRequest() TxOption {
	return func(o *TxOptions) error {
		o.NoCachingRequest = true
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

// WithTMSIDPointer filters by TMS identifier, if passed
func WithTMSIDPointer(id *token.TMSID) TxOption {
	return func(o *TxOptions) error {
		if id == nil {
			return nil
		}
		o.TMSID = *id
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

func WithNetworkTxID(id network.TxID) TxOption {
	return func(o *TxOptions) error {
		o.NetworkTxID = id
		return nil
	}
}

// WithAnonymousTransaction is used to tell if the transaction needs to be anonymous or not
func WithAnonymousTransaction(v bool) TxOption {
	return func(o *TxOptions) error {
		o.AnonymousTransaction = v
		return nil
	}
}

// CompileServiceOptions compiles the service options
func CompileServiceOptions(opts ...token.ServiceOption) (*token.ServiceOptions, error) {
	txOptions := &token.ServiceOptions{}
	for _, opt := range opts {
		if err := opt(txOptions); err != nil {
			return nil, err
		}
	}
	return txOptions, nil
}

// WithRecipientData is used to add a RecipientData to the service options
func WithRecipientData(recipientData *RecipientData) token.ServiceOption {
	return func(options *token.ServiceOptions) error {
		if options.Params == nil {
			options.Params = map[string]interface{}{}
		}
		options.Params["RecipientData"] = recipientData
		return nil
	}
}

// WithRecipientWalletID is used to add a recipient wallet id to the service options
func WithRecipientWalletID(walletID string) token.ServiceOption {
	return func(options *token.ServiceOptions) error {
		if len(walletID) == 0 {
			return nil
		}
		if options.Params == nil {
			options.Params = map[string]interface{}{}
		}
		options.Params["RecipientWalletID"] = walletID
		return nil
	}
}

func getRecipientWalletID(opts *token.ServiceOptions) string {
	wBoxed, ok := opts.Params["RecipientWalletID"]
	if !ok {
		return ""
	}
	return wBoxed.(string)
}
