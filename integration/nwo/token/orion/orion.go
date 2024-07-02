/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

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
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/dlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/fabtoken"
	topology2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/tcc"
	"github.com/hyperledger-labs/orion-sdk-go/pkg/bcdb"
	. "github.com/onsi/gomega"
)

var logger = flogging.MustGetLogger("token-sdk.integration.token.orion")

type orionPlatform interface {
	CreateDBInstance() bcdb.BCDB
	CreateUserSession(bcdb bcdb.BCDB, user string) bcdb.DBSession
}

type Entry struct {
	TMS     *topology2.TMS
	Wallets map[string]*generators.Wallets
}

type NetworkHandler struct {
	common2.NetworkHandler
	Entries map[string]*Entry
}

func NewNetworkHandler(tokenPlatform common2.TokenPlatform, builder api2.Builder) *NetworkHandler {
	return &NetworkHandler{
		NetworkHandler: common2.NetworkHandler{
			TokenPlatform:     tokenPlatform,
			EventuallyTimeout: 10 * time.Minute,
			CryptoMaterialGenerators: map[string]generators.CryptoMaterialGenerator{
				"fabtoken": fabtoken.NewCryptoMaterialGenerator(tokenPlatform, builder),
				"dlog":     dlog.NewCryptoMaterialGenerator(tokenPlatform, math3.BN254, builder),
			},
		},
		Entries: map[string]*Entry{},
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
	wallets := &generators.Wallets{}
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
}

func (p *NetworkHandler) GenerateExtension(tms *topology2.TMS, node *sfcnode.Node, uniqueName string) string {
	Expect(os.MkdirAll(p.TTXDBSQLDataSourceDir(uniqueName), 0775)).ToNot(HaveOccurred(), "failed to create [%s]", p.TTXDBSQLDataSourceDir(uniqueName))
	Expect(os.MkdirAll(p.TokensDBSQLDataSourceDir(uniqueName), 0775)).ToNot(HaveOccurred(), "failed to create [%s]", p.TokensDBSQLDataSourceDir(uniqueName))
	Expect(os.MkdirAll(p.AuditDBSQLDataSourceDir(uniqueName), 0775)).ToNot(HaveOccurred(), "failed to create [%s]", p.AuditDBSQLDataSourceDir(uniqueName))
	Expect(os.MkdirAll(p.IdentityDBSQLDataSourceDir(uniqueName), 0775)).ToNot(HaveOccurred(), "failed to create [%s]", p.IdentityDBSQLDataSourceDir(uniqueName))

	t, err := template.New("peer").Funcs(template.FuncMap{
		"TMSID":   func() string { return tms.ID() },
		"TMS":     func() *topology2.TMS { return tms },
		"Wallets": func() *generators.Wallets { return p.GetEntry(tms).Wallets[node.Name] },
		"IsCustodian": func() bool {
			custodianNode, ok := tms.BackendParams[Custodian]
			if !ok {
				return false
			}
			return custodianNode.(*sfcnode.Node).Name == node.Name
		},
		"CustodianID": func() string {
			return tms.BackendParams[Custodian].(*sfcnode.Node).Name
		},
		"SQLDriver":           func() string { return GetTokenPersistenceDriver(node.Options) },
		"SQLDataSource":       func() string { return p.GetSQLDataSource(node.Options, uniqueName, tms) },
		"TokensSQLDriver":     func() string { return GetTokenPersistenceDriver(node.Options) },
		"TokensSQLDataSource": func() string { return p.GetTokensSQLDataSource(node.Options, uniqueName, tms) },
	}).Parse(Extension)
	Expect(err).NotTo(HaveOccurred())

	ext := bytes.NewBufferString("")
	err = t.Execute(io.MultiWriter(ext), p)
	Expect(err).NotTo(HaveOccurred())

	return ext.String()
}

func (p *NetworkHandler) GetTokensSQLDataSource(opts *sfcnode.Options, uniqueName string, tms *topology2.TMS) string {
	switch GetTokenPersistenceDriver(opts) {
	case "sqlite":
		return p.DBPath(p.TokensDBSQLDataSourceDir(uniqueName), tms)
	case "postgres":
		return GetPostgresDataSource(opts)
	}
	panic("unknown driver type")

}

func (p *NetworkHandler) GetSQLDataSource(opts *sfcnode.Options, uniqueName string, tms *topology2.TMS) string {
	switch GetTokenPersistenceDriver(opts) {
	case "sqlite":
		return p.DBPath(p.TTXDBSQLDataSourceDir(uniqueName), tms)
	case "postgres":
		return GetPostgresDataSource(opts)
	}
	panic("unknown driver type")
}

func GetPostgresDataSource(opts *sfcnode.Options) string {
	if v := opts.Get("token.persistence.sql"); v != nil {
		return v.(string)
	}
	panic("unknown data source")
}

func GetTokenPersistenceDriver(opts *sfcnode.Options) string {
	if v := opts.Get("token.persistence.driver"); v != nil {
		return v.(string)
	}
	return "sqlite"
}

func (p *NetworkHandler) PostRun(load bool, tms *topology2.TMS) {
	if load {
		return
	}

	// Store the public parameters in orion
	orion, ok := p.TokenPlatform.GetContext().PlatformByName(tms.BackendTopology.Name()).(orionPlatform)
	Expect(ok).To(BeTrue(), "No orion platform found for topology %s", tms.BackendTopology.Name())
	db := orion.CreateDBInstance()
	custodianID := tms.BackendParams[Custodian].(*sfcnode.Node).Name
	session := orion.CreateUserSession(db, custodianID)
	tx, err := session.DataTx()
	Expect(err).ToNot(HaveOccurred(), "Failed to create data transaction")

	rwset := &RWSWrapper{
		db: tms.Namespace,
		me: custodianID,
		tx: tx,
	}
	w := translator.New("", rwset, "")
	ppRaw, err := os.ReadFile(p.TokenPlatform.PublicParametersFile(tms))
	Expect(err).ToNot(HaveOccurred(), "Failed to read public parameters file %s", p.TokenPlatform.PublicParametersFile(tms))
	action := &tcc.SetupAction{
		SetupParameters: ppRaw,
	}
	Expect(w.Write(action)).ToNot(HaveOccurred(), "Failed to store public parameters for namespace %s", tms.Namespace)
	_, _, err = tx.Commit(true)
	Expect(err).ToNot(HaveOccurred(), "Failed to commit transaction")
}

func (p *NetworkHandler) Cleanup() {
}

func (p *NetworkHandler) UpdatePublicParams(tms *topology2.TMS, ppRaw []byte) {
	panic("Should not be invoked")
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

	wallet := &generators.Wallets{
		Certifiers: []generators.Identity{},
		Issuers:    []generators.Identity{},
		Owners:     []generators.Identity{},
		Auditors:   []generators.Identity{},
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
	entry, ok := p.Entries[tms.Network+tms.Channel+tms.Namespace]
	if !ok {
		entry = &Entry{
			TMS:     tms,
			Wallets: map[string]*generators.Wallets{},
		}
		p.Entries[tms.Network+tms.Channel+tms.Namespace] = entry
	}
	return entry
}
