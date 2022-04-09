/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nfttx

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

type OwnerWallet struct {
	view2.ServiceProvider
	*token.OwnerWallet
	Precision uint64
}

func (o *OwnerWallet) QueryByKey(state interface{}, key string, value string) error {
	qe, err := NewQueryExecutor(o.ServiceProvider, o.OwnerWallet.ID(), o.Precision)
	if err != nil {
		return errors.WithMessagef(err, "failed to create query executor")
	}
	return qe.QueryByKey(state, key, value)
}

// WithType returns a list token option that filter by the passed token type.
// If the passed token type is the empty string, all token types are selected.
func WithType(tokenType string) token.ListTokensOption {
	return func(o *token.ListTokensOptions) error {
		o.TokenType = tokenType
		return nil
	}
}

// MyWallet returns the default wallet, nil if not found.
func MyWallet(sp view2.ServiceProvider, opts ...token.ServiceOption) *OwnerWallet {
	tms := token.GetManagementService(sp, opts...)
	w := tms.WalletManager().OwnerWallet("")
	if w == nil {
		return nil
	}
	return &OwnerWallet{OwnerWallet: w, ServiceProvider: sp, Precision: tms.PublicParametersManager().Precision()}
}

// MyWalletFromTx returns the default wallet for the tuple (network, channel, namespace) as identified by the passed
// transaction. Returns nil if no wallet is found.
func MyWalletFromTx(sp view2.ServiceProvider, tx *Transaction) *OwnerWallet {
	tms := token.GetManagementService(
		sp,
		token.WithNetwork(tx.Network()),
		token.WithChannel(tx.Channel()),
		token.WithNamespace(tx.Namespace()),
	)
	w := tms.WalletManager().OwnerWallet("")
	if w == nil {
		return nil
	}
	return &OwnerWallet{OwnerWallet: w, ServiceProvider: sp, Precision: tms.PublicParametersManager().Precision()}
}

// GetWallet returns the wallet whose id is the passed id.
// If the passed id is empty, GetWallet has the same behaviour of MyWallet.
// It returns nil, if no wallet is found.
func GetWallet(sp view2.ServiceProvider, id string, opts ...token.ServiceOption) *OwnerWallet {
	tms := token.GetManagementService(sp, opts...)
	w := tms.WalletManager().OwnerWallet(id)
	if w == nil {
		return nil
	}
	return &OwnerWallet{OwnerWallet: w, ServiceProvider: sp, Precision: tms.PublicParametersManager().Precision()}
}

// GetWalletForChannel returns the wallet whose id is the passed id for the passed channel.
// If the passed id is empty, GetWalletForChannel has the same behaviour of MyWalletFromTx.
// It returns nil, if no wallet is found.
func GetWalletForChannel(sp view2.ServiceProvider, channel, id string, opts ...token.ServiceOption) *OwnerWallet {
	tms := token.GetManagementService(sp, append(opts, token.WithChannel(channel))...)
	w := tms.WalletManager().OwnerWallet(id)
	if w == nil {
		return nil
	}
	return &OwnerWallet{OwnerWallet: w, ServiceProvider: sp, Precision: tms.PublicParametersManager().Precision()}
}

// MyIssuerWallet returns the default issuer wallet, nil if not found
func MyIssuerWallet(context view.Context, opts ...token.ServiceOption) *token.IssuerWallet {
	w := token.GetManagementService(context, opts...).WalletManager().IssuerWallet("")
	if w == nil {
		return nil
	}
	return w
}

// GetIssuerWallet returns the issuer wallet whose id is the passed id.
// If the passed id is empty, GetIssuerWallet has the same behaviour of MyIssuerWallet.
// It returns nil, if no wallet is found.
func GetIssuerWallet(sp view2.ServiceProvider, id string, opts ...token.ServiceOption) *token.IssuerWallet {
	w := token.GetManagementService(sp, opts...).WalletManager().IssuerWallet(id)
	if w == nil {
		return nil
	}
	return w
}

// GetIssuerWalletForChannel returns the issuer wallet whose id is the passed id for the passed channel.
// If the passed id is empty, GetIssuerWalletForChannel has the same behaviour of MyIssuerWallet.
// It returns nil, if no wallet is found.
func GetIssuerWalletForChannel(sp view2.ServiceProvider, channel, id string, opts ...token.ServiceOption) *token.IssuerWallet {
	w := token.GetManagementService(sp, append(opts, token.WithChannel(channel))...).WalletManager().IssuerWallet(id)
	if w == nil {
		return nil
	}
	return w
}

// MyAuditorWallet returns the default auditor wallet, nil if not found.
func MyAuditorWallet(sp view2.ServiceProvider, opts ...token.ServiceOption) *token.AuditorWallet {
	w := token.GetManagementService(sp, opts...).WalletManager().AuditorWallet("")
	if w == nil {
		return nil
	}
	return w
}
