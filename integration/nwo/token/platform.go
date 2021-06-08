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
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/template"
	"github.com/hyperledger/fabric/msp"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common/registry"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/topology"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	sfcnode "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"

	cryptofabtoken "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	cryptodlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
)

type Builder interface {
	Build(path string) string
}

type FabricNetwork interface {
	DeployChaincode(chaincode *topology.ChannelChaincode)
	DefaultIdemixOrgMSPDir() string
	Topology() *topology.Topology
	PeerChaincodeAddress(peerName string) string
}

type TCC struct {
	TMS       *TMS
	Chaincode *topology.ChannelChaincode
	Port      uint16
	Processes []ifrit.Process
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

type platform struct {
	Registry          *registry.Registry
	Topology          *Topology
	FabricNetwork     FabricNetwork
	EventuallyTimeout time.Duration
	Wallets           map[string]*Wallet
	TCCs              []*TCC

	colorIndex int
}

func NewPlatform(registry *registry.Registry) *platform {
	return &platform{
		Registry:          registry,
		EventuallyTimeout: 10 * time.Minute,
		Wallets:           map[string]*Wallet{},
		TCCs:              []*TCC{},
	}
}

func (p *platform) Name() string {
	return TopologyName
}

func (p *platform) GenerateConfigTree() {
	p.Topology = p.Registry.TopologyByName(TopologyName).(*Topology)
	p.FabricNetwork = p.Registry.PlatformByName(fabric.TopologyName).(FabricNetwork)
}

func (p *platform) GenerateArtifacts() {
	fscTopology := p.Registry.TopologyByName(fsc.TopologyName).(*fsc.Topology)

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
		chaincode, port := p.PrepareTCC(tms)

		p.TCCs = append(p.TCCs, &TCC{
			TMS:       tms,
			Chaincode: chaincode,
			Port:      port,
		})
	}
}

func (p *platform) Load() {
	p.Topology = p.Registry.TopologyByName(TopologyName).(*Topology)
	p.FabricNetwork = p.Registry.PlatformByName(fabric.TopologyName).(FabricNetwork)
}

func (p *platform) Members() []grouper.Member {
	return nil
}

func (p *platform) PostRun() {
	// Install Token Chaincodes
	p.DeployTokenChaincodes()
}

func (p *platform) Cleanup() {
	for _, tcc := range p.TCCs {
		for _, process := range tcc.Processes {
			process.Signal(syscall.SIGKILL)
		}
	}
}

func (p *platform) GenerateExtension(node *sfcnode.Node) {
	t, err := template.New("peer").Funcs(template.FuncMap{
		"TMSs":        func() []*TMS { return p.Topology.TMSs },
		"NodeKVSPath": func() string { return p.NodeKVSDir(node) },
		"Wallets":     func() *Wallet { return p.Wallets[node.Name] },
	}).Parse(Extension)
	Expect(err).NotTo(HaveOccurred())

	ext := bytes.NewBufferString("")
	err = t.Execute(io.MultiWriter(ext), p)
	Expect(err).NotTo(HaveOccurred())

	p.Registry.AddExtension(node.Name, TopologyName, ext.String())
}

func (p *platform) GenerateCryptoMaterial(node *sfcnode.Node) {
	o := node.PlatformOpts()
	opts := options(o)

	p.Wallets[node.Name] = &Wallet{
		Certifiers: []Identity{},
	}

	if opts.Certifier() {
		// for _, tms := range p.Topology.TMSs {
		// 	for _, certifier := range tms.Certifiers {
		// 		if certifier == node.Name {
		// 			sess, err := p.TokenGen(commands.CertifierKeygen{
		// 				Driver: tms.Driver,
		// 				PPPath: p.PublicParametersFile(tms),
		// 				Output: p.CertifierCryptoMaterialDir(tms, node),
		// 			})
		// 			Expect(err).NotTo(HaveOccurred())
		// 			Eventually(sess, p.EventuallyTimeout).Should(Exit(0))
		// 			p.Wallets[node.Name].Certifiers = append(p.Wallets[node.Name].Certifiers, Identity{
		// 				ID:      node.Name,
		// 				MSPType: "certifier",
		// 				MSPID:   "certifier",
		// 				Path:    p.CertifierCryptoMaterialDir(tms, node),
		// 			})
		// 		}
		// 	}
		// }
	}
}

func (p *platform) DeployTokenChaincodes() {
	for _, tcc := range p.TCCs {
		p.FabricNetwork.DeployChaincode(tcc.Chaincode)
	}
}

func (p *platform) TokenGen(command common.Command) (*Session, error) {
	cmd := common.NewCommand(p.Registry.Builder.Build("github.com/hyperledger-labs/fabric-token0sdk/cmd/tokengen"), command)
	return p.StartSession(cmd, command.SessionName())
}

func (p *platform) TokenChaincodeServerAddr(port uint16) string {
	return fmt.Sprintf("127.0.0.1:%d", port)
}

func (p *platform) StartSession(cmd *exec.Cmd, name string) (*Session, error) {
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

func (p *platform) GeneratePublicParameters() {
	// TODO: This should be generated in the context of the token-topology
	path := filepath.Join(p.FabricNetwork.DefaultIdemixOrgMSPDir(), msp.IdemixConfigDirMsp, msp.IdemixConfigFileIssuerPublicKey)
	ipkBytes, err := ioutil.ReadFile(path)
	Expect(err).ToNot(HaveOccurred())

	for _, tms := range p.Topology.TMSs {
		var ppRaw []byte
		switch tms.Driver {
		case "dlog":
			base, err := strconv.ParseInt(tms.TokenChaincode.PublicParamsGenArgs[0], 10, 64)
			Expect(err).ToNot(HaveOccurred())
			exp, err := strconv.ParseInt(tms.TokenChaincode.PublicParamsGenArgs[1], 10, 32)
			Expect(err).ToNot(HaveOccurred())
			pp, err := cryptodlog.Setup(
				base,
				int(exp),
				ipkBytes,
			)
			Expect(err).ToNot(HaveOccurred())

			ppRaw, err = pp.Serialize()
			Expect(err).ToNot(HaveOccurred())
		case "fabtoken":
			// setup
			pp, err := cryptofabtoken.Setup()
			Expect(err).ToNot(HaveOccurred())
			ppRaw, err = pp.Serialize()
			Expect(err).ToNot(HaveOccurred())
		default:
			panic(fmt.Sprintf("driver [%s] not recognized", tms.Driver))
		}
		// Store pp
		Expect(os.MkdirAll(p.PublicParametersDir(), 0766)).ToNot(HaveOccurred())
		Expect(ioutil.WriteFile(p.PublicParametersFile(tms), ppRaw, 0766)).ToNot(HaveOccurred())
	}
}

func (p *platform) NodeKVSDir(peer *sfcnode.Node) string {
	return filepath.Join(p.Registry.RootDir, "fscnodes", peer.ID(), "kvs")
}

func (p *platform) CertifierCryptoMaterialDir(tms *TMS, peer *sfcnode.Node) string {
	return filepath.Join(
		p.Registry.RootDir,
		"crypto",
		"fsc",
		peer.ID(),
		"wallets",
		"certifier",
		fmt.Sprintf("%s_%s_%s", tms.Channel, tms.Namespace, tms.Driver),
	)
}

func (p *platform) PublicParametersDir() string {
	return filepath.Join(
		p.Registry.RootDir,
		"crypto",
		"token-sdk",
		"pp",
	)
}

func (p *platform) PublicParametersFile(tms *TMS) string {
	return filepath.Join(
		p.Registry.RootDir,
		"crypto",
		"token-sdk",
		"pp",
		fmt.Sprintf("%s_%s_%s.pp", tms.Channel, tms.Namespace, tms.Driver),
	)
}

func (p *platform) nextColor() string {
	color := p.colorIndex%14 + 31
	if color > 37 {
		color = color + 90 - 37
	}

	p.colorIndex++
	return fmt.Sprintf("%dm", color)
}
