/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/config"
	"github.com/stretchr/testify/assert"
)

// TestGetTMSs tests the Configurations function
func TestGetTMSs(t *testing.T) {
	// create a new config service by loading the config file
	cp, err := config.NewProvider("./testdata/token0")
	assert.NoError(t, err)

	// instantiate the token sdk config
	tokenSDKConfig := NewService(cp)

	// compare the TMSs obtained from Configurations with the corresponding TMSs obtained from ConfigurationFor
	tmss, err := tokenSDKConfig.Configurations()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(tmss))
	for _, tms := range tmss {
		tms2, err := tokenSDKConfig.ConfigurationFor(tms.ID().Network, tms.ID().Channel, tms.ID().Namespace)
		assert.NoError(t, err)
		assert.Equal(t, tms, tms2)
	}
}
