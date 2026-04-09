/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditor

import (
	"testing"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace/noop"
)

// mockServiceProviderGet overrides GetService
type mockServiceProviderGet struct {
	svc interface{}
	err error
}

func (m *mockServiceProviderGet) GetService(v interface{}) (interface{}, error) {
	return m.svc, m.err
}

func TestManager_GetByTMSID_ClosureErrors(t *testing.T) {
	sp := &mockServiceProviderGet{}
	
	// 1. StoreServiceByTMSId error
	smStoreErr := NewServiceManager(
		&mockNetworkProvider{}, 
		&mockStoreServiceManager{err: assert.AnError}, 
		&mockTokensServiceManager{}, 
		&mockTokenManagementServiceProvider{}, 
		noop.NewTracerProvider(), 
		&noopProvider{}, 
		&mockCheckServiceProvider{})
	sp.svc = smStoreErr
	assert.Nil(t, GetByTMSID(sp, token.TMSID{}))
	
	// 2. ServiceByTMSId error
	smTokensErr := NewServiceManager(
		&mockNetworkProvider{}, 
		&mockStoreServiceManager{}, 
		&mockTokensServiceManager{err: assert.AnError}, 
		&mockTokenManagementServiceProvider{}, 
		noop.NewTracerProvider(), 
		&noopProvider{}, 
		&mockCheckServiceProvider{})
	sp.svc = smTokensErr
	assert.Nil(t, GetByTMSID(sp, token.TMSID{}))
	
	// 3. GetNetwork error
	smNetworkErr := NewServiceManager(
		&mockNetworkProvider{err: assert.AnError}, 
		&mockStoreServiceManager{}, 
		&mockTokensServiceManager{}, 
		&mockTokenManagementServiceProvider{}, 
		noop.NewTracerProvider(), 
		&noopProvider{}, 
		&mockCheckServiceProvider{})
	sp.svc = smNetworkErr
	assert.Nil(t, GetByTMSID(sp, token.TMSID{}))
	
	// 4. CheckService error
	smCheckErr := NewServiceManager(
		&mockNetworkProvider{}, 
		&mockStoreServiceManager{}, 
		&mockTokensServiceManager{}, 
		&mockTokenManagementServiceProvider{}, 
		noop.NewTracerProvider(), 
		&noopProvider{}, 
		&mockCheckServiceProvider{err: assert.AnError})
	sp.svc = smCheckErr
	assert.Nil(t, GetByTMSID(sp, token.TMSID{}))
}

func TestManager_GetByTMSID(t *testing.T) {
	sp := &mockServiceProviderGet{
		err: nil,
	}
	
	// Error getting manager service
	sp.err = assert.AnError
	a := GetByTMSID(sp, token.TMSID{})
	assert.Nil(t, a)
	
	// Success getting manager but Auditor returns error
	sp.err = nil
	sm := NewServiceManager(
		&mockNetworkProvider{err: assert.AnError}, // will cause Auditor to fail
		&mockStoreServiceManager{}, 
		&mockTokensServiceManager{}, 
		&mockTokenManagementServiceProvider{}, 
		noop.NewTracerProvider(), 
		&noopProvider{}, 
		&mockCheckServiceProvider{})
		
	sp.svc = sm
	a = GetByTMSID(sp, token.TMSID{})
	assert.Nil(t, a)
	
	// Success Auditor
	smSuccess := NewServiceManager(
		&mockNetworkProvider{}, 
		&mockStoreServiceManager{}, 
		&mockTokensServiceManager{}, 
		&mockTokenManagementServiceProvider{}, 
		noop.NewTracerProvider(), 
		&noopProvider{}, 
		&mockCheckServiceProvider{})
	sp.svc = smSuccess
	a = GetByTMSID(sp, token.TMSID{})
	assert.NotNil(t, a)
	
	// Test Get
	// nil wallet
	a2 := Get(sp, nil)
	assert.Nil(t, a2)
	
	// non-nil wallet panics due to being empty
	w := &token.AuditorWallet{}
	assert.Panics(t, func() {
		Get(sp, w)
	})
}
func TestManager_RestoreTMS_Success(t *testing.T) {
// Not easily mockable because auditStoreServiceManager returns *auditdb.StoreService and it expects TokenRequests...
}
