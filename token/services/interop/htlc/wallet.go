/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"encoding/json"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type QueryEngine interface {
	// UnspentTokensIteratorBy returns an iterator over all unspent tokens by type and id. Type can be empty
	UnspentTokensIteratorBy(id, typ string) (driver.UnspentTokensIterator, error)
}

// OwnerWallet is a combination of a wallet and a query service
type OwnerWallet struct {
	wallet       *token.OwnerWallet
	queryService QueryEngine
	vault        *vault.Vault
	bufferSize   int
}

// ListExpired returns a list of expired htlc-tokens whose sender id is in this wallet
func (w *OwnerWallet) ListExpired(opts ...token.ListTokensOption) (*token2.UnspentTokens, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filter(compiledOpts.TokenType, true, SelectExpired)
}

// ListExpiredIterator returns a iterator of expired htlc-tokens whose sender id is in this wallet
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
			if err := w.deleteExpiredReceivedTokens(context, buffer); err != nil {
				return errors.WithMessagef(err, "failed to process tokens [%v]", buffer)
			}
			buffer = nil
		}
	}
	if err := w.deleteExpiredReceivedTokens(context, buffer); err != nil {
		return errors.WithMessagef(err, "failed to process tokens [%v]", buffer)
	}

	return nil
}

func (w *OwnerWallet) deleteExpiredReceivedTokens(context view.Context, expiredTokens []*token2.UnspentToken) error {
	if len(expiredTokens) == 0 {
		return nil
	}

	// get spent flags
	ids := make([]*token2.ID, len(expiredTokens))
	for i, tok := range expiredTokens {
		ids[i] = tok.Id
	}
	tms := w.wallet.TMS()
	spentIDs, err := tms.WalletManager().SpentIDs(ids)
	if err != nil {
		return errors.WithMessagef(err, "failed to compute spent ids for [%v]", ids)
	}
	net := network.GetInstance(context, tms.Network(), tms.Channel())
	if net == nil {
		return errors.Errorf("cannot load network [%s:%s]", tms.Network(), tms.Channel())
	}
	spent, err := net.AreTokensSpent(context, tms.Namespace(), spentIDs)
	if err != nil {
		return errors.WithMessagef(err, "cannot fetch spent flags from network [%s:%s] for ids [%v]", tms.Network(), tms.Channel(), ids)
	}

	// remove the tokens flagged as spent
	var toDelete []*token2.ID
	for i, unspentToken := range expiredTokens {
		if spent[i] {
			logger.Debugf("token [%s] is spent", unspentToken.Id)
			toDelete = append(toDelete, unspentToken.Id)
		} else {
			logger.Debugf("token [%s] is not spent", unspentToken.Id)
		}
	}
	if err := w.vault.DeleteTokens(tms.Namespace(), toDelete...); err != nil {
		return errors.WithMessagef(err, "failed to remove token ids [%v]", toDelete)
	}

	return nil
}

func (w *OwnerWallet) filter(tokenType string, sender bool, selector SelectFunction) (*token2.UnspentTokens, error) {
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
		tokens = append(tokens, tok)
	}
	return &token2.UnspentTokens{Tokens: tokens}, nil
}

func (w *OwnerWallet) filterIterator(tokenType string, sender bool, selector SelectFunction) (*FilteredIterator, error) {
	var walletID string
	if sender {
		walletID = senderWallet(w.wallet)
	} else {
		walletID = recipientWallet(w.wallet)
	}
	it, err := w.queryService.UnspentTokensIteratorBy(walletID, tokenType)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get iterator over unspent tokens")
	}
	return &FilteredIterator{
		it:       it,
		selector: selector,
	}, nil
}

// GetWallet returns the wallet whose id is the passed id
func GetWallet(sp view2.ServiceProvider, id string, opts ...token.ServiceOption) *token.OwnerWallet {
	return ttx.GetWallet(sp, id, opts...)
}

// Wallet returns an OwnerWallet which contains a wallet and a query service
func Wallet(sp view2.ServiceProvider, wallet *token.OwnerWallet, opts ...token.ServiceOption) *OwnerWallet {
	tms := token.GetManagementService(sp, opts...)
	nw := network.GetInstance(sp, tms.Network(), tms.Channel())
	if nw == nil {
		return nil
	}
	vault, err := nw.Vault(tms.Namespace())
	if err != nil {
		logger.Errorf("failed to get vault for [%s:%s:%s]", tms.Network(), tms.Channel(), tms.Namespace())
		return nil
	}

	return &OwnerWallet{
		wallet:       wallet,
		vault:        vault.TokenVault(),
		queryService: vault.TokenVault().QueryEngine(),
		bufferSize:   100,
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
			return nil, nil
		}
		owner, err := identity.UnmarshallRawOwner(tok.Owner.Raw)
		logger.Debugf("Is Mine [%s,%s,%s]? No, failed unmarshalling [%s]", view.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, err)
		if owner.Type == ScriptType {
			script := &Script{}
			if err := json.Unmarshal(owner.Identity, script); err != nil {
				logger.Debugf("token [%s,%s,%s,%s] contains a script? No", tok.Id, view.Identity(tok.Owner.Raw).UniqueID(), tok.Type, tok.Quantity)
				continue
			}
			if script.Sender.IsNone() {
				logger.Debugf("token [%s,%s,%s,%s] contains a script? No", tok.Id, view.Identity(tok.Owner.Raw).UniqueID(), tok.Type, tok.Quantity)
				continue
			}
			logger.Debugf("token [%s,%s,%s,%s] contains a script? Yes", tok.Id, view.Identity(tok.Owner.Raw).UniqueID(), tok.Type, tok.Quantity)

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
