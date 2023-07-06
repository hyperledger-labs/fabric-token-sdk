/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fungible

import (
	"path/filepath"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/config"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/sig"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/registry"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	. "github.com/onsi/gomega"
)

type WalletManagerProvider struct {
	II       *integration.Infrastructure
	Managers map[string]*token.WalletManager
}

func NewWalletManagerProvider(II *integration.Infrastructure) *WalletManagerProvider {
	return &WalletManagerProvider{II: II, Managers: map[string]*token.WalletManager{}}
}

func (p *WalletManagerProvider) RecipientData(user string, wallet string) *token.RecipientData {
	wm := p.load(user)
	ownerWallet := wm.OwnerWallet(wallet)
	Expect(ownerWallet).ToNot(BeNil())
	recipientIdentity, err := ownerWallet.GetRecipientIdentity()
	Expect(err).ToNot(HaveOccurred())
	auditInfo, err := ownerWallet.GetAuditInfo(recipientIdentity)
	Expect(err).ToNot(HaveOccurred())
	metadata, err := ownerWallet.GetTokenMetadata(recipientIdentity)
	Expect(err).ToNot(HaveOccurred())
	return &token.RecipientData{
		Identity:  recipientIdentity,
		AuditInfo: auditInfo,
		Metadata:  metadata,
	}
}

func (p *WalletManagerProvider) GetSinger(user string, wallet string, party view.Identity) (token.Signer, error) {
	wm := p.load(user)
	ownerWallet := wm.OwnerWallet(wallet)
	Expect(ownerWallet).ToNot(BeNil())
	return ownerWallet.GetSigner(party)
}

func (p *WalletManagerProvider) load(user string) *token.WalletManager {
	m, ok := p.Managers[user]
	if ok {
		return m
	}

	tp := p.II.Ctx.PlatformByName("token").(*token2.Platform)
	tms := tp.Topology.TMSs[0]
	ppRaw := tp.PublicParameters(tms)

	// prepare a service provider with the required services
	sp := registry.New()
	configProvider, err := config.NewProvider(filepath.Join(p.II.Ctx.RootDir(), "fsc", "nodes", user))
	Expect(err).ToNot(HaveOccurred())
	Expect(sp.RegisterService(configProvider)).ToNot(HaveOccurred())
	dm, err := sig.NewMultiplexDeserializer(sp)
	Expect(err).ToNot(HaveOccurred())
	Expect(sp.RegisterService(dm)).ToNot(HaveOccurred())
	kvss, err := kvs.NewWithConfig(sp, "memory", "", configProvider)
	Expect(err).ToNot(HaveOccurred())
	Expect(sp.RegisterService(kvss))
	sigService := sig.NewSignService(sp, nil, kvss)
	Expect(sp.RegisterService(sigService))

	wm, err := token.NewWalletManager(sp, tms.Network, tms.Channel, tms.Namespace, ppRaw)
	Expect(err).ToNot(HaveOccurred())

	p.Managers[user] = wm
	return wm
}

func (p *WalletManagerProvider) SignerProvider(id string, wallet string) *SignerProvider {
	return NewSignerProvider(p, id, wallet)
}

type SignerProvider struct {
	*WalletManagerProvider
	Id, Wallet string
}

func NewSignerProvider(walletManagerProvider *WalletManagerProvider, id string, wallet string) *SignerProvider {
	return &SignerProvider{WalletManagerProvider: walletManagerProvider, Id: id, Wallet: wallet}
}

func (s *SignerProvider) GetSigner(party view.Identity) (token.Signer, error) {
	return s.WalletManagerProvider.GetSinger(s.Id, s.Wallet, party)
}
