/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

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
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/network"
	ftopology "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/topology"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/components"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/onsi/ginkgo/v2"
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
    {{- if len .UserSpecs }}
    Specs: {{ range .UserSpecs }}
    - Name: {{ .Name }}
      HSM: {{ .HSM }}
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

var logger = flogging.MustGetLogger("token-sdk.integration.token.generators.fabtoken")

type Peer struct {
	Name         string
	Organization string
}

type Layout struct {
	Orgs  []ftopology.Organization
	Peers []Peer
}

func (l *Layout) PeerOrgs() []ftopology.Organization {
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

type CryptoMaterialGenerator struct {
	TokenPlatform     generators.TokenPlatform
	EventuallyTimeout time.Duration
	ColorIndex        int
	Builder           *components.Builder
}

func NewCryptoMaterialGenerator(tokenPlatform generators.TokenPlatform, builder api.Builder) *CryptoMaterialGenerator {
	return &CryptoMaterialGenerator{
		TokenPlatform:     tokenPlatform,
		EventuallyTimeout: 10 * time.Minute,
		Builder:           components.NewBuilder(builder),
	}
}

func (d *CryptoMaterialGenerator) Setup(tms *topology.TMS) (string, error) {
	return "", nil
}

func (d *CryptoMaterialGenerator) GenerateCertifierIdentities(tms *topology.TMS, n *node.Node, certifiers ...string) []generators.Identity {
	return d.Generate(tms, n, "certifiers", certifiers...)
}

func (d *CryptoMaterialGenerator) GenerateOwnerIdentities(tms *topology.TMS, n *node.Node, owners ...string) []generators.Identity {
	return d.Generate(tms, n, "owners", owners...)
}

func (d *CryptoMaterialGenerator) GenerateIssuerIdentities(tms *topology.TMS, n *node.Node, issuers ...string) []generators.Identity {
	return d.Generate(tms, n, "issuers", issuers...)
}

func (d *CryptoMaterialGenerator) GenerateAuditorIdentities(tms *topology.TMS, n *node.Node, auditors ...string) []generators.Identity {
	return d.Generate(tms, n, "auditors", auditors...)
}

func (d *CryptoMaterialGenerator) Generate(tms *topology.TMS, n *node.Node, wallet string, names ...string) []generators.Identity {
	logger.Infof("generate [%s] identities [%v]", wallet, names)

	output := filepath.Join(d.TokenPlatform.TokenDir(), "crypto", tms.ID(), n.ID(), wallet)
	orgName := fmt.Sprintf("Org%s", n.ID())
	mspID := fmt.Sprintf("%sMSP", orgName)
	domain := fmt.Sprintf("%s.example.com", orgName)

	var userSpecs []ftopology.UserSpec
	for _, name := range names {
		us := ftopology.UserSpec{
			Name: name,
			HSM:  false,
		}
		switch wallet {
		case "issuers":
			us.HSM = topology.ToOptions(n.Options).IsUseHSMForIssuer(name)
		case "auditors":
			us.HSM = topology.ToOptions(n.Options).IsUseHSMForAuditor()
		}
		userSpecs = append(userSpecs, us)
	}
	l := &Layout{
		Orgs: []ftopology.Organization{
			{
				ID:            orgName,
				MSPID:         mspID,
				MSPType:       "bccsp",
				Name:          orgName,
				Domain:        domain,
				EnableNodeOUs: false,
				Users:         1,
				UserSpecs:     userSpecs,
			},
		},
		Peers: []Peer{
			{Name: orgName, Organization: orgName},
		},
	}
	d.GenerateCryptoConfig(output, l)
	d.GenerateArtifacts(output)

	var identities []generators.Identity
	for i, name := range names {
		idOutput := filepath.Join(
			output,
			"peerOrganizations",
			domain,
			"users",
			fmt.Sprintf("%s@%s", name, domain),
			"msp")

		tokenOpts := topology.ToOptions(n.Options)
		remote := tokenOpts.IsRemoteOwner(name)
		if remote {
			// copy the content of the keystore folder to keystoreFull
			in, err := os.Open(filepath.Join(idOutput, "keystore", "priv_sk"))
			Expect(err).NotTo(HaveOccurred())

			Expect(os.MkdirAll(filepath.Join(idOutput, "keystoreFull"), 0766)).NotTo(HaveOccurred())
			out, err := os.Create(filepath.Join(idOutput, "keystoreFull", "priv_sk"))
			Expect(err).NotTo(HaveOccurred())
			_, err = io.Copy(out, in)
			Expect(err).NotTo(HaveOccurred())
			err = out.Sync()
			Expect(err).NotTo(HaveOccurred())
			in.Close()
			out.Close()

			// delete keystore/priv_sk
			Expect(os.Remove(filepath.Join(idOutput, "keystore", "priv_sk"))).NotTo(HaveOccurred())
		}

		id := generators.Identity{
			ID:   name,
			Path: idOutput,
		}

		if wallet == "issuers" || wallet == "auditors" {
			if userSpecs[i].HSM {
				// PKCS11
				id.Opts = network.BCCSPOpts("PKCS11")
			} else {
				// SW
				id.Opts = network.BCCSPOpts("SW")
			}
		}

		identities = append(identities, id)
	}
	return identities

}

func (d *CryptoMaterialGenerator) GenerateCryptoConfig(output string, layout *Layout) {
	Expect(os.MkdirAll(output, 0770)).NotTo(HaveOccurred())
	crypto, err := os.Create(filepath.Join(output, "crypto-config.yaml"))
	Expect(err).NotTo(HaveOccurred())
	defer crypto.Close()

	t, err := template.New("crypto").Parse(DefaultCryptoTemplate)
	Expect(err).NotTo(HaveOccurred())

	err = t.Execute(io.MultiWriter(crypto), layout)
	Expect(err).NotTo(HaveOccurred())
}

func (d *CryptoMaterialGenerator) GenerateArtifacts(output string) {
	sess, err := d.Cryptogen(commands.Generate{
		Config: filepath.Join(output, "crypto-config.yaml"),
		Output: output,
	})
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess, d.EventuallyTimeout).Should(gexec.Exit(0))
}

func (d *CryptoMaterialGenerator) Cryptogen(command common.Command) (*gexec.Session, error) {
	cmd := common.NewCommand(d.Builder.FSCCLI(), command)
	return d.StartSession(cmd, command.SessionName())
}

func (d *CryptoMaterialGenerator) StartSession(cmd *exec.Cmd, name string) (*gexec.Session, error) {
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

func (d *CryptoMaterialGenerator) NextColor() string {
	color := d.ColorIndex%14 + 31
	if color > 37 {
		color = color + 90 - 37
	}

	d.ColorIndex++
	return fmt.Sprintf("%dm", color)
}
