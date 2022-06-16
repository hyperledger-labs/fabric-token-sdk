/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package exchange

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	fabric3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type QueryService interface {
	ListUnspentTokens() (*token2.UnspentTokens, error)
}

type OwnerWallet struct {
	wallet       *token.OwnerWallet
	queryService QueryService
}

// ListExpired returns a list of tokens with a passed deadline whose sender id is contained within the wallet
func (w *OwnerWallet) ListExpired() ([]*token2.UnspentToken, error) {
	unspentTokens, err := w.queryService.ListUnspentTokens()
	if err != nil {
		return nil, errors.Wrap(err, "token selection failed")
	}

	logger.Debugf("[%d] unspent tokens found", len(unspentTokens.Tokens))
	var res []*token2.UnspentToken
	for _, tok := range unspentTokens.Tokens {
		logger.Debugf("token [%s,%s,%s,%s] contains a script?", tok.Id, view.Identity(tok.Owner.Raw).UniqueID(), tok.Type, tok.Quantity)

		owner, err := identity.UnmarshallRawOwner(tok.Owner.Raw)
		if err != nil {
			logger.Debugf("Is Mine [%s,%s,%s]? No, failed unmarshalling [%s]", view.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, err)
			continue
		}
		if owner.Type == ScriptTypeExchange {
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

			// is this expired and I am the sender?
			now := time.Now()
			logger.Debugf("[%v]<=[%v] and sender [%s]?", script.Deadline, now, script.Sender.UniqueID())
			if script.Deadline.Before(now) && w.wallet.Contains(script.Sender) {
				logger.Debugf("[%v]<=[%v] and sender [%s]? Yes", script.Deadline, now, script.Sender.UniqueID())
				res = append(res, tok)
			}
			logger.Debugf("[%v]<=[%v] and sender [%s]? No", script.Deadline, now, script.Sender.UniqueID())
		}
	}
	return res, nil
}

// ListByPreImage returns a list of tokens with a matching preimage
func (w *OwnerWallet) ListByPreImage(preImage []byte) ([]*token2.UnspentToken, error) {
	unspentTokens, err := w.queryService.ListUnspentTokens()
	if err != nil {
		return nil, errors.Wrap(err, "token selection failed")
	}
	logger.Debugf("[%d] unspent tokens found", len(unspentTokens.Tokens))
	var res []*token2.UnspentToken
	for _, tok := range unspentTokens.Tokens {
		logger.Debugf("token [%s,%s,%s,%s] contains a script?", tok.Id, view.Identity(tok.Owner.Raw).UniqueID(), tok.Type, tok.Quantity)

		owner, err := identity.UnmarshallRawOwner(tok.Owner.Raw)
		if err != nil {
			logger.Debugf("Is Mine [%s,%s,%s]? No, failed unmarshalling [%s]", view.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, err)
			continue
		}
		if owner.Type == ScriptTypeExchange {
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

			hash := script.HashInfo.HashFunc.New()
			if _, err = hash.Write(preImage); err != nil {
				return nil, err
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

			if bytes.Equal(h, script.HashInfo.Hash) && w.wallet.Contains(script.Recipient) {
				res = append(res, tok)
			}
		}
	}
	return res, nil
}

// GetWallet returns the wallet whose id is the passed id.
// If the passed id is empty, GetWallet has the same behaviour of MyWallet.
// It returns nil, if no wallet is found.
func GetWallet(sp view2.ServiceProvider, id string, opts ...token.ServiceOption) *token.OwnerWallet {
	return ttx.GetWallet(sp, id, opts...)
}

func Wallet(sp view2.ServiceProvider, wallet *token.OwnerWallet, opts ...token.ServiceOption) *OwnerWallet {
	tms := token.GetManagementService(sp, opts...)
	ch := fabric.GetChannel(sp, tms.Network(), tms.Channel())
	v := vault.New(
		sp,
		ch.Name(),
		tms.Namespace(),
		fabric3.NewVault(ch),
	)
	return &OwnerWallet{
		wallet:       wallet,
		queryService: v.QueryEngine(),
	}
}
