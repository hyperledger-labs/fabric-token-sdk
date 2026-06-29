/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashescrow

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type QueryEngine interface {
	UnspentTokensIteratorBy(ctx context.Context, id string, tokenType token2.Type) (driver.UnspentTokensIterator, error)
}

type OwnerWallet struct {
	wallet      *token.OwnerWallet
	queryEngine QueryEngine
}

func (w *OwnerWallet) ListByPreImage(ctx context.Context, preImage []byte, opts ...token.ListTokensOption) (*token2.UnspentTokens, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	tokensByID := map[string]*token2.UnspentToken{}
	for _, sender := range []bool{false, true} {
		tokens, err := w.filter(ctx, compiledOpts.TokenType, sender, (&PreImageSelector{preImage: preImage}).Filter)
		if err != nil {
			return nil, err
		}
		for _, tok := range tokens.Tokens {
			tokensByID[tok.Id.String()] = tok
		}
	}

	tokens := make([]*token2.UnspentToken, 0, len(tokensByID))
	for _, tok := range tokensByID {
		tokens = append(tokens, tok)
	}

	return &token2.UnspentTokens{Tokens: tokens}, nil
}

func (w *OwnerWallet) filter(ctx context.Context, tokenType token2.Type, sender bool, selector SelectFunction) (*token2.UnspentTokens, error) {
	it, err := w.filterIterator(ctx, tokenType, sender, selector)
	if err != nil {
		return nil, errors.Wrap(err, "token selection failed")
	}
	tokens, err := iterators.ReadAllPointers(it)
	if err != nil {
		return nil, err
	}

	return &token2.UnspentTokens{Tokens: tokens}, nil
}

func (w *OwnerWallet) filterIterator(ctx context.Context, tokenType token2.Type, sender bool, selector SelectFunction) (iterators.Iterator[*token2.UnspentToken], error) {
	walletID := recipientWallet(w.wallet)
	if sender {
		walletID = senderWallet(w.wallet)
	}
	it, err := w.queryEngine.UnspentTokensIteratorBy(ctx, walletID, tokenType)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get iterator over unspent tokens")
	}

	return iterators.Filter(it, IsScript(selector)), nil
}

func GetWallet(context view.Context, id string, opts ...token.ServiceOption) *token.OwnerWallet {
	return ttx.GetWallet(context, id, opts...)
}

func Wallet(wallet *token.OwnerWallet) *OwnerWallet {
	if wallet == nil {
		return nil
	}

	return &OwnerWallet{
		wallet:      wallet,
		queryEngine: wallet.TMS().Vault().NewQueryEngine(),
	}
}
