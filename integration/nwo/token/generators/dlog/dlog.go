/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dlog

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/IBM/idemix"
	math3 "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/commands"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	idemix2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp/idemix"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

// WithAries notify the backend to use aries as crypto provider when possible
func WithAries(tms *topology.TMS) {
	tms.BackendParams["idemix.aries"] = true
}

// IsAries return true if this TMS requires to use aries as crypto provider when possible
func IsAries(tms *topology.TMS) bool {
	ariesBoxed, ok := tms.BackendParams["idemix.aries"]
	if ok {
		return ariesBoxed.(bool)
	}
	return false
}

var logger = flogging.MustGetLogger("token-sdk.integration.token.generators.dlog")

type CryptoMaterialGenerator struct {
	FabTokenGenerator *fabtoken.CryptoMaterialGenerator

	TokenPlatform     generators.TokenPlatform
	DefaultCurve      string
	EventuallyTimeout time.Duration

	ColorIndex             int
	RevocationHandlerIndex int
}

func NewCryptoMaterialGenerator(tokenPlatform generators.TokenPlatform, defaultCurveID math3.CurveID, builder api.Builder) *CryptoMaterialGenerator {
	return &CryptoMaterialGenerator{
		FabTokenGenerator: fabtoken.NewCryptoMaterialGenerator(tokenPlatform, builder),
		TokenPlatform:     tokenPlatform,
		EventuallyTimeout: 10 * time.Minute,
		DefaultCurve:      CurveIDToString(defaultCurveID),
	}
}

func NewCryptoMaterialGeneratorWithCurveIdentifier(tokenPlatform generators.TokenPlatform, curveID string, builder api.Builder) *CryptoMaterialGenerator {
	return &CryptoMaterialGenerator{
		FabTokenGenerator: fabtoken.NewCryptoMaterialGenerator(tokenPlatform, builder),
		TokenPlatform:     tokenPlatform,
		EventuallyTimeout: 10 * time.Minute,
		DefaultCurve:      curveID,
	}
}

func (d *CryptoMaterialGenerator) Setup(tms *topology.TMS) (string, error) {
	output := filepath.Join(d.TokenPlatform.TokenDir(), "crypto", tms.ID(), "idemix")
	if err := os.MkdirAll(output, 0766); err != nil {
		return "", err
	}

	curveID := d.DefaultCurve
	if IsAries(tms) {
		curveID = CurveIDToString(math3.BLS12_381_BBS)
	}

	// notice that if aries is enabled, curve is ignored
	sess, err := d.Idemixgen(commands.CAKeyGen{
		NetworkPrefix: tms.ID(),
		Output:        output,
		Curve:         curveID,
		Aries:         IsAries(tms),
	})
	if err != nil {
		return "", err
	}
	Eventually(sess, d.EventuallyTimeout).Should(gexec.Exit(0))
	return output, nil
}

func (d *CryptoMaterialGenerator) GenerateCertifierIdentities(tms *topology.TMS, node *node.Node, certifiers ...string) []generators.Identity {
	return nil
}

func (d *CryptoMaterialGenerator) GenerateOwnerIdentities(tms *topology.TMS, n *node.Node, owners ...string) []generators.Identity {
	logger.Infof("generate [owners] identities [%v]", owners)

	curveID := d.DefaultCurve
	if IsAries(tms) {
		curveID = CurveIDToString(math3.BLS12_381_BBS)
	}

	var res []generators.Identity
	tmsID := tms.ID()
	for i, owner := range owners {
		pathPrefix := ""
		tokenOpts := topology.ToOptions(n.Options)
		remote := tokenOpts.IsRemoteOwner(owner)
		if remote {
			// prepare a remote owner wallet
			pathPrefix = "remote"
		}

		logger.Debugf("Generating owner identity [%s] for [%s]", owner, tmsID)
		userOutput := filepath.Join(d.TokenPlatform.TokenDir(), "crypto", tmsID, "idemix", pathPrefix, owner)
		if err := os.MkdirAll(userOutput, 0766); err != nil {
			return nil
		}
		// notice that if aries is enabled, curve is ignored
		sess, err := d.Idemixgen(commands.SignerConfig{
			NetworkPrefix: tmsID,
			CAInput:       filepath.Join(d.TokenPlatform.TokenDir(), "crypto", tmsID, "idemix"),
			// CAInput:          d.TokenPlatform.GetContext().PlatformByName(tms.Network).(fabricPlatform).DefaultIdemixOrgMSPDir(),
			Output:           userOutput,
			OrgUnit:          tmsID + ".example.com",
			EnrollmentID:     owner,
			RevocationHandle: fmt.Sprintf("1%d%d", d.RevocationHandlerIndex, i),
			Curve:            curveID,
			Aries:            IsAries(tms),
		})
		Expect(err).NotTo(HaveOccurred())
		Eventually(sess, d.EventuallyTimeout).Should(gexec.Exit(0))

		if remote {
			// Prepare a folder for the remote wallet
			// This is done by stripping out Cred and Sk from the SignerConfig
			signerBytes, err := os.ReadFile(filepath.Join(userOutput, idemix.IdemixConfigDirUser, idemix.IdemixConfigFileSigner))
			Expect(err).NotTo(HaveOccurred())

			// nullify cred and sk
			signerConfig := &msp.IdemixMSPSignerConfig{}
			err = proto.Unmarshal(signerBytes, signerConfig)
			Expect(err).NotTo(HaveOccurred())
			signerConfig.Cred = nil
			signerConfig.Sk = nil

			// save the original signer config to a new file
			err = os.WriteFile(
				filepath.Join(userOutput, idemix.IdemixConfigDirUser, idemix2.SignerConfigFull),
				signerBytes,
				0766,
			)
			Expect(err).NotTo(HaveOccurred())

			// overwrite the signer config file so that the token-sdk will interpreter this wallet as a remote one or verify only wallet
			raw, err := proto.Marshal(signerConfig)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(
				filepath.Join(userOutput, idemix.IdemixConfigDirUser, idemix.IdemixConfigFileSigner),
				raw,
				0766,
			)
			Expect(err).NotTo(HaveOccurred())
		}

		res = append(res, generators.Identity{
			ID:   owner,
			Path: userOutput,
		})
	}
	d.RevocationHandlerIndex++
	return res
}

func (d *CryptoMaterialGenerator) GenerateIssuerIdentities(tms *topology.TMS, n *node.Node, issuers ...string) []generators.Identity {
	return d.FabTokenGenerator.GenerateIssuerIdentities(tms, n, issuers...)
}

func (d *CryptoMaterialGenerator) GenerateAuditorIdentities(tms *topology.TMS, n *node.Node, auditors ...string) []generators.Identity {
	return d.FabTokenGenerator.GenerateAuditorIdentities(tms, n, auditors...)
}

func (d *CryptoMaterialGenerator) Idemixgen(command common.Command) (*gexec.Session, error) {
	cmd := common.NewCommand(d.TokenPlatform.GetBuilder().Build("github.com/IBM/idemix/tools/idemixgen"), command)
	return d.StartSession(cmd, command.SessionName())
}

func (d *CryptoMaterialGenerator) StartSession(cmd *exec.Cmd, name string) (*gexec.Session, error) {
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

func (d *CryptoMaterialGenerator) nextColor() string {
	color := d.ColorIndex%14 + 31
	if color > 37 {
		color = color + 90 - 37
	}

	d.ColorIndex++
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
	case math3.BLS12_377_GURVY:
		return "BLS12_377_GURVY"
	case math3.BLS12_381_BBS:
		return "BLS12_381_BBS"
	default:
		panic("invalid curve id")
	}
}
