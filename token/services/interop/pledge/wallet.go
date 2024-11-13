/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

func MyOwnerWallet(sp view.ServiceProvider) (*token.OwnerWallet, error) {
	w := token.GetManagementService(sp).WalletManager().OwnerWallet("")
	if w == nil {
		return nil, errors.Errorf("owner Wallet needs to be initialized")
	}
	return w, nil
}

func GetOwnerWallet(sp view.ServiceProvider, id string, opts ...token.ServiceOption) (*token.OwnerWallet, error) {
	w := token.GetManagementService(sp, opts...).WalletManager().OwnerWallet(id)
	if w == nil {
		return nil, errors.Errorf("owner Wallet needs to be initialized")
	}
	return w, nil
}

func GetIssuerWallet(sp view.ServiceProvider, id string, opts ...token.ServiceOption) (*token.IssuerWallet, error) {
	w := token.GetManagementService(sp, opts...).WalletManager().IssuerWallet(id)
	if w == nil {
		return nil, errors.Errorf("issuer Wallet needs to be initialized")
	}
	return w, nil
}

type QueryService interface {
	// TODO: switch to UnspentTokensIteratorBy(id, typ string) (UnspentTokensIterator, error)
	ListUnspentTokens() (*token2.UnspentTokens, error)
}

type IssuerWallet struct {
	wallet       *token.IssuerWallet
	queryService QueryService
}

func NewIssuerWallet(sp view.ServiceProvider, wallet *token.IssuerWallet) *IssuerWallet {
	tmsID := wallet.TMS().ID()
	net := network.GetInstance(sp, tmsID.Network, tmsID.Channel)
	if net == nil {
		logger.Errorf("could not find network [%s]", tmsID)
		return nil
	}
	v, err := net.TokenVault(tmsID.Namespace)
	if err != nil {
		logger.Errorf("failed to get vault for [%s]: [%s]", tmsID, err)
	}

	return &IssuerWallet{
		wallet:       wallet,
		queryService: v.QueryEngine(),
	}
}

func (w *IssuerWallet) GetPledgedToken(tokenID *token2.ID) (*token2.UnspentToken, *Script, error) {
	unspentTokens, err := w.queryService.ListUnspentTokens()
	if err != nil {
		return nil, nil, errors.Wrap(err, "token selection failed")
	}
	return retrievePledgedToken(unspentTokens, tokenID)
}

type OwnerWallet struct {
	wallet       *token.OwnerWallet
	queryService QueryService
}

// GetWallet returns the wallet whose id is the passed id.
// If the passed id is empty, GetWallet has the same behaviour of MyWallet.
// It returns nil, if no wallet is found.
func GetWallet(sp view.ServiceProvider, id string, opts ...token.ServiceOption) *token.OwnerWallet {
	w := token.GetManagementService(sp, opts...).WalletManager().OwnerWallet(id)
	if w == nil {
		return nil
	}
	return w
}

func Wallet(sp view.ServiceProvider, wallet *token.OwnerWallet) *OwnerWallet {
	tmsID := wallet.TMS().ID()
	net := network.GetInstance(sp, tmsID.Network, tmsID.Channel)
	if net == nil {
		logger.Errorf("could not find network [%s]", tmsID)
		return nil
	}
	v, err := net.TokenVault(tmsID.Namespace)
	if err != nil {
		logger.Errorf("failed to get vault for [%s]: [%s]", tmsID, err)
	}

	return &OwnerWallet{
		wallet:       wallet,
		queryService: v.QueryEngine(),
	}
}

func (w *OwnerWallet) GetPledgedToken(tokenID *token2.ID) (*token2.UnspentToken, *Script, error) {
	unspentTokens, err := w.queryService.ListUnspentTokens()
	if err != nil {
		return nil, nil, errors.Wrap(err, "token selection failed")
	}

	return retrievePledgedToken(unspentTokens, tokenID)
}

func retrievePledgedToken(unspentTokens *token2.UnspentTokens, tokenID *token2.ID) (*token2.UnspentToken, *Script, error) {
	logger.Debugf("[%d] unspent tokens found, search [%s]", len(unspentTokens.Tokens), tokenID)

	var res []*token2.UnspentToken
	var scripts []*Script
	for _, tok := range unspentTokens.Tokens {
		if tok.Id.String() == tokenID.String() {
			owner, err := identity.UnmarshalTypedIdentity(tok.Owner.Raw)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to unmarshal owner")
			}
			logger.Debugf("found token [%s] with type [%s]", tokenID, owner.Type)
			if owner.Type == ScriptType {
				res = append(res, tok)
				script := &Script{}
				if err := json.Unmarshal(owner.Identity, script); err != nil {
					return nil, nil, errors.Wrapf(err, "failed unmarshalling pledge script")
				}
				scripts = append(scripts, script)
			}
		}
	}
	if len(res) > 1 {
		return nil, nil, errors.Errorf("multiple pledged tokens with the same identifier [%s]", tokenID.String())
	}
	if len(res) == 0 {
		return nil, nil, errors.Errorf("no pledged token exists with identifier [%s]", tokenID.String())
	}
	return res[0], scripts[0], nil
}

// ScriptAuth implements the Authorization interface for this script
type ScriptAuth struct {
	WalletService driver.WalletService
}

func NewScriptAuth(walletService driver.WalletService) *ScriptAuth {
	return &ScriptAuth{WalletService: walletService}
}

// AmIAnAuditor returns false for script ownership
func (s *ScriptAuth) AmIAnAuditor() bool {
	return false
}

// IsMine returns true if either the sender or the recipient is in one of the owner wallets.
// It returns an empty wallet id.
func (s *ScriptAuth) IsMine(tok *token2.Token) (string, []string, bool) {
	identity, err := identity.UnmarshalTypedIdentity(tok.Owner.Raw)
	if err != nil {
		logger.Debugf("Is Mine [%s,%s,%s]? No, failed unmarshalling [%s]", view2.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, err)
		return "", nil, false
	}
	if identity.Type != ScriptType {
		return "", nil, false
	}

	script := &Script{}
	if err := json.Unmarshal(identity.Identity, script); err != nil {
		logger.Debugf("Is Mine [%s,%s,%s]? No, failed unmarshalling [%s]", view2.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, err)
		return "", nil, false
	}
	if script.Sender.IsNone() || script.Recipient.IsNone() || script.Issuer.IsNone() {
		logger.Debugf("Is Mine [%s,%s,%s]? No, invalid content [%v]", view2.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, script)
		return "", nil, false
	}

	// I'm either a sender, recipient, or issuer
	var ids []string
	for _, beneficiary := range []struct {
		identity view2.Identity
		desc     string
		prefix   string
	}{
		{
			identity: script.Sender,
			desc:     "sender",
			prefix:   "pledge.sender",
		},
		{
			identity: script.Recipient,
			desc:     "recipient",
			prefix:   "pledge.recipient",
		},
		{
			identity: script.Issuer,
			desc:     "issuer",
			prefix:   "pledge.issuer",
		},
	} {
		logger.Debugf("Is Mine [%s,%s,%s] as a %s?", view2.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, beneficiary.desc)
		// TODO: differentiate better
		if wallet, err := s.WalletService.OwnerWallet(beneficiary.identity); err == nil {
			logger.Debugf("Is Mine [%s,%s,%s] as a %s? Yes", view2.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, beneficiary.desc)
			ids = append(ids, beneficiary.prefix+wallet.ID())
		}
	}
	logger.Debugf("Is Mine [%s,%s,%s]? [%b] with [%s]", view2.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, len(ids) != 0, ids)
	return "", ids, len(ids) != 0
}

func (s *ScriptAuth) Issued(issuer driver.Identity, tok *token2.Token) bool {
	return false
}

func (s *ScriptAuth) OwnerType(raw []byte) (string, []byte, error) {
	owner, err := identity.UnmarshalTypedIdentity(raw)
	if err != nil {
		return "", nil, err
	}
	return owner.Type, owner.Identity, nil
}
