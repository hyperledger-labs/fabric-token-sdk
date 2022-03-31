/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric/commands"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators"

	math3 "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type DLogFabricCryptoMaterialGenerator struct {
	tokenPlatform tokenPlatform
	curveID       math3.CurveID
}

func NewDLogFabricCryptoMaterialGenerator(tokenPlatform tokenPlatform) *DLogFabricCryptoMaterialGenerator {
	return &DLogFabricCryptoMaterialGenerator{tokenPlatform: tokenPlatform, curveID: math3.FP256BN_AMCL}
}

func NewDLogFabricCryptoMaterialGeneratorWithCurve(tokenPlatform tokenPlatform, curveID math3.CurveID) *DLogFabricCryptoMaterialGenerator {
	return &DLogFabricCryptoMaterialGenerator{tokenPlatform: tokenPlatform, curveID: curveID}
}

func (d *DLogFabricCryptoMaterialGenerator) Setup(tms *topology.TMS) (string, error) {
	return d.tokenPlatform.GetContext().PlatformByName(tms.Network).(fabricPlatform).DefaultIdemixOrgMSPDir(), nil
}

func (d *DLogFabricCryptoMaterialGenerator) GenerateCertifierIdentities(tms *topology.TMS, node *node.Node, certifiers ...string) []generators.Identity {
	return nil
}

func (d *DLogFabricCryptoMaterialGenerator) GenerateOwnerIdentities(tms *topology.TMS, n *node.Node, owners ...string) []generators.Identity {
	fp := d.tokenPlatform.GetContext().PlatformByName(tms.Network).(fabricPlatform)
	peer := fp.PeersByID(n.ID())
	if peer == nil {
		// This peer is not in that fabric network
		return nil
	}

	var res []generators.Identity
	for _, owner := range owners {
		found := false
		for _, identity := range peer.Identities {
			if identity.Type != "idemix" {
				continue
			}

			if identity.ID == owner || (identity.ID == "idemix" && owner == n.ID()) {
				res = append(res, generators.Identity{
					ID:   owner,
					Type: identity.Type + ":" + identity.MSPID + ":" + CurveIDToString(d.curveID),
					Path: identity.Path,
				})
				found = true
				break
			}
		}
		Expect(found).To(BeTrue())
	}

	return res
}

func (d *DLogFabricCryptoMaterialGenerator) GenerateIssuerIdentities(tms *topology.TMS, n *node.Node, issuers ...string) []generators.Identity {
	fp := d.tokenPlatform.GetContext().PlatformByName(tms.Network).(fabricPlatform)
	peer := fp.PeersByID(n.ID())
	if peer == nil {
		// This peer is not in that fabric network
		return nil
	}

	var res []generators.Identity
	for _, issuer := range issuers {
		found := false
		for _, identity := range peer.Identities {
			if identity.ID == issuer && identity.Type == "bccsp" {
				res = append(res, generators.Identity{
					ID:   issuer,
					Type: identity.Type + ":" + identity.MSPID,
					Path: identity.Path,
				})
				found = true
			}
		}
		Expect(found).To(BeTrue())
	}
	return res
}

func (d *DLogFabricCryptoMaterialGenerator) GenerateAuditorIdentities(tms *topology.TMS, n *node.Node, auditors ...string) []generators.Identity {
	fp := d.tokenPlatform.GetContext().PlatformByName(tms.Network).(fabricPlatform)
	peer := fp.PeersByID(n.ID())
	if peer == nil {
		// This peer is not in that fabric network
		return nil
	}

	var res []generators.Identity
	for _, auditor := range auditors {
		found := false
		for _, identity := range peer.Identities {
			if identity.ID == auditor && identity.Type == "bccsp" {
				res = append(res, generators.Identity{
					ID:   auditor,
					Type: identity.Type + ":" + identity.MSPID,
					Path: identity.Path,
				})
				found = true
			}
		}
		Expect(found).To(BeTrue())
	}
	return res
}

type DLogCustomCryptoMaterialGenerator struct {
	*DLogFabricCryptoMaterialGenerator

	TokenPlatform     tokenPlatform
	Curve             string
	EventuallyTimeout time.Duration

	colorIndex             int
	revocationHandlerIndex int
}

func NewDLogCustomCryptoMaterialGenerator(tokenPlatform tokenPlatform, curveID math3.CurveID) *DLogCustomCryptoMaterialGenerator {
	return &DLogCustomCryptoMaterialGenerator{
		DLogFabricCryptoMaterialGenerator: NewDLogFabricCryptoMaterialGeneratorWithCurve(tokenPlatform, curveID),
		TokenPlatform:                     tokenPlatform,
		EventuallyTimeout:                 10 * time.Minute,
		Curve:                             CurveIDToString(curveID),
	}
}

func (d *DLogCustomCryptoMaterialGenerator) Setup(tms *topology.TMS) (string, error) {
	output := filepath.Join(d.TokenPlatform.TokenDir(), "crypto", tms.ID(), "idemix")
	if err := os.MkdirAll(output, 0766); err != nil {
		return "", err
	}
	sess, err := d.Idemixgen(commands.CAKeyGen{
		NetworkPrefix: tms.ID(),
		Output:        output,
		Curve:         d.Curve,
	})
	if err != nil {
		return "", err
	}
	Eventually(sess, d.EventuallyTimeout).Should(gexec.Exit(0))
	return output, nil
	// return d.tokenPlatform.GetContext().PlatformByName(tms.Network).(fabricPlatform).DefaultIdemixOrgMSPDir(), nil
}

func (d *DLogCustomCryptoMaterialGenerator) GenerateCertifierIdentities(tms *topology.TMS, node *node.Node, certifiers ...string) []generators.Identity {
	return d.DLogFabricCryptoMaterialGenerator.GenerateCertifierIdentities(tms, node, certifiers...)
}

func (d *DLogCustomCryptoMaterialGenerator) GenerateOwnerIdentities(tms *topology.TMS, n *node.Node, owners ...string) []generators.Identity {
	var res []generators.Identity
	tmsID := tms.ID()
	for i, owner := range owners {
		logger.Debugf("Generating owner identity [%s] for [%s]", owner, tmsID)
		userOutput := filepath.Join(d.TokenPlatform.TokenDir(), "crypto", tmsID, "idemix", owner)
		if err := os.MkdirAll(userOutput, 0766); err != nil {
			return nil
		}
		sess, err := d.Idemixgen(commands.SignerConfig{
			NetworkPrefix: tmsID,
			CAInput:       filepath.Join(d.TokenPlatform.TokenDir(), "crypto", tmsID, "idemix"),
			// CAInput:          d.tokenPlatform.GetContext().PlatformByName(tms.Network).(fabricPlatform).DefaultIdemixOrgMSPDir(),
			Output:           userOutput,
			OrgUnit:          tmsID + ".example.com",
			EnrollmentID:     owner,
			RevocationHandle: fmt.Sprintf("1%d%d", d.revocationHandlerIndex, i),
			Curve:            d.Curve,
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, d.EventuallyTimeout).Should(gexec.Exit(0))

		res = append(res, generators.Identity{
			ID:   owner,
			Type: "idemix:" + "IdemixOrgMSP" + ":" + CurveIDToString(d.curveID),
			Path: userOutput,
		})
	}
	d.revocationHandlerIndex++
	return res
}

func (d *DLogCustomCryptoMaterialGenerator) GenerateIssuerIdentities(tms *topology.TMS, n *node.Node, issuers ...string) []generators.Identity {
	return d.DLogFabricCryptoMaterialGenerator.GenerateIssuerIdentities(tms, n, issuers...)
}

func (d *DLogCustomCryptoMaterialGenerator) GenerateAuditorIdentities(tms *topology.TMS, n *node.Node, auditors ...string) []generators.Identity {
	return d.DLogFabricCryptoMaterialGenerator.GenerateAuditorIdentities(tms, n, auditors...)
}

func (d *DLogCustomCryptoMaterialGenerator) Idemixgen(command common.Command) (*gexec.Session, error) {
	cmd := common.NewCommand(d.TokenPlatform.GetBuilder().Build("github.com/IBM/idemix/tools/idemixgen"), command)
	return d.StartSession(cmd, command.SessionName())
}

func (d *DLogCustomCryptoMaterialGenerator) StartSession(cmd *exec.Cmd, name string) (*gexec.Session, error) {
	ansiColorCode := d.nextColor()
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

func (d *DLogCustomCryptoMaterialGenerator) nextColor() string {
	color := d.colorIndex%14 + 31
	if color > 37 {
		color = color + 90 - 37
	}

	d.colorIndex++
	return fmt.Sprintf("%dm", color)
}

func CurveIDToString(id math3.CurveID) string {
	switch id {
	case math3.BN254:
		return "BN254"
	case math3.FP256BN_AMCL:
		return "FP256BN_AMCL"
	case math3.FP256BN_AMCL_MIRACL:
		return "FP256BN_AMCL_MIRACL"
	default:
		panic("invalid curve id")
	}
}
