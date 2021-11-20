/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package generators

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	. "github.com/onsi/gomega"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
)

type FabTokenPublicParamsGenerator struct{}

func (f *FabTokenPublicParamsGenerator) Generate(tms *topology.TMS, args ...interface{}) ([]byte, error) {
	pp, err := fabtoken.Setup()
	if err != nil {
		return nil, err
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

func (d *FabTokenFabricCryptoMaterialGenerator) GenerateCertifierIdentities(tms *topology.TMS, node *node.Node, name ...string) []Identity {
	panic("not supported")
}

func (d *FabTokenFabricCryptoMaterialGenerator) GenerateOwnerIdentities(tms *topology.TMS, n *node.Node, owners ...string) []Identity {
	fp := d.tokenPlatform.GetContext().PlatformByName(tms.Network).(fabricPlatform)
	peer := fp.PeersByID(n.ID())
	Expect(peer).NotTo(BeNil())

	var res []Identity
	for _, owner := range owners {
		found := false
		for _, identity := range peer.Identities {
			if identity.ID == owner && identity.Type == "bccsp" {
				res = append(res, Identity{
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

func (d *FabTokenFabricCryptoMaterialGenerator) GenerateIssuerIdentities(tms *topology.TMS, n *node.Node, issuers ...string) []Identity {
	fp := d.tokenPlatform.GetContext().PlatformByName(tms.Network).(fabricPlatform)
	peer := fp.PeersByID(n.ID())
	Expect(peer).NotTo(BeNil())

	var res []Identity
	for _, issuer := range issuers {
		found := false
		for _, identity := range peer.Identities {
			if identity.ID == issuer && identity.Type == "bccsp" {
				res = append(res, Identity{
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

func (d *FabTokenFabricCryptoMaterialGenerator) GenerateAuditorIdentities(tms *topology.TMS, n *node.Node, auditors ...string) []Identity {
	fp := d.tokenPlatform.GetContext().PlatformByName(tms.Network).(fabricPlatform)
	peer := fp.PeersByID(n.ID())
	Expect(peer).NotTo(BeNil())

	var res []Identity
	for _, auditor := range auditors {
		found := false
		for _, identity := range peer.Identities {
			if identity.ID == auditor && identity.Type == "bccsp" {
				res = append(res, Identity{
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
