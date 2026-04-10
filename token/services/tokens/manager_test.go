/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens/mock"
	"github.com/test-go/testify/assert"
	"github.com/test-go/testify/require"
)

func TestServiceManager(t *testing.T) {
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	mockTMSProv := &mock.FakeTMSProvider{}
	mockStoreServProv := &mock.FakeStoreServiceManager{}

	mockNetProv := &mock.FakeNetworkProvider{}
	mockPub := &mock.FakePublisher{}

	manager := tokens.NewServiceManager(mockTMSProv, mockStoreServProv, mockNetProv, mockPub)
	require.NotNil(t, manager)

	// Test ServiceByTMSId
	service, err := manager.ServiceByTMSId(tmsID)
	require.NoError(t, err)
	assert.NotNil(t, service)
	assert.Equal(t, mockTMSProv, service.TMSProvider)
	assert.Equal(t, mockNetProv, service.NetworkProvider)

	// Test GetService helper
	sp := &mock.FakeServiceProvider{}
	sp.GetServiceReturns(manager, nil)

	service2, err := tokens.GetService(sp, tmsID)
	require.NoError(t, err)
	assert.Equal(t, service, service2)

	// Test GetService error
	spErr := &mock.FakeServiceProvider{}
	spErr.GetServiceReturns(nil, assert.AnError)

	_, err = tokens.GetService(spErr, tmsID)
	assert.Error(t, err)
}
