/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/template"
	api2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit/grouper"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/topology"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	sfcnode "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/commands"
)

const (
	DefaultTokenChaincode = "github.com/hyperledger-labs/fabric-token-sdk/token/services/tcc/main"
	DefaultTokenGenPath   = "github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen"
)

var logger = flogging.MustGetLogger("integration.token")

type Builder interface {
	Build(path string) string
}

type FabricNetwork interface {
	DeployChaincode(chaincode *topology.ChannelChaincode)
	DefaultIdemixOrgMSPDir() string
	Topology() *topology.Topology
	PeerChaincodeAddress(peerName string) string
}

type PublicParamsGenerator interface {
	Generate(p *Platform, tms *TMS) ([]byte, error)
}

type TCC struct {
	TMS       *TMS
	Chaincode *topology.ChannelChaincode
}

type Identity struct {
	ID      string
	MSPType string
	MSPID   string
	Path    string
}

type Wallet struct {
	Certifiers []Identity
}

type Platform struct {
	Context                api2.Context
	Topology               *Topology
	Builder                api2.Builder
	EventuallyTimeout      time.Duration
	Wallets                map[string]*Wallet
	TCCs                   []*TCC
	PublicParamsGenerators map[string]PublicParamsGenerator
	TokenChaincodePath     string
	TokenGenPath           string

	colorIndex int
}

func NewPlatform(ctx api2.Context, t api2.Topology, builder api2.Builder) *Platform {
	return &Platform{
		Context:           ctx,
		Topology:          t.(*Topology),
		Builder:           builder,
		EventuallyTimeout: 10 * time.Minute,
		Wallets:           map[string]*Wallet{},
		TCCs:              []*TCC{},
		PublicParamsGenerators: map[string]PublicParamsGenerator{
			"fabtoken": &FabTokenPublicParamsGenerator{},
			"dlog":     &DLogPublicParamsGenerator{},
		},
		TokenChaincodePath: DefaultTokenChaincode,
		TokenGenPath:       DefaultTokenGenPath,
	}
}

func (p *Platform) Name() string {
	return TopologyName
}

func (p *Platform) Type() string {
	return TopologyName
}

func (p *Platform) GenerateConfigTree() {
}

func (p *Platform) GenerateArtifacts() {
	fscTopology := p.Context.TopologyByName(fsc.TopologyName).(*fsc.Topology)

	// Generate public parameters
	p.GeneratePublicParameters()

	// Generate crypto material
	for _, node := range fscTopology.Nodes {
		p.GenerateCryptoMaterial(node)
	}

	// Generate fsc configuration extension
	for _, node := range fscTopology.Nodes {
		p.GenerateExtension(node)
	}

	// Prepare chaincodes
	for _, tms := range p.Topology.TMSs {
		var chaincode *topology.ChannelChaincode

		if tms.TokenChaincode.Private {
			cc := p.Fabric(tms).Topology().AddFPC(
				tms.Namespace,
				tms.TokenChaincode.DockerImage,
				tms.TokenChaincode.Orgs...,
			)
			cc.Chaincode.Ctor = p.TCCCtor(tms)
			chaincode = cc
		} else {
			chaincode, _ = p.PrepareTCC(tms)
			p.Fabric(tms).Topology().AppendChaincode(chaincode)
		}

		p.TCCs = append(p.TCCs, &TCC{
			TMS:       tms,
			Chaincode: chaincode,
		})
	}
}

func (p *Platform) Load() {
}

func (p *Platform) Members() []grouper.Member {
	return nil
}

func (p *Platform) PostRun() {
	// Install Token Chaincodes
	p.DeployTokenChaincodes()
}

func (p *Platform) Cleanup() {
}

func (p *Platform) SetPublicParamsGenerator(name string, gen PublicParamsGenerator) {
	p.PublicParamsGenerators[name] = gen
}

func (p *Platform) GenerateExtension(node *sfcnode.Node) {
	t, err := template.New("peer").Funcs(template.FuncMap{
		"TMSs":        func() []*TMS { return p.Topology.TMSs },
		"NodeKVSPath": func() string { return p.FSCNodeKVSDir(node) },
		"Wallets":     func() *Wallet { return p.Wallets[node.Name] },
	}).Parse(Extension)
	Expect(err).NotTo(HaveOccurred())

	ext := bytes.NewBufferString("")
	err = t.Execute(io.MultiWriter(ext), p)
	Expect(err).NotTo(HaveOccurred())

	p.Context.AddExtension(node.Name, TopologyName, ext.String())
}

func (p *Platform) GenerateCryptoMaterial(node *sfcnode.Node) {
	o := node.PlatformOpts()
	opts := options(o)

	p.Wallets[node.Name] = &Wallet{
		Certifiers: []Identity{},
	}

	if opts.Certifier() {
		for _, tms := range p.Topology.TMSs {
			for _, certifier := range tms.Certifiers {
				if certifier == node.Name {
					sess, err := p.TokenGen(commands.CertifierKeygen{
						Driver: tms.Driver,
						PPPath: p.PublicParametersFile(tms),
						Output: p.FSCCertifierCryptoMaterialDir(tms, node),
					})
					Expect(err).NotTo(HaveOccurred())
					Eventually(sess, p.EventuallyTimeout).Should(Exit(0))
					p.Wallets[node.Name].Certifiers = append(p.Wallets[node.Name].Certifiers, Identity{
						ID:      node.Name,
						MSPType: "certifier",
						MSPID:   "certifier",
						Path:    p.FSCCertifierCryptoMaterialDir(tms, node),
					})
				}
			}
		}
	}
}

func (p *Platform) DeployTokenChaincodes() {
	// for _, tcc := range p.TCCs {
	// 	p.Fabric(tcc.TMS).DeployChaincode(tcc.Chaincode)
	// }
}

func (p *Platform) TokenGen(command common.Command) (*Session, error) {
	cmd := common.NewCommand(p.Builder.Build(p.TokenGenPath), command)
	return p.StartSession(cmd, command.SessionName())
}

func (p *Platform) TokenChaincodeServerAddr(port uint16) string {
	return fmt.Sprintf("127.0.0.1:%d", port)
}

func (p *Platform) StartSession(cmd *exec.Cmd, name string) (*Session, error) {
	ansiColorCode := p.nextColor()
	fmt.Fprintf(
		ginkgo.GinkgoWriter,
		"\x1b[33m[d]\x1b[%s[%s]\x1b[0m starting %s %s\n",
		ansiColorCode,
		name,
		filepath.Base(cmd.Args[0]),
		strings.Join(cmd.Args[1:], " "),
	)
	return Start(
		cmd,
		NewPrefixedWriter(
			fmt.Sprintf("\x1b[32m[o]\x1b[%s[%s]\x1b[0m ", ansiColorCode, name),
			ginkgo.GinkgoWriter,
		),
		NewPrefixedWriter(
			fmt.Sprintf("\x1b[91m[e]\x1b[%s[%s]\x1b[0m ", ansiColorCode, name),
			ginkgo.GinkgoWriter,
		),
	)
}

func (p *Platform) GeneratePublicParameters() {
	for _, tms := range p.Topology.TMSs {
		var ppRaw []byte
		generator, ok := p.PublicParamsGenerators[tms.Driver]
		if !ok {
			panic(fmt.Sprintf("driver [%s] not recognized", tms.Driver))
		}
		ppRaw, err := generator.Generate(p, tms)
		Expect(err).ToNot(HaveOccurred())
		// Store pp
		Expect(os.MkdirAll(p.PublicParametersDir(), 0766)).ToNot(HaveOccurred())
		Expect(ioutil.WriteFile(p.PublicParametersFile(tms), ppRaw, 0766)).ToNot(HaveOccurred())
	}
}

func (p *Platform) FSCNodeKVSDir(peer *sfcnode.Node) string {
	return filepath.Join(p.Context.RootDir(), "fscnodes", peer.ID(), "kvs")
}

func (p *Platform) FSCCertifierCryptoMaterialDir(tms *TMS, peer *sfcnode.Node) string {
	return filepath.Join(
		p.Context.RootDir(),
		"crypto",
		"fsc",
		peer.ID(),
		"wallets",
		"certifier",
		fmt.Sprintf("%s_%s_%s_%s", tms.Network, tms.Channel, tms.Namespace, tms.Driver),
	)
}

func (p *Platform) PublicParametersDir() string {
	return filepath.Join(
		p.Context.RootDir(),
		"token",
		"crypto",
		"pp",
	)
}

func (p *Platform) PublicParametersFile(tms *TMS) string {
	return filepath.Join(
		p.Context.RootDir(),
		"token",
		"crypto",
		"pp",
		fmt.Sprintf("%s_%s_%s_%s", tms.Network, tms.Channel, tms.Namespace, tms.Driver),
	)
}

func (p *Platform) nextColor() string {
	color := p.colorIndex%14 + 31
	if color > 37 {
		color = color + 90 - 37
	}

	p.colorIndex++
	return fmt.Sprintf("%dm", color)
}

func (p *Platform) Fabric(tms *TMS) FabricNetwork {
	return p.Context.PlatformByName(tms.Network).(FabricNetwork)
}
