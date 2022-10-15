/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"time"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

// PickFunction is a prototype for (token,script) pair selection
type PickFunction = func(*token2.UnspentToken, *Script) (bool, error)

type QueryEngine interface {
	ListUnspentTokens() (*token2.UnspentTokens, error)
	// UnspentTokensIterator returns an iterator over all unspent tokens
	UnspentTokensIterator() (driver.UnspentTokensIterator, error)
	// UnspentTokensIteratorBy returns an iterator over all unspent tokens by type and id
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

	return w.filter(func(tok *token2.UnspentToken, script *Script) (bool, error) {
		logger.Debugf("token [%s,%s,%s,%s] contains a script? Yes", tok.Id, view.Identity(tok.Owner.Raw).UniqueID(), tok.Type, tok.Quantity)

		if len(compiledOpts.TokenType) != 0 && tok.Type != compiledOpts.TokenType {
			logger.Debugf("discarding token of type [%s]!=[%s]", tok.Type, compiledOpts.TokenType)
			return false, nil
		}

		// is this expired and I am the sender?
		now := time.Now()
		logger.Debugf("[%v]<=[%v] and sender [%s]?", script.Deadline, now, script.Sender.UniqueID())
		if script.Deadline.Before(now) && w.wallet.Contains(script.Sender) {
			logger.Debugf("[%v]<=[%v] and sender [%s]? Yes", script.Deadline, now, script.Sender.UniqueID())
			return true, nil
		}
		logger.Debugf("[%v]<=[%v] and sender [%s]? No", script.Deadline, now, script.Sender.UniqueID())
		return false, nil
	})
}

// ListByPreImage returns a list of tokens with a matching preimage
func (w *OwnerWallet) ListByPreImage(preImage []byte, opts ...token.ListTokensOption) (*token2.UnspentTokens, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filter(func(tok *token2.UnspentToken, script *Script) (bool, error) {
		logger.Debugf("token [%s,%s,%s,%s] contains a script? Yes", tok.Id, view.Identity(tok.Owner.Raw).UniqueID(), tok.Type, tok.Quantity)

		if len(compiledOpts.TokenType) != 0 && tok.Type != compiledOpts.TokenType {
			logger.Debugf("discarding token of type [%s]!=[%s]", tok.Type, compiledOpts.TokenType)
			return false, nil
		}

		if !script.HashInfo.HashFunc.Available() {
			logger.Errorf("script hash function not available [%d]", script.HashInfo.HashFunc)
			return false, nil
		}
		hash := script.HashInfo.HashFunc.New()
		if _, err := hash.Write(preImage); err != nil {
			return false, err
		}
		h := hash.Sum(nil)
		h = []byte(script.HashInfo.HashEncoding.New().EncodeToString(h))

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("searching for script matching (pre-image, image) = (%s,%s)",
				base64.StdEncoding.EncodeToString(preImage),
				base64.StdEncoding.EncodeToString(h),
			)
		}

		// does the preimage match?
		logger.Debugf("token [%s,%s,%s,%s] does hashes match?", tok.Id, view.Identity(tok.Owner.Raw).UniqueID(), tok.Type, tok.Quantity,
			base64.StdEncoding.EncodeToString(h), base64.StdEncoding.EncodeToString(script.HashInfo.Hash))

		return bytes.Equal(h, script.HashInfo.Hash) && w.wallet.Contains(script.Recipient), nil
	})
}

func (w *OwnerWallet) ListByPreImageIterator(preImage []byte, opts ...token.ListTokensOption) (*FilteredIterator, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filterIterator(compiledOpts.TokenType, func(tok *token2.UnspentToken, script *Script) (bool, error) {
		logger.Debugf("token [%s,%s,%s,%s] contains a script? Yes", tok.Id, view.Identity(tok.Owner.Raw).UniqueID(), tok.Type, tok.Quantity)

		if !script.HashInfo.HashFunc.Available() {
			logger.Errorf("script hash function not available [%d]", script.HashInfo.HashFunc)
			return false, nil
		}
		hash := script.HashInfo.HashFunc.New()
		if _, err := hash.Write(preImage); err != nil {
			return false, err
		}
		h := hash.Sum(nil)
		h = []byte(script.HashInfo.HashEncoding.New().EncodeToString(h))

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("searching for script matching (pre-image, image) = (%s,%s)",
				base64.StdEncoding.EncodeToString(preImage),
				base64.StdEncoding.EncodeToString(h),
			)
		}

		// does the preimage match?
		logger.Debugf("token [%s,%s,%s,%s] does hashes match?", tok.Id, view.Identity(tok.Owner.Raw).UniqueID(), tok.Type, tok.Quantity,
			base64.StdEncoding.EncodeToString(h), base64.StdEncoding.EncodeToString(script.HashInfo.Hash))

		return bytes.Equal(h, script.HashInfo.Hash) && w.wallet.Contains(script.Recipient), nil
	})
}

// ListTokens returns a list of tokens that matches the passed options and whose recipient belongs to this wallet
func (w *OwnerWallet) ListTokens(opts ...token.ListTokensOption) (*token2.UnspentTokens, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filter(func(tok *token2.UnspentToken, script *Script) (bool, error) {
		if len(compiledOpts.TokenType) != 0 && tok.Type != compiledOpts.TokenType {
			logger.Debugf("discarding token of type [%s]!=[%s]", tok.Type, compiledOpts.TokenType)
			return false, nil
		}

		now := time.Now()
		logger.Debugf("[%v]<=[%v] and sender [%s]?", script.Deadline, now, script.Sender.UniqueID())
		return script.Deadline.After(now) && w.wallet.Contains(script.Recipient), nil
	})
}

// ListExpiredReceivedTokens returns a list of tokens that matches the passed options, whose recipient belongs to this wallet, and are expired
func (w *OwnerWallet) ListExpiredReceivedTokens(opts ...token.ListTokensOption) (*token2.UnspentTokens, error) {
	compiledOpts, err := token.CompileListTokensOption(opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}

	return w.filter(func(tok *token2.UnspentToken, script *Script) (bool, error) {
		if len(compiledOpts.TokenType) != 0 && tok.Type != compiledOpts.TokenType {
			logger.Debugf("discarding token of type [%s]!=[%s]", tok.Type, compiledOpts.TokenType)
			return false, nil
		}

		logger.Debugf("token [%s,%s,%s,%s] contains a script? Yes", tok.Id, view.Identity(tok.Owner.Raw).UniqueID(), tok.Type, tok.Quantity)
		now := time.Now()
		logger.Debugf("[%v]<=[%v] and sender [%s]?", script.Deadline, now, script.Sender.UniqueID())
		return script.Deadline.Before(now) && w.wallet.Contains(script.Recipient), nil
	})
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

func (w *OwnerWallet) filter(pick PickFunction) (*token2.UnspentTokens, error) {
	unspentTokens, err := w.queryService.ListUnspentTokens()
	if err != nil {
		return nil, errors.Wrap(err, "token selection failed")
	}
	logger.Debugf("[%d] unspent tokens found", len(unspentTokens.Tokens))
	var tokens []*token2.UnspentToken
	for _, tok := range unspentTokens.Tokens {
		logger.Debugf("token [%s,%s,%s,%s] contains a script?", tok.Id, view.Identity(tok.Owner.Raw).UniqueID(), tok.Type, tok.Quantity)

		owner, err := identity.UnmarshallRawOwner(tok.Owner.Raw)
		if err != nil {
			logger.Debugf("Is Mine [%s,%s,%s]? No, failed unmarshalling [%s]", view.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, err)
			continue
		}
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

			pickItem, err := pick(tok, script)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to pick (token,script)[%v:%v] pair", tok, script)
			}
			if pickItem {
				tokens = append(tokens, tok)
			}
		}
	}
	return &token2.UnspentTokens{Tokens: tokens}, nil
}

func (w *OwnerWallet) filterIterator(tokenType string, pick PickFunction) (*FilteredIterator, error) {
	var it driver.UnspentTokensIterator
	var err error
	if len(tokenType) != 0 {
		it, err = w.queryService.UnspentTokensIteratorBy(w.wallet.ID(), tokenType)
	} else {
		it, err = w.queryService.UnspentTokensIterator()
	}
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
	panic("implement me")
}
