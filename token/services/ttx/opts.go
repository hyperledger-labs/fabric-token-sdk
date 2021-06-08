/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ttx

import "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

type txOptions struct {
	auditor view.Identity
	channel string
	cc      string
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

func WithChannel(channel string) TxOption {
	return func(o *txOptions) error {
		o.channel = channel
		return nil
	}
}
