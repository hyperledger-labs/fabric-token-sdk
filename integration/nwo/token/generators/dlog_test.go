/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package generators

import (
	"os"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/mock"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
)

//go:generate counterfeiter -o mock/token_platform.go -fake-name TokenPlatform . tokenPlatform

func TestDLogFabricCryptoMaterialGenerator_Setup(t *testing.T) {
	gomega.RegisterTestingT(t)
	buildServer := common.NewBuildServer()
	buildServer.Serve()
	defer buildServer.Shutdown()

	tp := &mock.TokenPlatform{}
	tp.TokenDirReturns("./testdata/token")
	defer os.RemoveAll("./testdata/token")
	tp.GetBuilderReturns(buildServer.Client())
	tms := &topology.TMS{
		Network:   "test_network",
		Channel:   "test_channel",
		Namespace: "test_namespace",
	}
	gen := NewDLogCryptoMaterialGenerator(tp, math.BN254, buildServer.Client())
	assert.NotNil(t, gen)
	path, err := gen.Setup(tms)
	assert.NoError(t, err)
	assert.NotEmpty(t, path)
}
