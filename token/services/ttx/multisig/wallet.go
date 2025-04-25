/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	"context"

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
func (w *OwnerWallet) ListTokensAsEscrow(opts ...token.ListTokensOption) (*FilteredIterator, error) {
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

	return w.filter(compiledOpts.TokenType)
}

// ListTokensIterator returns an iterator of tokens that matches the passed options and whose recipient belongs to this wallet
func (w *OwnerWallet) ListTokensIterator(opts ...token.ListTokensOption) (*FilteredIterator, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}
	return w.filterIterator(compiledOpts.TokenType)
}

func (w *OwnerWallet) filter(tokenType token2.Type) (*token2.UnspentTokens, error) {
	it, err := w.filterIterator(tokenType)
	if err != nil {
		return nil, errors.Wrap(err, "token selection failed")
	}
	defer it.Close()
	var tokens []*token2.UnspentToken
	for {
		tok, err := it.Next()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get next unspent token from iterator")
		}
		if tok == nil {
			break
		}
		logger.Debugf("filtered token [%s]", tok.Id)

		tokens = append(tokens, tok)
	}
	return &token2.UnspentTokens{Tokens: tokens}, nil
}

func (w *OwnerWallet) filterIterator(tokenType token2.Type) (*FilteredIterator, error) {
	walletID := escrowWallet(w.wallet)
	it, err := w.queryEngine.UnspentTokensIteratorBy(context.TODO(), walletID, tokenType)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get iterator over unspent tokens")
	}
	return &FilteredIterator{
		it: it,
	}, nil
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

type FilteredIterator struct {
	it driver.UnspentTokensIterator
}

func (f *FilteredIterator) Close() {
	f.it.Close()
}

func (f *FilteredIterator) Next() (*token2.UnspentToken, error) {
	for {
		tok, err := f.it.Next()
		if err != nil {
			return nil, err
		}
		if tok == nil {
			logger.Debugf("no more tokens!")
			return nil, nil
		}
		owner, err := identity.UnmarshalTypedIdentity(tok.Owner)
		if err != nil {
			logger.Debugf("Is Mine [%s,%s,%s]? No, failed unmarshalling [%s]", view.Identity(tok.Owner), tok.Type, tok.Quantity, err)
			continue
		}
		if owner.Type == multisig.Multisig {
			escrow := &multisig.MultiIdentity{}
			if err := escrow.Deserialize(owner.Identity); err != nil {
				logger.Debugf("token [%s,%s,%s,%s] contains an escrow? No", tok.Id, view.Identity(tok.Owner).UniqueID(), tok.Type, tok.Quantity)
				continue
			}

			logger.Debugf("token [%s,%s,%s,%s] contains an escrow? Yes", tok.Id, view.Identity(tok.Owner).UniqueID(), tok.Type, tok.Quantity)
			return tok, nil
		}
	}
}

// Sum  computes the sum of the quantities of the tokens in the iterator.
// Sum closes the iterator at the end of the execution.
func (f *FilteredIterator) Sum(precision uint64) (token2.Quantity, error) {
	defer f.Close()
	sum := token2.NewZeroQuantity(precision)
	var tokens []*token2.UnspentToken
	counter := 0

	for {
		isDuplicate := false
		tok, err := f.Next()
		if err != nil {
			return nil, err
		}
		if tok == nil {
			break
		}
		for _, t := range tokens {
			if t.Id.TxId == tok.Id.TxId {
				isDuplicate = true
				break
			}
		}
		if !isDuplicate {
			q, err := token2.ToQuantity(tok.Quantity, precision)
			if err != nil {
				return nil, err
			}
			sum = sum.Add(q)
			tokens = append(tokens, tok)
		}
		counter++
	}
	return sum, nil
}
