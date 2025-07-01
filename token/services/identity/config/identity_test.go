/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config_test

import (
	"fmt"
	"testing"

	config3 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/config"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509/crypto"
	"github.com/stretchr/testify/assert"
)

func TestInvalidConf(t *testing.T) {
	cp, err := config3.NewProvider("./testdata/invalid")
	assert.NoError(t, err)
	tms := config2.NewConfiguration(cp, "v1", "n1c1ns1", driver.TMSID{})
	_, err = config.NewIdentityConfig(tms)
	assert.Error(t, err, "should have failed due to invalid config")
}

func TestCacheSizeIdentityConfig(t *testing.T) {
	cp, err := config3.NewProvider("./testdata/token0")
	assert.NoError(t, err)
	tms := config2.NewConfiguration(cp, "v1", "n1c1ns1", driver.TMSID{})
	identityConfig, err := config.NewIdentityConfig(tms)
	assert.NoError(t, err, "failed creating identity config")
	assert.Equal(t, 3, identityConfig.DefaultCacheSize(), "default cache size should be 3")
	assert.Equal(t, 5, identityConfig.CacheSizeForOwnerID("owner1"), "alice cache size should be 5")
	assert.Equal(t, 3, identityConfig.CacheSizeForOwnerID("unknown"))
}

func TestTranslatePath(t *testing.T) {
	cp, err := config3.NewProvider("./testdata/token0")
	assert.NoError(t, err)
	tms := config2.NewConfiguration(cp, "v1", "n1c1ns1", driver.TMSID{})
	identityConfig, err := config.NewIdentityConfig(tms)
	assert.NoError(t, err, "failed creating identity config")
	assert.Contains(t, identityConfig.TranslatePath("./testdata/token0"), "fabric-token-sdk", "translate path should contain fabric-token-sdk")
}

func TestIdentitiesForRole(t *testing.T) {
	cp, err := config3.NewProvider("./testdata/token0")
	assert.NoError(t, err)
	tms := config2.NewConfiguration(cp, "v1", "n1c1ns1", driver.TMSID{})
	identityConfig, err := config.NewIdentityConfig(tms)
	assert.NoError(t, err, "failed creating identity config")

	identities, err := identityConfig.IdentitiesForRole(idriver.OwnerRole)
	assert.NoError(t, err, "failed getting identities for owner role")
	assert.Equal(t, 1, len(identities), "should have 1 owner identity")
	for i, identity := range identities {
		index := i + 1
		assert.Equal(t, fmt.Sprintf("owner%d", index), identity.ID, "id should have been owner%d", index)
		assert.Equal(t, fmt.Sprintf("owner%d", index), identity.String(), "id should have been owner%d", index)
		assert.Equal(t, fmt.Sprintf("/path/to/crypto/owner%d", index), identity.Path, "path should have been path /path/to/crypto/owner%d", index)
	}

	identities, err = identityConfig.IdentitiesForRole(idriver.IssuerRole)
	assert.NoError(t, err, "failed getting identities for issuer role")
	assert.Equal(t, 2, len(identities), "should have 2 issuer identity")
	iss, err := crypto.ToBCCSPOpts(identities[1].Opts)
	assert.NoError(t, err, "failed converting to bccsp opts")
	assert.Equal(t, "SW", iss.Default)
	assert.Equal(t, "1234", iss.PKCS11.Pin)
	assert.Equal(t, 256, iss.SW.Security)

	identities, err = identityConfig.IdentitiesForRole(idriver.AuditorRole)
	assert.NoError(t, err, "failed getting identities for auditor role")
	assert.Equal(t, 3, len(identities), "should have 3 auditor identity")

	identities, err = identityConfig.IdentitiesForRole(idriver.CertifierRole)
	assert.NoError(t, err, "failed getting identities for certifier role")
	assert.Equal(t, 4, len(identities), "should have 4 certifier identity")

	_, err = identityConfig.IdentitiesForRole(1234)
	assert.Error(t, err, "should throw identity for invalid role (1234)")
	assert.Equal(t, err.Error(), "unknown role [1234]", "should throw identity for invalid role (1234)")
}
