/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ttx

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

type TxOptions struct {
	Auditor                   view.Identity
	Network                   string
	Channel                   string
	Namespace                 string
	NoTransactionVerification bool
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
		o.Network = network
		return nil
	}
}

func WithChannel(channel string) TxOption {
	return func(o *TxOptions) error {
		o.Channel = channel
		return nil
	}
}

func WithNamespace(namespace string) TxOption {
	return func(o *TxOptions) error {
		o.Namespace = namespace
		return nil
	}
}

// WithTMS filters by network, channel and namespace. Each of them can be empty
func WithTMS(network, channel, namespace string) TxOption {
	return func(o *TxOptions) error {
		o.Network = network
		o.Channel = channel
		o.Namespace = namespace
		return nil
	}
}

// WithTMSID filters by TMS identifier
func WithTMSID(id token.TMSID) TxOption {
	return func(o *TxOptions) error {
		o.Network = id.Network
		o.Channel = id.Channel
		o.Namespace = id.Namespace
		return nil
	}
}

func WithNoTransactionVerification() TxOption {
	return func(o *TxOptions) error {
		o.NoTransactionVerification = true
		return nil
	}
}
