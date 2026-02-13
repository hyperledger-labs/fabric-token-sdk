/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/config/mock"
	"github.com/stretchr/testify/assert"
)

func TestServiceListenerManagerConfig_Type(t *testing.T) {
	fakeConfig := &mock.Configuration{}

	// Case 1: Configuration returns a valid manager type
	fakeConfig.GetStringReturns("custom-type")
	c := config.NewListenerManagerConfig(fakeConfig)
	assert.Equal(t, config.ManagerType("custom-type"), c.Type())
	assert.Equal(t, 1, fakeConfig.GetStringCallCount())
	assert.Equal(t, config.Type, fakeConfig.GetStringArgsForCall(0))

	// Case 2: Configuration returns an empty string, should default to Notification
	fakeConfig.GetStringReturns("")
	assert.Equal(t, config.Notification, c.Type())
	assert.Equal(t, 2, fakeConfig.GetStringCallCount())
}
