/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"bytes"
	"fmt"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"
	"time"

	api2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/topology"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	sfcnode "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators"
	topology2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
)

var logger = flogging.MustGetLogger("integration.token.fabric")

const (
	DefaultTokenChaincode = "github.com/hyperledger-labs/fabric-token-sdk/token/services/tcc/main"
)

type fabricPlatform interface {
	DeployChaincode(chaincode *topology.ChannelChaincode)
	InvokeChaincode(cc *topology.ChannelChaincode, method string, args ...[]byte) []byte
	DefaultIdemixOrgMSPDir() string
	Topology() *topology.Topology
	PeerChaincodeAddress(peerName string) string
	PeersByID(id string) *fabric.Peer
}

type tokenPlatform interface {
	TokenGen(keygen common.Command) (*Session, error)
	PublicParametersFile(tms *topology2.TMS) string
	GetContext() api2.Context
	PublicParameters(tms *topology2.TMS) []byte
	GetPublicParamsGenerators(driver string) generators.PublicParamsGenerator
	GetCryptoMaterialGenerator(driver string) generators.CryptoMaterialGenerator
	PublicParametersDir() string
	GetBuilder() api2.Builder
	TokenDir() string
}

type Entry struct {
	TMS     *topology2.TMS
	TCC     *TCC
	Wallets map[string]*generators.Wallets
}

type NetworkHandler struct {
	TokenPlatform      tokenPlatform
	EventuallyTimeout  time.Duration
	TokenChaincodePath string
	colorIndex         int
	Entries            map[string]*Entry
}

func NewNetworkHandler(tokenPlatform tokenPlatform) *NetworkHandler {
	return &NetworkHandler{
		TokenPlatform:      tokenPlatform,
		EventuallyTimeout:  10 * time.Minute,
		TokenChaincodePath: DefaultTokenChaincode,
		Entries:            map[string]*Entry{},
	}
}

func (p *NetworkHandler) GenerateArtifacts(tms *topology2.TMS) {
	entry := p.GetEntry(tms)

	// Generate crypto material
	cmGenerator := p.TokenPlatform.GetCryptoMaterialGenerator(tms.Driver)

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
	Expect(ioutil.WriteFile(p.TokenPlatform.PublicParametersFile(tms), ppRaw, 0766)).ToNot(HaveOccurred())

	// Prepare chaincodes
	var chaincode *topology.ChannelChaincode

	if tms.TokenChaincode.Private {
		cc := p.Fabric(tms).Topology().AddFPCAtOrgs(
			tms.Namespace,
			tms.TokenChaincode.DockerImage,
			tms.TokenChaincode.Orgs,
		)
		cc.Chaincode.Ctor = p.TCCCtor(tms)
		chaincode = cc
	} else {
		chaincode, _ = p.PrepareTCC(tms)
		p.Fabric(tms).Topology().AddChaincode(chaincode)
	}
	entry.TCC = &TCC{Chaincode: chaincode}
}

func (p *NetworkHandler) GenerateExtension(tms *topology2.TMS, node *sfcnode.Node) string {
	t, err := template.New("peer").Funcs(template.FuncMap{
		"TMS":     func() *topology2.TMS { return tms },
		"Wallets": func() *generators.Wallets { return p.GetEntry(tms).Wallets[node.Name] },
	}).Parse(Extension)
	Expect(err).NotTo(HaveOccurred())

	ext := bytes.NewBufferString("")
	err = t.Execute(io.MultiWriter(ext), p)
	Expect(err).NotTo(HaveOccurred())

	return ext.String()
}

func (p *NetworkHandler) PostRun(load bool, tms *topology2.TMS) {
	if !load {
		p.setupTokenChaincodes(tms)
	}
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
		for i, owner := range issuers {
			if owner == node.ID() || owner == "_default_" {
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
			for _, id := range ids {
				wallet.Issuers = append(wallet.Issuers, generators.Identity(id))
			}
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
			for _, id := range ids {
				wallet.Owners = append(wallet.Owners, generators.Identity(id))
			}
			wallet.Owners[index].Default = true
		}
	}

	// Auditor identity
	if opts.Auditor() {
		ids := cmGenerator.GenerateAuditorIdentities(tms, node, node.Name)
		if len(ids) > 0 {
			for _, id := range ids {
				wallet.Auditors = append(wallet.Auditors, generators.Identity(id))
			}
			wallet.Auditors[len(wallet.Auditors)-1].Default = true
		}
	}

	// Certifier identities
	if opts.Certifier() {
		ids := cmGenerator.GenerateCertifierIdentities(tms, node, node.Name)
		if len(ids) > 0 {
			for _, id := range ids {
				wallet.Certifiers = append(wallet.Certifiers, generators.Identity(id))
			}
			wallet.Certifiers[len(wallet.Certifiers)-1].Default = true
		}
	}
}

func (p *NetworkHandler) Fabric(tms *topology2.TMS) fabricPlatform {
	return p.TokenPlatform.GetContext().PlatformByName(tms.Network).(fabricPlatform)
}

func (p *NetworkHandler) FSCCertifierCryptoMaterialDir(tms *topology2.TMS, peer *sfcnode.Node) string {
	return filepath.Join(
		p.TokenPlatform.GetContext().RootDir(),
		"crypto",
		"fsc",
		peer.ID(),
		"wallets",
		"certifier",
		fmt.Sprintf("%s_%s_%s_%s", tms.Network, tms.Channel, tms.Namespace, tms.Driver),
	)
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
