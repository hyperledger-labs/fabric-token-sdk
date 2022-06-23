/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package generators

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/commands"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const (
	DefaultCryptoTemplate = `---
{{ with $w := . -}}
PeerOrgs:{{ range .PeerOrgs }}
- Name: {{ .Name }}
  Domain: {{ .Domain }}
  EnableNodeOUs: {{ .EnableNodeOUs }}
  {{- if .CA }}
  CA:{{ if .CA.Hostname }}
    hostname: {{ .CA.Hostname }}
    SANS:
    - localhost
    - 127.0.0.1
    - ::1
  {{- end }}
  {{- end }}
  Users:
    Count: {{ .Users }}
    {{- if len .UserNames }}
    Names: {{ range .UserNames }}
    - {{ . }}
    {{- end }}
    {{- end }}

  Specs:{{ range $w.PeersInOrg .Name }}
  - Hostname: {{ .Name }}
    SANS:
    - localhost
    - 127.0.0.1
    - ::1
  {{- end }}
{{- end }}
{{- end }}
`
)

type Organization struct {
	ID            string
	MSPID         string
	MSPType       string
	Name          string
	Domain        string
	EnableNodeOUs bool
	Users         int
	UserNames     []string
	CA            *CA
}

type CA struct {
	Hostname string `yaml:"hostname,omitempty"`
}

type Peer struct {
	Name         string
	Organization string
}

type Layout struct {
	Orgs  []Organization
	Peers []Peer
}

func (l *Layout) PeerOrgs() []Organization {
	return l.Orgs
}

func (l *Layout) PeersInOrg(orgName string) []Peer {
	var peers []Peer
	for _, o := range l.Peers {
		if o.Organization == orgName {
			peers = append(peers, o)
		}
	}
	return peers
}

type FSCPlatform interface {
	PeerOrgs() []*node.Organization
}

type FabTokenFabricCryptoMaterialGenerator struct {
	TokenPlatform     TokenPlatform
	EventuallyTimeout time.Duration
	colorIndex        int
	Builder           *Builder
}

func NewFabTokenFabricCryptoMaterialGenerator(tokenPlatform TokenPlatform, builder api.Builder) *FabTokenFabricCryptoMaterialGenerator {
	return &FabTokenFabricCryptoMaterialGenerator{
		TokenPlatform:     tokenPlatform,
		EventuallyTimeout: 10 * time.Minute,
		Builder:           &Builder{client: builder},
	}
}

func (d *FabTokenFabricCryptoMaterialGenerator) Setup(tms *topology.TMS) (string, error) {
	return "", nil
}

func (d *FabTokenFabricCryptoMaterialGenerator) GenerateCertifierIdentities(tms *topology.TMS, n *node.Node, certifiers ...string) []Identity {
	return d.generate(tms, n, "certifiers", certifiers...)
}

func (d *FabTokenFabricCryptoMaterialGenerator) GenerateOwnerIdentities(tms *topology.TMS, n *node.Node, owners ...string) []Identity {
	return d.generate(tms, n, "owners", owners...)
}

func (d *FabTokenFabricCryptoMaterialGenerator) GenerateIssuerIdentities(tms *topology.TMS, n *node.Node, issuers ...string) []Identity {
	return d.generate(tms, n, "issuers", issuers...)
}

func (d *FabTokenFabricCryptoMaterialGenerator) GenerateAuditorIdentities(tms *topology.TMS, n *node.Node, auditors ...string) []Identity {
	return d.generate(tms, n, "auditors", auditors...)
}

func (d *FabTokenFabricCryptoMaterialGenerator) generate(tms *topology.TMS, n *node.Node, typ string, names ...string) []Identity {
	output := filepath.Join(d.TokenPlatform.TokenDir(), "crypto", tms.ID(), n.ID(), typ)
	orgName := fmt.Sprintf("Org%s", n.ID())
	mspID := fmt.Sprintf("%sMSP", orgName)
	domain := fmt.Sprintf("%s.example.com", orgName)
	l := &Layout{
		Orgs: []Organization{
			{
				ID:            orgName,
				MSPID:         mspID,
				MSPType:       "bccsp",
				Name:          orgName,
				Domain:        domain,
				EnableNodeOUs: false,
				Users:         1,
			},
		},
	}
	for _, name := range names {
		l.Peers = append(l.Peers, Peer{
			Name:         name,
			Organization: orgName,
		})
	}
	d.GenerateCryptoConfig(output, l)
	d.GenerateArtifacts(output)

	var identities []Identity
	for _, name := range names {
		identities = append(identities, Identity{
			ID:   name,
			Type: "bccsp:" + mspID,
			Path: filepath.Join(
				output,
				"peerOrganizations",
				domain,
				"peers",
				fmt.Sprintf("%s.%s", name, domain),
				"msp"),
		})
	}
	return identities

}

func (d *FabTokenFabricCryptoMaterialGenerator) GenerateCryptoConfig(output string, layout *Layout) {
	Expect(os.MkdirAll(output, 0770)).NotTo(HaveOccurred())
	crypto, err := os.Create(filepath.Join(output, "crypto-config.yaml"))
	Expect(err).NotTo(HaveOccurred())
	defer crypto.Close()

	t, err := template.New("crypto").Parse(DefaultCryptoTemplate)
	Expect(err).NotTo(HaveOccurred())

	err = t.Execute(io.MultiWriter(crypto), layout)
	Expect(err).NotTo(HaveOccurred())
}

func (d *FabTokenFabricCryptoMaterialGenerator) GenerateArtifacts(output string) {
	sess, err := d.Cryptogen(commands.Generate{
		Config: filepath.Join(output, "crypto-config.yaml"),
		Output: output,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, d.EventuallyTimeout).Should(gexec.Exit(0))
}

func (d *FabTokenFabricCryptoMaterialGenerator) Cryptogen(command common.Command) (*gexec.Session, error) {
	cmd := common.NewCommand(d.Builder.FSCCLI(), command)
	return d.StartSession(cmd, command.SessionName())
}

func (d *FabTokenFabricCryptoMaterialGenerator) StartSession(cmd *exec.Cmd, name string) (*gexec.Session, error) {
	ansiColorCode := d.NextColor()
	fmt.Fprintf(
		ginkgo.GinkgoWriter,
		"\x1b[33m[d]\x1b[%s[%s]\x1b[0m starting %s %s\n",
		ansiColorCode,
		name,
		filepath.Base(cmd.Args[0]),
		strings.Join(cmd.Args[1:], " "),
	)
	return gexec.Start(
		cmd,
		gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[32m[o]\x1b[%s[%s]\x1b[0m ", ansiColorCode, name),
			ginkgo.GinkgoWriter,
		),
		gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[91m[e]\x1b[%s[%s]\x1b[0m ", ansiColorCode, name),
			ginkgo.GinkgoWriter,
		),
	)
}

func (d *FabTokenFabricCryptoMaterialGenerator) NextColor() string {
	color := d.colorIndex%14 + 31
	if color > 37 {
		color = color + 90 - 37
	}

	d.colorIndex++
	return fmt.Sprintf("%dm", color)
}
