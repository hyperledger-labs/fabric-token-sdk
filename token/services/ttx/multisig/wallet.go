/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/multisig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type Vault interface {
	DeleteTokens(toDelete ...*token2.ID) error
}

type QueryEngine interface {
	// UnspentTokensIteratorBy returns an iterator over all unspent tokens by type and id. Type can be empty
	UnspentTokensIteratorBy(ctx context.Context, id string, tokenType token2.Type) (driver.UnspentTokensIterator, error)
}

type TokenVault interface {
	DeleteTokens(toDelete ...*token2.ID) error
}

// OwnerWallet is a combination of a wallet and a query service
type OwnerWallet struct {
	wallet      *token.OwnerWallet
	queryEngine QueryEngine
	vault       TokenVault
	bufferSize  int
}

// ListTokensAsEscrow returns a list of tokens which are co-owned by OwnerWallet
func (w *OwnerWallet) ListTokensAsEscrow(opts ...token.ListTokensOption) (iterators.Iterator[*token2.UnspentToken], error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filterIterator(compiledOpts.TokenType)
}

// ListTokens returns a list of tokens that matches the passed options and whose recipient belongs to this wallet
func (w *OwnerWallet) ListTokens(opts ...token.ListTokensOption) (*token2.UnspentTokens, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	it, err := w.filterIterator(compiledOpts.TokenType)
	if err != nil {
		return nil, errors.Wrap(err, "token selection failed")
	}
	tokens, err := iterators.ReadAllPointers(it)
	if err != nil {
		return nil, err
	}
	return &token2.UnspentTokens{Tokens: tokens}, nil
}

// ListTokensIterator returns an iterator of tokens that matches the passed options and whose recipient belongs to this wallet
func (w *OwnerWallet) ListTokensIterator(opts ...token.ListTokensOption) (iterators.Iterator[*token2.UnspentToken], error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}
	return w.filterIterator(compiledOpts.TokenType)
}

func (w *OwnerWallet) filterIterator(tokenType token2.Type) (iterators.Iterator[*token2.UnspentToken], error) {
	walletID := escrowWallet(w.wallet)
	it, err := w.queryEngine.UnspentTokensIteratorBy(context.TODO(), walletID, tokenType)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get iterator over unspent tokens")
	}
	return iterators.Filter(it, containsEscrow), nil
}

// GetWallet returns the wallet whose id is the passed id
func GetWallet(sp token.ServiceProvider, id string, opts ...token.ServiceOption) *token.OwnerWallet {
	return ttx.GetWallet(sp, id, opts...)
}

// Wallet returns an OwnerWallet which contains a wallet and a query service
func Wallet(sp token.ServiceProvider, wallet *token.OwnerWallet) *OwnerWallet {
	if wallet == nil {
		return nil
	}

	tms := wallet.TMS()
	tokens, err := tokens.GetService(sp, tms.ID())
	if err != nil {
		return nil
	}
	return &OwnerWallet{
		wallet:      wallet,
		vault:       tokens,
		queryEngine: tms.Vault().NewQueryEngine(),
		bufferSize:  100,
	}
}

func containsEscrow(tok *token2.UnspentToken) bool {
	owner, err := identity.UnmarshalTypedIdentity(tok.Owner)
	if err != nil {
		logger.Debugf("Is Mine [%s,%s,%s]? No, failed unmarshalling [%s]", view.Identity(tok.Owner), tok.Type, tok.Quantity, err)
		return false
	}
	if owner.Type != multisig.Multisig {
		return false
	}

	if err := (&multisig.MultiIdentity{}).Deserialize(owner.Identity); err != nil {
		logger.Debugf("token [%s,%s,%s,%s] contains an escrow? No", tok.Id, view.Identity(tok.Owner).UniqueID(), tok.Type, tok.Quantity)
		return false
	}

	logger.Debugf("token [%s,%s,%s,%s] contains an escrow? Yes", tok.Id, view.Identity(tok.Owner).UniqueID(), tok.Type, tok.Quantity)
	return true
}
