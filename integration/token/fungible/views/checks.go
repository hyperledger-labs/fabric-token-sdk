/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TokenTransactionDB interface {
	GetTokenRequest(txID string) ([]byte, error)
	Transactions(params driver.QueryTransactionsParams) (driver.TransactionIterator, error)
}

type CheckTTXDB struct {
	Auditor         bool
	AuditorWalletID string
	TMSID           token.TMSID
}

// CheckTTXDBView is a view that performs consistency checks among the transaction db (either auditor or owner),
// the vault, and the backed. It reports a list of mismatch that can be used for debug purposes.
type CheckTTXDBView struct {
	*CheckTTXDB
}

func (m *CheckTTXDBView) Call(context view.Context) (interface{}, error) {
	// prepare
	defaultOwnerWallet := htlc.GetWallet(context, "", token.WithTMSID(m.TMSID))
	if defaultOwnerWallet != nil {
		htlcWallet := htlc.Wallet(context, defaultOwnerWallet)
		assert.NotNil(htlcWallet, "cannot load htlc wallet")
		assert.NoError(htlcWallet.DeleteClaimedSentTokens(context), "failed to delete claimed sent tokens")
		assert.NoError(htlcWallet.DeleteExpiredReceivedTokens(context), "failed to delete expired received tokens")
	}

	// check
	tms := token.GetManagementService(context, token.WithTMSID(m.TMSID))
	assert.NotNil(tms, "failed to get default tms")
	if m.Auditor {
		auditorWallet := tms.WalletManager().AuditorWallet(m.AuditorWalletID)
		assert.NotNil(auditorWallet, "cannot find auditor wallet [%s]", m.AuditorWalletID)
		db, err := ttx.NewAuditor(context, auditorWallet)
		assert.NoError(err, "failed to get auditor instance")
		return db.Check(context.Context())
	}
	db := ttx.NewOwner(context, tms)
	return db.Check(context.Context())
}

type CheckTTXDBViewFactory struct{}

func (p *CheckTTXDBViewFactory) NewView(in []byte) (view.View, error) {
	f := &CheckTTXDBView{CheckTTXDB: &CheckTTXDB{}}
	err := json.Unmarshal(in, f.CheckTTXDB)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

type PruneInvalidUnspentTokens struct {
	TMSID token.TMSID
}

type PruneInvalidUnspentTokensView struct {
	*PruneInvalidUnspentTokens
}

func (p *PruneInvalidUnspentTokensView) Call(context view.Context) (interface{}, error) {
	net := network.GetInstance(context, p.TMSID.Network, p.TMSID.Channel)
	assert.NotNil(net, "cannot find network [%s:%s]", p.TMSID.Network, p.TMSID.Channel)
	vault, err := net.TokenVault(p.TMSID.Namespace)
	assert.NoError(err, "failed to get vault for [%s:%s:%s]", p.TMSID.Network, p.TMSID.Channel, p.TMSID.Namespace)

	return vault.PruneInvalidUnspentTokens(context)
}

type PruneInvalidUnspentTokensViewFactory struct{}

func (p *PruneInvalidUnspentTokensViewFactory) NewView(in []byte) (view.View, error) {
	f := &PruneInvalidUnspentTokensView{PruneInvalidUnspentTokens: &PruneInvalidUnspentTokens{}}
	err := json.Unmarshal(in, f.PruneInvalidUnspentTokens)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

type ListVaultUnspentTokens struct {
	TMSID token.TMSID
}

type ListVaultUnspentTokensView struct {
	*ListVaultUnspentTokens
}

func (l *ListVaultUnspentTokensView) Call(context view.Context) (interface{}, error) {
	net := token.GetManagementService(context, token.WithTMSID(l.TMSID))
	assert.NotNil(net, "cannot find tms [%s]", l.TMSID)
	return net.Vault().NewQueryEngine().ListUnspentTokens()
}

type ListVaultUnspentTokensViewFactory struct{}

func (l *ListVaultUnspentTokensViewFactory) NewView(in []byte) (view.View, error) {
	f := &ListVaultUnspentTokensView{ListVaultUnspentTokens: &ListVaultUnspentTokens{}}
	err := json.Unmarshal(in, f.ListVaultUnspentTokens)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

type CheckIfExistsInVault struct {
	TMSID token.TMSID
	IDs   []*token2.ID
}

type CheckIfExistsInVaultView struct {
	*CheckIfExistsInVault
}

func (c *CheckIfExistsInVaultView) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(context, token.WithTMSID(c.TMSID))
	assert.NotNil(tms, "cannot find tms [%s]", c.TMSID)
	qe := tms.Vault().NewQueryEngine()
	var IDs []*token2.ID
	count := 0
	assert.NoError(qe.GetTokenOutputs(c.IDs, func(id *token2.ID, tokenRaw []byte) error {
		if len(tokenRaw) == 0 {
			return errors.Errorf("token id [%s] is nil", id)
		}
		IDs = append(IDs, id)
		count++
		return nil
	}), "failed to match tokens")
	assert.Equal(len(c.IDs), count, "got a mismatch; count is [%d] while there are [%d] ids", count, len(c.IDs))
	return IDs, nil
}

type CheckIfExistsInVaultViewFactory struct {
}

func (c *CheckIfExistsInVaultViewFactory) NewView(in []byte) (view.View, error) {
	f := &CheckIfExistsInVaultView{CheckIfExistsInVault: &CheckIfExistsInVault{}}
	err := json.Unmarshal(in, f.CheckIfExistsInVault)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
