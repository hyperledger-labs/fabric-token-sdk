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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/owner"
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
	v, err := net.Vault(tmsID.Namespace)
	if err != nil {
		logger.Errorf("failed to get vault for [%s]: [%s]", tmsID, err)
	}

	return &IssuerWallet{
		wallet:       wallet,
		queryService: v,
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
	v, err := net.Vault(tmsID.Namespace)
	if err != nil {
		logger.Errorf("failed to get vault for [%s]: [%s]", tmsID, err)
	}

	return &OwnerWallet{
		wallet:       wallet,
		queryService: v,
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
	logger.Debugf("[%d] unspent tokens found", len(unspentTokens.Tokens))

	var res []*token2.UnspentToken
	var scripts []*Script
	for _, tok := range unspentTokens.Tokens {
		if tok.Id.String() == tokenID.String() {
			owner, err := owner.UnmarshallTypedIdentity(tok.Owner.Raw)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to unmarshal owner")
			}
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

type ScriptOwnership struct{}

func (s *ScriptOwnership) AmIAnAuditor(tms *token.ManagementService) bool {
	return false
}

func (s *ScriptOwnership) IsMine(tms *token.ManagementService, tok *token2.Token) ([]string, bool) {
	identity, err := owner.UnmarshallTypedIdentity(tok.Owner.Raw)
	if err != nil {
		logger.Debugf("Is Mine [%s,%s,%s]? No, failed unmarshalling [%s]", view2.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, err)
		return nil, false
	}
	if identity.Type != ScriptType {
		return nil, false
	}

	script := &Script{}
	if err := json.Unmarshal(identity.Identity, script); err != nil {
		logger.Debugf("Is Mine [%s,%s,%s]? No, failed unmarshalling [%s]", view2.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, err)
		return nil, false
	}
	if script.Sender.IsNone() || script.Recipient.IsNone() || script.Issuer.IsNone() {
		logger.Debugf("Is Mine [%s,%s,%s]? No, invalid content [%v]", view2.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, script)
		return nil, false
	}

	// I'm either a sender, recipient, or issuer
	for _, beneficiary := range []struct {
		identity view2.Identity
		desc     string
	}{
		{
			identity: script.Sender,
			desc:     "sender",
		}, {
			identity: script.Recipient,
			desc:     "recipient",
		}, {
			identity: script.Issuer,
			desc:     "issuer",
		},
	} {
		logger.Debugf("Is Mine [%s,%s,%s] as a %s?", view2.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, beneficiary.desc)
		// TODO: differentiate better
		if wallet := tms.WalletManager().OwnerWalletByIdentity(beneficiary.identity); wallet != nil {
			logger.Debugf("Is Mine [%s,%s,%s] as a %s? Yes", view2.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, beneficiary.desc)
			return []string{wallet.ID()}, true
		}
	}

	logger.Debugf("Is Mine [%s,%s,%s]? No", view2.Identity(tok.Owner.Raw), tok.Type, tok.Quantity)

	return nil, false
}
