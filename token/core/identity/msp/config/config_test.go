/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/config"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/config"
	"github.com/stretchr/testify/assert"
)

func TestToBCCSPOpts(t *testing.T) {
	// create a new config service by loading the config file
	cp, err := config.NewProvider("./testdata/sqlite")
	assert.NoError(t, err)

	// instantiate the token sdk config
	tokenSDKConfig := config2.NewTokenSDK(cp)

	// compare the TMSs obtained from GetTMSs with the corresponding TMSs obtained from GetTMS
	tmss, err := tokenSDKConfig.GetTMSs()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(tmss))
	for _, tms := range tmss {
		tms2, err := tokenSDKConfig.GetTMS(tms.TMS().Network, tms.TMS().Channel, tms.TMS().Namespace)
		assert.NoError(t, err)
		assert.Equal(t, tms, tms2)

		assert.Len(t, tms2.TMS().Wallets.Owners, 2)

		bccspOpts, err := ToBCCSPOpts(tms2.TMS().Wallets.Owners[0].Opts)
		assert.NoError(t, err)
		assert.NotNil(t, bccspOpts)
		assert.Equal(t, "SW", bccspOpts.Default)
		assert.Equal(t, 256, bccspOpts.SW.Security)
		assert.Equal(t, 256, bccspOpts.PKCS11.Security)
	}
}
