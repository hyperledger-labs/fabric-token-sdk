/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"context"
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
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

// ListTokensAsSender returns a list of non-expired htlc-tokens whose sender id is in this wallet
func (w *OwnerWallet) ListTokensAsSender(opts ...token.ListTokensOption) (*FilteredIterator, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filterIterator(compiledOpts.TokenType, true, SelectNonExpired)
}

// GetExpiredByHash returns the expired htlc-token whose sender id is in this wallet and whose hash is equal to the one passed as argument.
// It fails if no tokens are found or if more than one token is found.
func (w *OwnerWallet) GetExpiredByHash(hash []byte, opts ...token.ListTokensOption) (*token2.UnspentToken, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	tokens, err := w.filter(compiledOpts.TokenType, true, (&ExpiredAndHashSelector{Hash: hash}).Select)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to filter")
	}
	if len(tokens.Tokens) != 1 {
		return nil, errors.Errorf("expected to find only one token for the hash [%v], found [%d]", hash, len(tokens.Tokens))
	}
	return tokens.Tokens[0], nil
}

// ListExpired returns a list of expired htlc-tokens whose sender id is in this wallet
func (w *OwnerWallet) ListExpired(opts ...token.ListTokensOption) (*token2.UnspentTokens, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filter(compiledOpts.TokenType, true, SelectExpired)
}

// ListExpiredIterator returns an iterator of expired htlc-tokens whose sender id is in this wallet
func (w *OwnerWallet) ListExpiredIterator(opts ...token.ListTokensOption) (*FilteredIterator, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filterIterator(compiledOpts.TokenType, true, SelectExpired)
}

// ListByPreImage returns a list of tokens whose recipient is this wallet and with a matching preimage
func (w *OwnerWallet) ListByPreImage(preImage []byte, opts ...token.ListTokensOption) (*token2.UnspentTokens, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filter(compiledOpts.TokenType, false, (&PreImageSelector{preImage: preImage}).Filter)
}

// ListByPreImageIterator returns an iterator of tokens whose recipient is this wallet and with a matching preimage
func (w *OwnerWallet) ListByPreImageIterator(preImage []byte, opts ...token.ListTokensOption) (*FilteredIterator, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filterIterator(compiledOpts.TokenType, false, (&PreImageSelector{preImage: preImage}).Filter)
}

// ListTokens returns a list of tokens that matches the passed options and whose recipient belongs to this wallet
func (w *OwnerWallet) ListTokens(opts ...token.ListTokensOption) (*token2.UnspentTokens, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filter(compiledOpts.TokenType, false, SelectNonExpired)
}

// ListTokensIterator returns an iterator of tokens that matches the passed options and whose recipient belongs to this wallet
func (w *OwnerWallet) ListTokensIterator(opts ...token.ListTokensOption) (*FilteredIterator, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filterIterator(compiledOpts.TokenType, false, SelectNonExpired)
}

// GetExpiredReceivedTokenByHash returns the expired htlc-token that matches the passed options, whose recipient belongs to this wallet, is expired, and hash the same hash.
// It fails if no tokens are found or if more than one token is found.
func (w *OwnerWallet) GetExpiredReceivedTokenByHash(hash []byte, opts ...token.ListTokensOption) (*token2.UnspentToken, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	tokens, err := w.filter(compiledOpts.TokenType, false, (&ExpiredAndHashSelector{Hash: hash}).Select)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to filter")
	}
	if len(tokens.Tokens) != 1 {
		return nil, errors.Errorf("expected to find only one token for the hash [%v], found [%d]", hash, len(tokens.Tokens))
	}
	return tokens.Tokens[0], nil
}

// ListExpiredReceivedTokens returns a list of tokens that matches the passed options, whose recipient belongs to this wallet, and are expired
func (w *OwnerWallet) ListExpiredReceivedTokens(opts ...token.ListTokensOption) (*token2.UnspentTokens, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filter(compiledOpts.TokenType, false, SelectExpired)
}

// ListExpiredReceivedTokensIterator returns an iterator of tokens that matches the passed options, whose recipient belongs to this wallet, and are expired
func (w *OwnerWallet) ListExpiredReceivedTokensIterator(opts ...token.ListTokensOption) (*FilteredIterator, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filterIterator(compiledOpts.TokenType, false, SelectExpired)
}

// DeleteExpiredReceivedTokens removes the expired htlc-tokens that have been reclaimed
func (w *OwnerWallet) DeleteExpiredReceivedTokens(context view.Context, opts ...token.ListTokensOption) error {
	it, err := w.ListExpiredReceivedTokensIterator(opts...)
	if err != nil {
		return errors.WithMessage(err, "failed to get an iterator of expired received tokens")
	}
	defer it.Close()
	var buffer []*token2.UnspentToken
	for {
		tok, err := it.Next()
		if err != nil {
			return errors.WithMessagef(err, "failed to get next expired received token")
		}
		if tok == nil {
			break
		}
		buffer = append(buffer, tok)
		if len(buffer) > w.bufferSize {
			if err := w.deleteTokens(context, buffer); err != nil {
				return errors.WithMessagef(err, "failed to process tokens [%v]", buffer)
			}
			buffer = nil
		}
	}
	if err := w.deleteTokens(context, buffer); err != nil {
		return errors.WithMessagef(err, "failed to process tokens [%v]", buffer)
	}

	return nil
}

// DeleteClaimedSentTokens removes the claimed htlc-tokens whose sender id is in this wallet
func (w *OwnerWallet) DeleteClaimedSentTokens(context view.Context, opts ...token.ListTokensOption) error {
	it, err := w.ListTokensAsSender(opts...)
	if err != nil {
		return errors.WithMessage(err, "failed to get an iterator of expired received tokens")
	}
	defer it.Close()
	var buffer []*token2.UnspentToken
	for {
		tok, err := it.Next()
		if err != nil {
			return errors.WithMessagef(err, "failed to get next expired received token")
		}
		if tok == nil {
			break
		}
		buffer = append(buffer, tok)
		if len(buffer) > w.bufferSize {
			if err := w.deleteTokens(context, buffer); err != nil {
				return errors.WithMessagef(err, "failed to process tokens [%v]", buffer)
			}
			buffer = nil
		}
	}
	if err := w.deleteTokens(context, buffer); err != nil {
		return errors.WithMessagef(err, "failed to process tokens [%v]", buffer)
	}

	return nil
}

func (w *OwnerWallet) deleteTokens(context view.Context, tokens []*token2.UnspentToken) error {
	logger.Debugf("delete tokens from vault [%d][%v]", len(tokens), tokens)
	if len(tokens) == 0 {
		return nil
	}

	// get spent flags
	ids := make([]*token2.ID, len(tokens))
	for i, tok := range tokens {
		ids[i] = tok.Id
	}
	tms := w.wallet.TMS()
	meta, err := tms.WalletManager().SpentIDs(ids)
	if err != nil {
		return errors.WithMessagef(err, "failed to compute spent ids for [%v]", ids)
	}
	net := network.GetInstance(context, tms.Network(), tms.Channel())
	if net == nil {
		return errors.Errorf("cannot load network [%s:%s]", tms.Network(), tms.Channel())
	}
	spent, err := net.AreTokensSpent(context.Context(), tms.Namespace(), ids, meta)
	if err != nil {
		return errors.WithMessagef(err, "cannot fetch spent flags from network [%s:%s] for ids [%v]", tms.Network(), tms.Channel(), ids)
	}

	// remove the tokens flagged as spent
	var toDelete []*token2.ID
	for i, tok := range tokens {
		if spent[i] {
			logger.Debugf("token [%s] is spent", tok.Id)
			toDelete = append(toDelete, tok.Id)
		} else {
			logger.Debugf("token [%s] is not spent", tok.Id)
		}
	}
	if err := w.vault.DeleteTokens(toDelete...); err != nil {
		return errors.WithMessagef(err, "failed to remove token ids [%v]", toDelete)
	}

	return nil
}

func (w *OwnerWallet) filter(tokenType token2.Type, sender bool, selector SelectFunction) (*token2.UnspentTokens, error) {
	it, err := w.filterIterator(tokenType, sender, selector)
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

func (w *OwnerWallet) filterIterator(tokenType token2.Type, sender bool, selector SelectFunction) (*FilteredIterator, error) {
	var walletID string
	if sender {
		walletID = senderWallet(w.wallet)
	} else {
		walletID = recipientWallet(w.wallet)
	}
	it, err := w.queryEngine.UnspentTokensIteratorBy(context.TODO(), walletID, tokenType)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get iterator over unspent tokens")
	}
	return &FilteredIterator{
		it:       it,
		selector: selector,
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
	it       driver.UnspentTokensIterator
	selector SelectFunction
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
		if owner.Type == ScriptType {
			script := &Script{}
			if err := json.Unmarshal(owner.Identity, script); err != nil {
				logger.Debugf("token [%s,%s,%s,%s] contains a script? No", tok.Id, view.Identity(tok.Owner).UniqueID(), tok.Type, tok.Quantity)
				continue
			}
			if script.Sender.IsNone() {
				logger.Debugf("token [%s,%s,%s,%s] contains a script? No", tok.Id, view.Identity(tok.Owner).UniqueID(), tok.Type, tok.Quantity)
				continue
			}
			logger.Debugf("token [%s,%s,%s,%s] contains a script? Yes", tok.Id, view.Identity(tok.Owner).UniqueID(), tok.Type, tok.Quantity)

			pickItem, err := f.selector(tok, script)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to select (token,script)[%v:%v] pair", tok, script)
			}
			if pickItem {
				return tok, nil
			}
		}
	}
}

// Sum  computes the sum of the quantities of the tokens in the iterator.
// Sum closes the iterator at the end of the execution.
func (f *FilteredIterator) Sum(precision uint64) (token2.Quantity, error) {
	defer f.Close()
	sum := token2.NewZeroQuantity(precision)
	for {
		tok, err := f.Next()
		if err != nil {
			return nil, err
		}
		if tok == nil {
			break
		}

		q, err := token2.ToQuantity(tok.Quantity, precision)
		if err != nil {
			return nil, err
		}
		sum = sum.Add(q)
	}

	return sum, nil
}
