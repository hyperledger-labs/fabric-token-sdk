/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"os"
	"strings"
	"testing"

	math "github.com/IBM/mathlib"
	idemix2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/idemix"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/mock"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	sig2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/core/sig"
	_ "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	registry2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/registry"
	"github.com/hyperledger/fabric/msp"
	"github.com/stretchr/testify/assert"
)

func TestDeserialization(t *testing.T) {
	registry := registry2.New()
	kvss, err := kvs.NewWithConfig(registry, "memory", "", &mock.ConfigProvider{})
	assert.NoError(t, err)
	assert.NoError(t, registry.RegisterService(kvss))
	sigService := sig2.NewSignService(registry, nil, kvss)
	assert.NoError(t, registry.RegisterService(sigService))
	config, err := msp.GetLocalMspConfigWithType("./testdata/idemix", nil, "idemix", "idemix")
	assert.NoError(t, err)
	provider, err := idemix2.NewProviderWithAnyPolicyAndCurve(config, registry, math.FP256BN_AMCL)
	assert.NoError(t, err)

	ipk, err := os.ReadFile("./testdata/idemix/msp/IssuerPublicKey")
	assert.NoError(t, err)
	rpk, err := os.ReadFile("./testdata/idemix/msp/RevocationPublicKey")
	assert.NoError(t, err)

	deserializer, err := NewDeserializer(ipk, rpk, math.FP256BN_AMCL)
	assert.NoError(t, err)

	id, _, err := provider.Identity(nil)
	assert.NoError(t, err)
	_, err = deserializer.DeserializeVerifier(id)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "no EidNym provided but ExpectEidNymRhNym required"))

	id, _, err = provider.Identity(&driver.IdentityOptions{EIDExtension: true})
	assert.NoError(t, err)
	_, err = deserializer.DeserializeVerifier(id)
	assert.NoError(t, err)
}
