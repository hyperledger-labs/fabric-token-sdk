/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"bytes"
	"fmt"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
	"io"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	math3 "github.com/IBM/mathlib"
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

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators"
	topology2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
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
	InvokeChaincode(cc *topology.ChannelChaincode, method string, args ...[]byte) []byte
	DefaultIdemixOrgMSPDir() string
	Topology() *topology.Topology
	PeerChaincodeAddress(peerName string) string
}

type NetworkHandler interface {
	GenerateArtifacts(tms *topology2.TMS)
	GenerateExtension(tms *topology2.TMS, node *sfcnode.Node) string
	PostRun(load bool, tms *topology2.TMS)
}

type TCC struct {
	TMS       *topology2.TMS
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
	PublicParamsGenerators map[string]generators.PublicParamsGenerator
	NetworkHandlers        map[string]NetworkHandler

	TCCs                     []*TCC
	TokenChaincodePath       string
	TokenGenPath             string
	colorIndex               int
	CryptoMaterialGenerators map[string]generators.CryptoMaterialGenerator
}

func NewPlatform(ctx api2.Context, t api2.Topology, builder api2.Builder) *Platform {
	curveID := math3.BN254
	p := &Platform{
		Context:                  ctx,
		Topology:                 t.(*Topology),
		Builder:                  builder,
		EventuallyTimeout:        10 * time.Minute,
		TCCs:                     []*TCC{},
		PublicParamsGenerators:   map[string]generators.PublicParamsGenerator{},
		CryptoMaterialGenerators: map[string]generators.CryptoMaterialGenerator{},
		TokenChaincodePath:       DefaultTokenChaincode,
		TokenGenPath:             DefaultTokenGenPath,
		NetworkHandlers:          map[string]NetworkHandler{},
	}
	p.PublicParamsGenerators["fabtoken"] = fabric.NewFabTokenPublicParamsGenerator()
	p.PublicParamsGenerators["dlog"] = fabric.NewDLogPublicParamsGenerator(curveID)
	p.CryptoMaterialGenerators["fabtoken"] = fabric.NewFabTokenFabricCryptoMaterialGenerator(p)
	p.CryptoMaterialGenerators["dlog"] = fabric.NewDLogCustomCryptoMaterialGenerator(p, curveID)

	return p
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
	// loop over TMS and generate artifacts
	for _, tms := range p.Topology.TMSs {
		// get the network handler for this TMS
		nh := p.NetworkHandlers[p.Context.TopologyByName(tms.Network).Type()]
		// generate artifacts
		nh.GenerateArtifacts(tms)
	}

	// Generate fsc configuration extension
	fscTopology := p.Context.TopologyByName(fsc.TopologyName).(*fsc.Topology)
	for _, node := range fscTopology.Nodes {
		p.GenerateExtension(node)

		for _, tms := range p.Topology.TMSs {
			// get the network handler for this TMS
			nh := p.NetworkHandlers[p.Context.TopologyByName(tms.Network).Type()]
			// generate artifacts
			ext := nh.GenerateExtension(tms, node)
			p.Context.AddExtension(node.Name, TopologyName, ext)
		}
	}
}

func (p *Platform) Load() {
}

func (p *Platform) Members() []grouper.Member {
	return nil
}

func (p *Platform) PostRun(load bool) {
	// loop over TMS and generate artifacts
	for _, tms := range p.Topology.TMSs {
		// get the network handler for this TMS
		targetNetwork := p.NetworkHandlers[p.Context.TopologyByName(tms.Network).Type()]
		// generate artifacts
		targetNetwork.PostRun(load, tms)
	}
}

func (p *Platform) Cleanup() {
}

func (p *Platform) GetContext() api2.Context {
	return p.Context
}

func (p *Platform) AddNetworkHandler(label string, nh NetworkHandler) {
	p.NetworkHandlers[label] = nh
}

func (p *Platform) GetPublicParamsGenerators(driver string) generators.PublicParamsGenerator {
	return p.PublicParamsGenerators[driver]
}

func (p *Platform) GetCryptoMaterialGenerator(driver string) generators.CryptoMaterialGenerator {
	return p.CryptoMaterialGenerators[driver]
}

func (p *Platform) SetPublicParamsGenerator(name string, gen generators.PublicParamsGenerator) {
	p.PublicParamsGenerators[name] = gen
}

func (p *Platform) SetCryptoMaterialGenerator(name string, gen generators.CryptoMaterialGenerator) {
	p.CryptoMaterialGenerators[name] = gen
}

func (p *Platform) GenerateExtension(node *sfcnode.Node) {
	t, err := template.New("peer").Funcs(template.FuncMap{
		"NodeKVSPath": func() string { return p.FSCNodeKVSDir(node) },
	}).Parse(Extension)
	Expect(err).NotTo(HaveOccurred())

	ext := bytes.NewBufferString("")
	err = t.Execute(io.MultiWriter(ext), p)
	Expect(err).NotTo(HaveOccurred())

	p.Context.AddExtension(node.Name, TopologyName, ext.String())
}

func (p *Platform) GetBuilder() api2.Builder {
	return p.Builder
}

func (p *Platform) TokenGen(command common.Command) (*Session, error) {
	cmd := common.NewCommand(p.Builder.Build(p.TokenGenPath), command)
	return p.StartSession(cmd, command.SessionName())
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

func (p *Platform) FSCNodeKVSDir(peer *sfcnode.Node) string {
	return filepath.Join(p.Context.RootDir(), "fsc", "nodes", peer.ID(), "kvs")
}

func (p *Platform) TokenDir() string {
	return filepath.Join(
		p.Context.RootDir(),
		"token",
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

func (p *Platform) PublicParametersFile(tms *topology2.TMS) string {
	return filepath.Join(
		p.Context.RootDir(),
		"token",
		"crypto",
		"pp",
		fmt.Sprintf("%s_%s_%s_%s", tms.Network, tms.Channel, tms.Namespace, tms.Driver),
	)
}

func (p *Platform) PublicParameters(tms *topology2.TMS) []byte {
	raw, err := ioutil.ReadFile(p.PublicParametersFile(tms))
	Expect(err).ToNot(HaveOccurred())
	return raw
}

func (p *Platform) nextColor() string {
	color := p.colorIndex%14 + 31
	if color > 37 {
		color = color + 90 - 37
	}

	p.colorIndex++
	return fmt.Sprintf("%dm", color)
}
