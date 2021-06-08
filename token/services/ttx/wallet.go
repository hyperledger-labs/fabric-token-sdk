/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ttx

import (
	"fmt"

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

// MyWallet returns the default wallet
func MyWallet(sp view2.ServiceProvider) *token.OwnerWallet {
	w := token.GetManagementService(sp).WalletManager().OwnerWallet("")
	if w == nil {
		panic(fmt.Sprint("cannot find default wallet for default channel"))
	}
	return w
}

// MyWalletForChannel returns the default wallet for the passed channel
func MyWalletForChannel(sp view2.ServiceProvider, channel string) *token.OwnerWallet {
	w := token.GetManagementService(sp, token.WithChannel(channel)).WalletManager().OwnerWallet("")
	if w == nil {
		panic(fmt.Sprintf("cannot find default wallet for channel [%s]", channel))
	}
	return w
}

// GetWallet returns the wallet whose id is the passed id.
// If the passed id is empty, GetWallet has the same behaviour of MyWallet.
func GetWallet(sp view2.ServiceProvider, id string) *token.OwnerWallet {
	w := token.GetManagementService(sp).WalletManager().OwnerWallet(id)
	if w == nil {
		panic(fmt.Sprint("cannot find default wallet for default channel"))
	}
	return w
}

// GetWalletForChannel returns the wallet whose id is the passed id for the passed channel.
// If the passed id is empty, GetWalletForChannel has the same behaviour of MyWalletForChannel.
func GetWalletForChannel(sp view2.ServiceProvider, channel, id string) *token.OwnerWallet {
	w := token.GetManagementService(sp, token.WithChannel(channel)).WalletManager().OwnerWallet(id)
	if w == nil {
		panic(fmt.Sprintf("cannot find wallet [%s] for channel [%s]", id, channel))
	}
	return w
}

// MyIssuerWallet returns the default issuer wallet
func MyIssuerWallet(context view.Context) *token.IssuerWallet {
	w := token.GetManagementService(context).WalletManager().IssuerWallet("")
	if w == nil {
		panic(fmt.Sprint("cannot find default wallet for default channel"))
	}
	return w
}

// GetIssuerWallet returns the issuer wallet whose id is the passed id.
// If the passed id is empty, GetIssuerWallet has the same behaviour of MyIssuerWallet.
func GetIssuerWallet(sp view2.ServiceProvider, id string) *token.IssuerWallet {
	w := token.GetManagementService(sp).WalletManager().IssuerWallet(id)
	if w == nil {
		panic(fmt.Sprintf("cannot find wallet [%s] for default channel", id))
	}
	return w
}

// GetIssuerWalletForChannel returns the issuer wallet whose id is the passed id for the passed channel.
// If the passed id is empty, GetIssuerWalletForChannel has the same behaviour of MyIssuerWallet.
func GetIssuerWalletForChannel(sp view2.ServiceProvider, channel, id string) *token.IssuerWallet {
	w := token.GetManagementService(sp, token.WithChannel(channel)).WalletManager().IssuerWallet(id)
	if w == nil {
		panic(fmt.Sprintf("cannot find wallet [%s] for channel [%s]", id, channel))
	}
	return w
}

// MyMyAuditorWalletWallet returns the default auditor wallet
func MyAuditorWallet(sp view2.ServiceProvider) *token.AuditorWallet {
	w := token.GetManagementService(sp).WalletManager().AuditorWallet("")
	if w == nil {
		panic(fmt.Sprint("cannot find default wallet for default channel"))
	}
	return w
}
