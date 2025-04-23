/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"bytes"
	"io"
	"os"
	"text/template"
	"time"

	math3 "github.com/IBM/mathlib"
	api2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	sfcnode "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/dlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/fabtoken"
	topology2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	. "github.com/onsi/gomega"
)

var logger = logging.MustGetLogger("token-sdk.integration.token.fabric")

type Entry struct {
	TMS     *topology2.TMS
	CA      common2.CA
	Wallets map[string]*topology2.Wallets
}

type Backend interface {
	PrepareNamespace(tms *topology2.TMS)
	UpdatePublicParams(tms *topology2.TMS, raw []byte)
}

type NetworkHandler struct {
	common2.NetworkHandler
	Entries map[string]*Entry
	Backend Backend
}

func NewNetworkHandler(tokenPlatform common2.TokenPlatform, builder api2.Builder, backend Backend) *NetworkHandler {
	return &NetworkHandler{
		NetworkHandler: common2.NetworkHandler{
			TokenPlatform:     tokenPlatform,
			EventuallyTimeout: 10 * time.Minute,
			CryptoMaterialGenerators: map[string]generators.CryptoMaterialGenerator{
				"fabtoken": fabtoken.NewCryptoMaterialGenerator(tokenPlatform, builder),
				"dlog":     dlog.NewCryptoMaterialGenerator(tokenPlatform, math3.BN254, builder),
			},
			CASupports: map[string]common2.CAFactory{
				"dlog": common2.NewIdemixCASupport,
			},
		},
		Entries: map[string]*Entry{},
		Backend: backend,
	}
}

func (p *NetworkHandler) GenerateArtifacts(tms *topology2.TMS) {
	entry := p.GetEntry(tms)

	// Generate crypto material
	cmGenerator := p.CryptoMaterialGenerators[tms.Driver]
	Expect(cmGenerator).NotTo(BeNil(), "Crypto material generator for driver %s not found", tms.Driver)

	// - Setup
	root, err := cmGenerator.Setup(tms)
	Expect(err).NotTo(HaveOccurred())

	// - Generate crypto material for each FSC node
	fscTopology := p.TokenPlatform.GetContext().TopologyByName(fsc.TopologyName).(*fsc.Topology)
	for _, node := range fscTopology.Nodes {
		p.GenerateCryptoMaterial(cmGenerator, tms, node)
	}

	// Generate public parameters
	var ppRaw []byte
	ppGenerator := p.TokenPlatform.GetPublicParamsGenerators(tms.Driver)
	Expect(ppGenerator).NotTo(BeNil(), "No public params generator for driver %s", tms.Driver)
	args := []interface{}{root}
	for _, arg := range tms.PublicParamsGenArgs {
		args = append(args, arg)
	}

	logger.Debugf("Generating public parameters for [%s:%s] with args [%+v]", tms.ID(), args)
	wallets := &topology2.Wallets{}
	for _, w := range entry.Wallets {
		wallets.Issuers = append(wallets.Issuers, w.Issuers...)
		wallets.Auditors = append(wallets.Auditors, w.Auditors...)
		wallets.Certifiers = append(wallets.Certifiers, w.Certifiers...)
	}
	ppRaw, err = ppGenerator.Generate(tms, wallets, args...)
	Expect(err).ToNot(HaveOccurred())

	// - Store pp
	Expect(os.MkdirAll(p.TokenPlatform.PublicParametersDir(), 0766)).ToNot(HaveOccurred())
	Expect(os.WriteFile(p.TokenPlatform.PublicParametersFile(tms), ppRaw, 0766)).ToNot(HaveOccurred())

	// Prepare namespace
	p.Backend.PrepareNamespace(tms)

	// Prepare CA, if needed
	if IsFabricCA(tms) {
		caFactory, found := p.CASupports[tms.Driver]
		if found {
			ca, err := caFactory(p.TokenPlatform, tms, root)
			Expect(err).ToNot(HaveOccurred(), "failed to instantiate CA for [%s]", tms.ID())
			entry.CA = ca
		}
	}
}

func (p *NetworkHandler) GenerateExtension(tms *topology2.TMS, node *sfcnode.Node, uniqueName string) string {
	Expect(os.MkdirAll(p.TTXDBSQLDataSourceDir(uniqueName), 0775)).ToNot(HaveOccurred(), "failed to create [%s]", p.TTXDBSQLDataSourceDir(uniqueName))
	Expect(os.MkdirAll(p.TokensDBSQLDataSourceDir(uniqueName), 0775)).ToNot(HaveOccurred(), "failed to create [%s]", p.TokensDBSQLDataSourceDir(uniqueName))
	Expect(os.MkdirAll(p.AuditDBSQLDataSourceDir(uniqueName), 0775)).ToNot(HaveOccurred(), "failed to create [%s]", p.AuditDBSQLDataSourceDir(uniqueName))
	Expect(os.MkdirAll(p.IdentityDBSQLDataSourceDir(uniqueName), 0775)).ToNot(HaveOccurred(), "failed to create [%s]", p.IdentityDBSQLDataSourceDir(uniqueName))

	persistenceNames := fsc.GetPersistenceNames(node.Options, common2.AllPrefixes...)

	t, err := template.New("peer").Funcs(template.FuncMap{
		"TMSID":                func() string { return tms.TmsID() },
		"TMS":                  func() *topology2.TMS { return tms },
		"Wallets":              func() *topology2.Wallets { return p.GetEntry(tms).Wallets[node.Name] },
		"TokenPersistence":     func() driver.PersistenceName { return persistenceNames[common2.TokenKey] },
		"IdentityPersistence":  func() driver.PersistenceName { return persistenceNames[common2.IdentityKey] },
		"TokenLockPersistence": func() driver.PersistenceName { return persistenceNames[common2.TokenLockKey] },
		"AuditTxPersistence":   func() driver.PersistenceName { return persistenceNames[common2.AuditTransactionKey] },
		"OwnerTxPersistence":   func() driver.PersistenceName { return persistenceNames[common2.OwnerTransactionKey] },
		"Endorsement":          func() bool { return IsFSCEndorsementEnabled(tms) },
		"Endorsers":            func() []string { return Endorsers(tms) },
		"EndorserID":           func() string { return "" },
		"Endorser":             func() bool { return topology2.ToOptions(node.Options).Endorser() },
	}).Parse(Extension)
	Expect(err).NotTo(HaveOccurred())

	ext := bytes.NewBufferString("")
	err = t.Execute(io.MultiWriter(ext), p)
	Expect(err).NotTo(HaveOccurred())

	return ext.String()
}

func (p *NetworkHandler) PostRun(load bool, tms *topology2.TMS) {
	if !load {
		// Start the CA, if available
		entry := p.GetEntry(tms)
		if entry.CA != nil {
			Expect(entry.CA.Start()).ToNot(HaveOccurred(), "failed to start CA for [%s]", tms.ID())
		}
	}
}

func (p *NetworkHandler) Cleanup() {
	for _, entry := range p.Entries {
		if entry.CA != nil {
			entry.CA.Stop()
		}
	}
}

func (p *NetworkHandler) UpdatePublicParams(tms *topology2.TMS, ppRaw []byte) {
	p.Backend.UpdatePublicParams(tms, ppRaw)
}

func (p *NetworkHandler) GenIssuerCryptoMaterial(tms *topology2.TMS, nodeID string, walletID string) string {
	cmGenerator := p.CryptoMaterialGenerators[tms.Driver]
	Expect(cmGenerator).NotTo(BeNil(), "Crypto material generator for driver %s not found", tms.Driver)

	fscTopology := p.TokenPlatform.GetContext().TopologyByName(fsc.TopologyName).(*fsc.Topology)
	for _, node := range fscTopology.Nodes {
		if node.ID() == nodeID {
			ids := cmGenerator.GenerateIssuerIdentities(tms, node, walletID)
			return ids[0].Path
		}
	}
	Expect(false).To(BeTrue(), "cannot find FSC node [%s:%s]", tms.Network, nodeID)
	return ""
}

func (p *NetworkHandler) GenOwnerCryptoMaterial(tms *topology2.TMS, nodeID string, walletID string, useCAIfAvailable bool) (res token.IdentityConfiguration) {
	if useCAIfAvailable {
		// check if the ca is available
		ca := p.GetEntry(tms).CA
		if ca != nil {
			// Use the ca
			// return the path where the credential is stored
			logger.Infof("generate owner crypto material using ca")
			ic, err := ca.Gen(walletID)
			Expect(err).ToNot(HaveOccurred(), "failed to generate owner crypto material using ca [%s]", tms.ID())
			return ic
		}
		// continue without the ca
	}

	cmGenerator := p.CryptoMaterialGenerators[tms.Driver]
	Expect(cmGenerator).NotTo(BeNil(), "Crypto material generator for driver %s not found", tms.Driver)

	fscTopology := p.TokenPlatform.GetContext().TopologyByName(fsc.TopologyName).(*fsc.Topology)
	for _, node := range fscTopology.Nodes {
		if node.ID() == nodeID {
			ids := cmGenerator.GenerateOwnerIdentities(tms, node, walletID)
			res.ID = ids[0].ID
			res.URL = ids[0].Path
			res.Raw = ids[0].Raw
			return
		}
	}
	Expect(false).To(BeTrue(), "cannot find FSC node [%s:%s]", tms.Network, nodeID)
	return
}

func (p *NetworkHandler) SetCryptoMaterialGenerator(driver string, generator generators.CryptoMaterialGenerator) {
	p.CryptoMaterialGenerators[driver] = generator
}

func (p *NetworkHandler) GenerateCryptoMaterial(cmGenerator generators.CryptoMaterialGenerator, tms *topology2.TMS, node *sfcnode.Node) {
	entry := p.GetEntry(tms)
	o := node.PlatformOpts()
	opts := topology2.ToOptions(o)

	wallet := &topology2.Wallets{
		Certifiers: []topology2.Identity{},
		Issuers:    []topology2.Identity{},
		Owners:     []topology2.Identity{},
		Auditors:   []topology2.Identity{},
	}
	entry.Wallets[node.Name] = wallet

	// Issuer identities
	issuers := opts.Issuers()
	if len(issuers) != 0 {
		var index int
		found := false
		for i, issuer := range issuers {
			if issuer == node.ID() || issuer == "_default_" {
				index = i
				found = true
				issuers[i] = node.ID()
				break
			}
		}
		if !found {
			issuers = append(issuers, node.ID())
			index = len(issuers) - 1
		}

		ids := cmGenerator.GenerateIssuerIdentities(tms, node, issuers...)
		if len(ids) > 0 {
			wallet.Issuers = append(wallet.Issuers, ids...)
			wallet.Issuers[index].Default = true
		}
	}

	// Owner identities
	owners := opts.Owners()
	if len(owners) != 0 {
		var index int
		found := false
		for i, owner := range owners {
			if owner == node.ID() || owner == "_default_" {
				index = i
				found = true
				owners[i] = node.ID()
				break
			}
		}
		if !found {
			owners = append(owners, node.ID())
			index = len(owners) - 1
		}
		ids := cmGenerator.GenerateOwnerIdentities(tms, node, owners...)
		if len(ids) > 0 {
			wallet.Owners = append(wallet.Owners, ids...)
			wallet.Owners[index].Default = true
		}
	}

	// Auditor identity
	if opts.Auditor() {
		ids := cmGenerator.GenerateAuditorIdentities(tms, node, node.Name)
		if len(ids) > 0 {
			wallet.Auditors = append(wallet.Auditors, ids...)
			wallet.Auditors[len(wallet.Auditors)-1].Default = true
		}
	}

	// Certifier identities
	if opts.Certifier() {
		ids := cmGenerator.GenerateCertifierIdentities(tms, node, node.Name)
		if len(ids) > 0 {
			wallet.Certifiers = append(wallet.Certifiers, ids...)
			wallet.Certifiers[len(wallet.Certifiers)-1].Default = true
		}
	}
}

func (p *NetworkHandler) GetEntry(tms *topology2.TMS) *Entry {
	k := tms.Network + tms.Channel + tms.Namespace + string(tms.Alias)
	entry, ok := p.Entries[k]
	if !ok {
		entry = &Entry{
			TMS:     tms,
			Wallets: map[string]*topology2.Wallets{},
		}
		p.Entries[k] = entry
	}
	return entry
}
