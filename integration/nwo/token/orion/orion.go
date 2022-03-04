/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"time"

	api2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	sfcnode "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	. "github.com/onsi/gomega/gexec"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators"
	topology2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
)

var logger = flogging.MustGetLogger("integration.token.orion")

type tokenPlatform interface {
	TokenGen(keygen common.Command) (*Session, error)
	PublicParametersFile(tms *topology2.TMS) string
	GetContext() api2.Context
	PublicParameters(tms *topology2.TMS) []byte
	GetPublicParamsGenerators(driver string) generators.PublicParamsGenerator
	GetCryptoMaterialGenerator(driver string) generators.CryptoMaterialGenerator
	PublicParametersDir() string
	GetBuilder() api2.Builder
	TokenDir() string
}

type NetworkHandler struct {
	TokenPlatform     tokenPlatform
	EventuallyTimeout time.Duration
}

func NewNetworkHandler(tokenPlatform tokenPlatform) *NetworkHandler {
	return &NetworkHandler{
		TokenPlatform:     tokenPlatform,
		EventuallyTimeout: 10 * time.Minute,
	}
}

func (p *NetworkHandler) GenerateArtifacts(tms *topology2.TMS) {
	panic("not implemented")
}

func (p *NetworkHandler) GenerateExtension(tms *topology2.TMS, node *sfcnode.Node) string {
	panic("not implemented")
}

func (p *NetworkHandler) PostRun(load bool, tms *topology2.TMS) {
	panic("not implemented")
}
