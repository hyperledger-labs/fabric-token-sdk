/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/config"
	"github.com/stretchr/testify/assert"
)

// TestGetTMSs tests the GetTMSs function
func TestGetTMSs(t *testing.T) {
	// create a new config service by loading the config file
	cp, err := config.NewProvider("./testdata/token0")
	assert.NoError(t, err)

	// instantiate the token sdk config
	tokenSDKConfig := NewTokenSDK(cp)

	// compare the TMSs obtained from GetTMSs with the corresponding TMSs obtained from GetTMS
	tmss, err := tokenSDKConfig.GetTMSs()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(tmss))
	for _, tms := range tmss {
		tms2, err := tokenSDKConfig.GetTMS(tms.TMS().Network, tms.TMS().Channel, tms.TMS().Namespace)
		assert.NoError(t, err)
		assert.Equal(t, tms, tms2)
	}
}
