/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"strings"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
)

type FabTokenPublicParamsGenerator struct {
}

func NewFabTokenPublicParamsGenerator() *FabTokenPublicParamsGenerator {
	return &FabTokenPublicParamsGenerator{}
}

func (f *FabTokenPublicParamsGenerator) Generate(tms *topology.TMS, wallets *generators.Wallets, args ...interface{}) ([]byte, error) {
	pp, err := fabtoken.Setup()
	if err != nil {
		return nil, err
	}

	if len(tms.Auditors) != 0 {
		if len(wallets.Auditors) == 0 {
			return nil, errors.Errorf("no auditor wallets provided")
		}
		for _, auditor := range wallets.Auditors {
			// Build an MSP Identity
			types := strings.Split(auditor.Type, ":")
			provider, err := x509.NewProvider(auditor.Path, types[1], nil)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to create x509 provider")
			}
			id, _, err := provider.Identity(nil)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to get identity")
			}
			pp.AddAuditor(id)
		}
	}

	if len(tms.Issuers) != 0 {
		if len(wallets.Issuers) == 0 {
			return nil, errors.Errorf("no issuer wallets provided")
		}
		for _, issuer := range wallets.Issuers {
			// Build an MSP Identity
			types := strings.Split(issuer.Type, ":")
			provider, err := x509.NewProvider(issuer.Path, types[1], nil)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to create x509 provider")
			}
			id, _, err := provider.Identity(nil)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to get identity")
			}
			pp.AddIssuer(id)
		}
	}

	ppRaw, err := pp.Serialize()
	if err != nil {
		return nil, err
	}
	return ppRaw, nil
}

type FabTokenFabricCryptoMaterialGenerator struct {
	tokenPlatform tokenPlatform
}

func NewFabTokenFabricCryptoMaterialGenerator(tokenPlatform tokenPlatform) *FabTokenFabricCryptoMaterialGenerator {
	return &FabTokenFabricCryptoMaterialGenerator{tokenPlatform: tokenPlatform}
}

func (d *FabTokenFabricCryptoMaterialGenerator) Setup(tms *topology.TMS) (string, error) {
	return "", nil
}

func (d *FabTokenFabricCryptoMaterialGenerator) GenerateCertifierIdentities(tms *topology.TMS, node *node.Node, name ...string) []generators.Identity {
	return nil
}

func (d *FabTokenFabricCryptoMaterialGenerator) GenerateOwnerIdentities(tms *topology.TMS, n *node.Node, owners ...string) []generators.Identity {
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
			if identity.ID == owner && identity.Type == "bccsp" {
				res = append(res, generators.Identity{
					ID:   owner,
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

func (d *FabTokenFabricCryptoMaterialGenerator) GenerateIssuerIdentities(tms *topology.TMS, n *node.Node, issuers ...string) []generators.Identity {
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

func (d *FabTokenFabricCryptoMaterialGenerator) GenerateAuditorIdentities(tms *topology.TMS, n *node.Node, auditors ...string) []generators.Identity {
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
