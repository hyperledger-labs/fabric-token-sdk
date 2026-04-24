/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package boolpolicy

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/boolpolicy"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// QueryEngine knows how to iterate over unspent tokens.
type QueryEngine interface {
	UnspentTokensIteratorBy(ctx context.Context, id string, tokenType token2.Type) (driver.UnspentTokensIterator, error)
}

// TokenVault supports token deletion.
type TokenVault interface {
	DeleteTokens(ctx context.Context, toDelete ...*token2.ID) error
}

// OwnerWallet is a wallet wrapper that exposes policy-identity-filtered token lists.
type OwnerWallet struct {
	wallet      *token.OwnerWallet
	queryEngine QueryEngine
	vault       TokenVault
}

// ListTokens returns unspent tokens whose owner is a policy identity and whose
// component identities include at least one belonging to this wallet.
func (w *OwnerWallet) ListTokens(ctx context.Context, opts ...token.ListTokensOption) (*token2.UnspentTokens, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}
	it, err := w.filterIterator(ctx, compiledOpts.TokenType)
	if err != nil {
		return nil, errors.Wrap(err, "token selection failed")
	}
	toks, err := iterators.ReadAllPointers(it)
	if err != nil {
		return nil, err
	}

	return &token2.UnspentTokens{Tokens: toks}, nil
}

// ListTokensIterator is the iterator variant of ListTokens.
func (w *OwnerWallet) ListTokensIterator(ctx context.Context, opts ...token.ListTokensOption) (iterators.Iterator[*token2.UnspentToken], error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filterIterator(ctx, compiledOpts.TokenType)
}

func (w *OwnerWallet) filterIterator(ctx context.Context, tokenType token2.Type) (iterators.Iterator[*token2.UnspentToken], error) {
	walletID := policyWallet(w.wallet)
	it, err := w.queryEngine.UnspentTokensIteratorBy(ctx, walletID, tokenType)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get iterator over unspent tokens")
	}

	return iterators.Filter(it, containsPolicy), nil
}

// GetWallet returns the owner wallet for the given id.
func GetWallet(context view.Context, id string, opts ...token.ServiceOption) *token.OwnerWallet {
	return ttx.GetWallet(context, id, opts...)
}

// Wallet returns an OwnerWallet wrapping the given wallet with policy-token filtering.
func Wallet(context view.Context, wallet *token.OwnerWallet) *OwnerWallet {
	if wallet == nil {
		return nil
	}
	tms := wallet.TMS()
	svc, err := tokens.GetService(context, tms.ID())
	if err != nil {
		return nil
	}

	return &OwnerWallet{
		wallet:      wallet,
		vault:       svc,
		queryEngine: tms.Vault().NewQueryEngine(),
	}
}

// containsPolicy returns true when the token's owner is a valid policy identity.
func containsPolicy(tok *token2.UnspentToken) bool {
	owner, err := identity.UnmarshalTypedIdentity(tok.Owner)
	if err != nil {
		return false
	}
	if owner.Type != boolpolicy.Policy {
		return false
	}

	return (&boolpolicy.PolicyIdentity{}).Deserialize(owner.Identity) == nil
}
