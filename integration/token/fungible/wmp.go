/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fungible

import (
	"path/filepath"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/config"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/registry"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	topology2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/certification"
	identity2 "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/deserializer"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/sig"
	. "github.com/onsi/gomega"
)

var logger = flogging.MustGetLogger("token-sdk.fungible")

// WalletManagerProvider is used to simulate external wallets.
// It can generate recipient data and signatures for given users and wallets
type WalletManagerProvider struct {
	Loader   WalletManagerLoader
	managers map[string]*token.WalletManager
}

func NewWalletManagerProvider(loader WalletManagerLoader) *WalletManagerProvider {
	return &WalletManagerProvider{Loader: loader, managers: map[string]*token.WalletManager{}}
}

// RecipientData returns the RecipientData for the given user and wallet
func (p *WalletManagerProvider) RecipientData(user string, wallet string) *token.RecipientData {
	wm := p.load(user)
	ownerWallet := wm.OwnerWallet(wallet)
	Expect(ownerWallet).ToNot(BeNil())
	recipientIdentity, err := ownerWallet.GetRecipientIdentity()
	Expect(err).ToNot(HaveOccurred())
	auditInfo, err := ownerWallet.GetAuditInfo(recipientIdentity)
	Expect(err).ToNot(HaveOccurred())
	tokenMetadata, err := ownerWallet.GetTokenMetadata(recipientIdentity)
	Expect(err).ToNot(HaveOccurred())
	tokenIdentityMetadata, err := ownerWallet.GetTokenMetadataAuditInfo(recipientIdentity)
	Expect(err).ToNot(HaveOccurred())
	logger.Debugf("new recipient data [%s:%s:%s:%s]", recipientIdentity, auditInfo, tokenMetadata, tokenIdentityMetadata)
	return &token.RecipientData{
		Identity:               recipientIdentity,
		AuditInfo:              auditInfo,
		TokenMetadata:          tokenMetadata,
		TokenMetadataAuditInfo: tokenIdentityMetadata,
	}
}

// GetSinger returns a signer for the given user, wallet and identity
func (p *WalletManagerProvider) GetSinger(user string, wallet string, party view.Identity) (token.Signer, error) {
	wm := p.load(user)
	ownerWallet := wm.OwnerWallet(wallet)
	Expect(ownerWallet).ToNot(BeNil())
	return ownerWallet.GetSigner(party)
}

// SignerProvider returns the SignerProvider for the given user and wallet
func (p *WalletManagerProvider) SignerProvider(user string, wallet string) *SignerProvider {
	return NewSignerProvider(p, user, wallet)
}

func (p *WalletManagerProvider) load(user string) *token.WalletManager {
	m, ok := p.managers[user]
	if ok {
		return m
	}

	wm := p.Loader.Load(user)

	p.managers[user] = wm
	return wm
}

type TMSTopology interface {
	GetTopology() *token2.Topology
	PublicParameters(tms *topology2.TMS) []byte
}

type WalletManagerLoader interface {
	Load(user string) *token.WalletManager
}

type walletManagerLoader struct {
	II *integration.Infrastructure
}

func (l *walletManagerLoader) Load(user string) *token.WalletManager {
	ctx := l.II.Ctx
	tp := ctx.PlatformByName("token").(TMSTopology)
	tms := tp.GetTopology().TMSs[0]
	ppRaw := tp.PublicParameters(tms)

	// prepare a service provider with the required services
	sp := registry.New()
	configProvider, err := config.NewProvider(filepath.Join(ctx.RootDir(), "fsc", "nodes", node.ReplicaUniqueName(user, 0)))
	Expect(err).ToNot(HaveOccurred())
	Expect(sp.RegisterService(configProvider)).ToNot(HaveOccurred())
	kvss, err := kvs.NewWithConfig(sp, "memory", "", configProvider)
	Expect(err).ToNot(HaveOccurred())
	Expect(sp.RegisterService(kvss)).ToNot(HaveOccurred())
	sigService := sig.NewService(deserializer.NewMultiplexDeserializer(), kvs2.NewIdentityDB(kvss, token.TMSID{Network: "pineapple"}))
	Expect(sp.RegisterService(sigService)).ToNot(HaveOccurred())
	Expect(sp.RegisterService(identity2.NewKVSStorageProvider(kvss))).ToNot(HaveOccurred())
	Expect(sp.RegisterService(certification.NewKVSStorageProvider(kvss))).ToNot(HaveOccurred())

	wm, err := token.NewWalletManager(sp, tms.Network, tms.Channel, tms.Namespace, ppRaw)
	Expect(err).ToNot(HaveOccurred())
	return wm
}

// SignerProvider provides instances of the token.Signer interface for the passed identity
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
