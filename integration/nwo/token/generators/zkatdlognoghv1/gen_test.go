/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package zkatdlognoghv1

import (
	"os"
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/zkatdlognoghv1/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
)

//go:generate counterfeiter -o mock/token_platform.go -fake-name TokenPlatform . tokenPlatform

func TestDLogFabricCryptoMaterialGenerator_Setup(t *testing.T) {
	gomega.RegisterTestingT(t)
	buildServer := common.NewBuildServer("-tags", "pkcs11")
	buildServer.Serve()
	defer buildServer.Shutdown(true)

	tp := &mock.TokenPlatform{}
	tp.TokenDirReturns("./testdata/token")
	defer utils.IgnoreErrorWithOneArg(os.RemoveAll, "./testdata")
	tp.GetBuilderReturns(buildServer.Client())
	tms := &topology.TMS{
		Network:   "test_network",
		Channel:   "test_channel",
		Namespace: "test_namespace",
	}
	gen := NewCryptoMaterialGenerator(tp, math.BN254, buildServer.Client())
	assert.NotNil(t, gen)
	path, err := gen.Setup(tms)
	assert.NoError(t, err)
	assert.NotEmpty(t, path)
}
