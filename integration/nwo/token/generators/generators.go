/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package generators

import (
	api2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/onsi/gomega/gexec"
)

type TokenPlatform interface {
	TokenGen(keygen common.Command) (*gexec.Session, error)
	PublicParametersFile(tms *topology.TMS) string
	GetContext() api2.Context
	PublicParameters(tms *topology.TMS) []byte
	GetPublicParamsGenerators(driver string) PublicParamsGenerator
	PublicParametersDir() string
	GetBuilder() api2.Builder
	TokenDir() string
}

type PublicParamsGenerator interface {
	Generate(tms *topology.TMS, wallets *Wallets, args ...interface{}) ([]byte, error)
}

type CryptoMaterialGenerator interface {
	Setup(tms *topology.TMS) (string, error)
	GenerateCertifierIdentities(tms *topology.TMS, node *node.Node, certifiers ...string) []Identity
	GenerateOwnerIdentities(tms *topology.TMS, n *node.Node, owners ...string) []Identity
	GenerateIssuerIdentities(tms *topology.TMS, n *node.Node, issuers ...string) []Identity
	GenerateAuditorIdentities(tms *topology.TMS, n *node.Node, auditors ...string) []Identity
}
