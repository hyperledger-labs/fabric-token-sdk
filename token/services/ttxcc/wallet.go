/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ttxcc

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

// WithType returns a list token option that filter by the passed token type.
// If the passed token type is the empty string, all token types are selected.
func WithType(tokenType string) token.ListTokensOption {
	return func(o *token.ListTokensOptions) error {
		o.TokenType = tokenType
		return nil
	}
}

// MyWallet returns the default wallet, nil if not found.
func MyWallet(sp view2.ServiceProvider) *token.OwnerWallet {
	w := token.GetManagementService(sp).WalletManager().OwnerWallet("")
	if w == nil {
		return nil
	}
	return w
}

// MyWalletForChannel returns the default wallet for the passed channel, nil if not found.
func MyWalletForChannel(sp view2.ServiceProvider, channel string) *token.OwnerWallet {
	w := token.GetManagementService(sp, token.WithChannel(channel)).WalletManager().OwnerWallet("")
	if w == nil {
		return nil
	}
	return w
}

// GetWallet returns the wallet whose id is the passed id.
// If the passed id is empty, GetWallet has the same behaviour of MyWallet.
// It returns nil, if no wallet is found.
func GetWallet(sp view2.ServiceProvider, id string) *token.OwnerWallet {
	w := token.GetManagementService(sp).WalletManager().OwnerWallet(id)
	if w == nil {
		return nil
	}
	return w
}

// GetWalletForChannel returns the wallet whose id is the passed id for the passed channel.
// If the passed id is empty, GetWalletForChannel has the same behaviour of MyWalletForChannel.
// It returns nil, if no wallet is found.
func GetWalletForChannel(sp view2.ServiceProvider, channel, id string) *token.OwnerWallet {
	w := token.GetManagementService(sp, token.WithChannel(channel)).WalletManager().OwnerWallet(id)
	if w == nil {
		return nil
	}
	return w
}

// MyIssuerWallet returns the default issuer wallet, nil if not found
func MyIssuerWallet(context view.Context) *token.IssuerWallet {
	w := token.GetManagementService(context).WalletManager().IssuerWallet("")
	if w == nil {
		return nil
	}
	return w
}

// GetIssuerWallet returns the issuer wallet whose id is the passed id.
// If the passed id is empty, GetIssuerWallet has the same behaviour of MyIssuerWallet.
// It returns nil, if no wallet is found.
func GetIssuerWallet(sp view2.ServiceProvider, id string) *token.IssuerWallet {
	w := token.GetManagementService(sp).WalletManager().IssuerWallet(id)
	if w == nil {
		return nil
	}
	return w
}

// GetIssuerWalletForChannel returns the issuer wallet whose id is the passed id for the passed channel.
// If the passed id is empty, GetIssuerWalletForChannel has the same behaviour of MyIssuerWallet.
// It returns nil, if no wallet is found.
func GetIssuerWalletForChannel(sp view2.ServiceProvider, channel, id string) *token.IssuerWallet {
	w := token.GetManagementService(sp, token.WithChannel(channel)).WalletManager().IssuerWallet(id)
	if w == nil {
		return nil
	}
	return w
}

// MyAuditorWallet returns the default auditor wallet, nil if not found.
func MyAuditorWallet(sp view2.ServiceProvider) *token.AuditorWallet {
	w := token.GetManagementService(sp).WalletManager().AuditorWallet("")
	if w == nil {
		return nil
	}
	return w
}
