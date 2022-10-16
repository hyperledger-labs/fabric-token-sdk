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
}

// ListExpired returns a list of tokens with a passed deadline whose sender id is contained within the wallet
func (w *OwnerWallet) ListExpired(opts ...token.ListTokensOption) (*token2.UnspentTokens, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filter(compiledOpts.TokenType, true, DeadlineBefore)
}

// ListExpiredIterator returns an iterator of tokens with a passed deadline whose sender id is contained within the wallet
func (w *OwnerWallet) ListExpiredIterator(opts ...token.ListTokensOption) (*FilteredIterator, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filterIterator(compiledOpts.TokenType, true, DeadlineBefore)
}

// ListByPreImage returns a list of tokens with a matching preimage
func (w *OwnerWallet) ListByPreImage(preImage []byte, opts ...token.ListTokensOption) (*token2.UnspentTokens, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filter(compiledOpts.TokenType, false, (&PreImageFilter{preImage: preImage}).Filter)
}

// ListByPreImageIterator returns an iterator of tokens with a matching preimage
func (w *OwnerWallet) ListByPreImageIterator(preImage []byte, opts ...token.ListTokensOption) (*FilteredIterator, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filterIterator(compiledOpts.TokenType, false, (&PreImageFilter{preImage: preImage}).Filter)
}

// ListTokens returns a list of tokens that matches the passed options and whose recipient belongs to this wallet
func (w *OwnerWallet) ListTokens(opts ...token.ListTokensOption) (*token2.UnspentTokens, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filter(compiledOpts.TokenType, false, DeadlineAfter)
}

// ListTokensIterator returns an iterator of tokens that matches the passed options and whose recipient belongs to this wallet
func (w *OwnerWallet) ListTokensIterator(opts ...token.ListTokensOption) (*FilteredIterator, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filterIterator(compiledOpts.TokenType, false, DeadlineAfter)
}

// ListExpiredReceivedTokens returns a list of tokens that matches the passed options, whose recipient belongs to this wallet, and are expired
func (w *OwnerWallet) ListExpiredReceivedTokens(opts ...token.ListTokensOption) (*token2.UnspentTokens, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filter(compiledOpts.TokenType, false, DeadlineBefore)
}

// ListExpiredReceivedTokensIterator returns an iterator of tokens that matches the passed options, whose recipient belongs to this wallet, and are expired
func (w *OwnerWallet) ListExpiredReceivedTokensIterator(opts ...token.ListTokensOption) (*FilteredIterator, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filterIterator(compiledOpts.TokenType, false, DeadlineBefore)
}

func (w *OwnerWallet) CleanupExpiredReceivedTokens(context view.Context, opts ...token.ListTokensOption) (*token2.UnspentTokens, error) {
	expiredTokens, err := w.ListExpiredReceivedTokens(opts...)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to fetch list of expired received tokens")
	}

	// get spent flags
	ids := make([]*token2.ID, expiredTokens.Count())
	for i, tok := range expiredTokens.Tokens {
		ids[i] = tok.Id
	}
	tms := w.wallet.TMS()
	spentIDs, err := tms.WalletManager().SpentIDs(ids)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to compute spent ids for [%v]", ids)
	}
	net := network.GetInstance(context, tms.Network(), tms.Channel())
	if net == nil {
		return nil, errors.Errorf("cannot load network [%s:%s]", tms.Network(), tms.Channel())
	}
	spent, err := net.AreTokensSpent(context, tms.Namespace(), spentIDs)
	if err != nil {
		return nil, errors.Errorf("cannot to fetch spent flags from network [%s:%s] for ids [%v]", tms.Network(), tms.Channel(), ids)
	}

	// remove the tokens flagged as spent
	var res []*token2.UnspentToken
	for i, unspentToken := range expiredTokens.Tokens {
		if spent[i] {
			logger.Debugf("token [%s] is spent", unspentToken.Id)
		} else {
			logger.Debugf("token [%s] is not spent", unspentToken.Id)
			res = append(res, unspentToken)
		}
	}
	return &token2.UnspentTokens{Tokens: res}, nil
}

func (w *OwnerWallet) filter(tokenType string, sender bool, pick PickFunction) (*token2.UnspentTokens, error) {
	it, err := w.filterIterator(tokenType, sender, pick)
	if err != nil {
		return nil, errors.Wrap(err, "token selection failed")
	}
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

func (w *OwnerWallet) filterIterator(tokenType string, sender bool, pick PickFunction) (*FilteredIterator, error) {
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
		it:   it,
		pick: pick,
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
		queryService: vault.TokenVault().QueryEngine(),
	}
}

type FilteredIterator struct {
	it   driver.UnspentTokensIterator
	pick PickFunction
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

			pickItem, err := f.pick(tok, script)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pick (token,script)[%v:%v] pair", tok, script)
			}
			if pickItem {
				return tok, nil
			}
		}
	}
}
