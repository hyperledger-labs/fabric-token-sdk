/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ttxcc

import "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

type txOptions struct {
	auditor   view.Identity
	network   string
	channel   string
	namespace string
}

func compile(opts ...TxOption) (*txOptions, error) {
	txOptions := &txOptions{}
	for _, opt := range opts {
		if err := opt(txOptions); err != nil {
			return nil, err
		}
	}
	return txOptions, nil
}

type TxOption func(*txOptions) error

func WithAuditor(auditor view.Identity) TxOption {
	return func(o *txOptions) error {
		o.auditor = auditor
		return nil
	}
}

func WithNetwork(network string) TxOption {
	return func(o *txOptions) error {
		o.network = network
		return nil
	}
}

func WithChannel(channel string) TxOption {
	return func(o *txOptions) error {
		o.channel = channel
		return nil
	}
}

func WithNamespace(namespace string) TxOption {
	return func(o *txOptions) error {
		o.namespace = namespace
		return nil
	}
}
