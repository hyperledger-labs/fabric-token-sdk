/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	. "github.com/onsi/gomega"
)

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
