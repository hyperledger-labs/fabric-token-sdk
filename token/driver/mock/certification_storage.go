// Code generated by counterfeiter. DO NOT EDIT.
package mock

import (
	"context"
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type CertificationStorage struct {
	ExistsStub        func(context.Context, *token.ID) bool
	existsMutex       sync.RWMutex
	existsArgsForCall []struct {
		arg1 context.Context
		arg2 *token.ID
	}
	existsReturns struct {
		result1 bool
	}
	existsReturnsOnCall map[int]struct {
		result1 bool
	}
	StoreStub        func(context.Context, map[*token.ID][]byte) error
	storeMutex       sync.RWMutex
	storeArgsForCall []struct {
		arg1 context.Context
		arg2 map[*token.ID][]byte
	}
	storeReturns struct {
		result1 error
	}
	storeReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *CertificationStorage) Exists(arg1 context.Context, arg2 *token.ID) bool {
	fake.existsMutex.Lock()
	ret, specificReturn := fake.existsReturnsOnCall[len(fake.existsArgsForCall)]
	fake.existsArgsForCall = append(fake.existsArgsForCall, struct {
		arg1 context.Context
		arg2 *token.ID
	}{arg1, arg2})
	stub := fake.ExistsStub
	fakeReturns := fake.existsReturns
	fake.recordInvocation("Exists", []interface{}{arg1, arg2})
	fake.existsMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *CertificationStorage) ExistsCallCount() int {
	fake.existsMutex.RLock()
	defer fake.existsMutex.RUnlock()
	return len(fake.existsArgsForCall)
}

func (fake *CertificationStorage) ExistsCalls(stub func(context.Context, *token.ID) bool) {
	fake.existsMutex.Lock()
	defer fake.existsMutex.Unlock()
	fake.ExistsStub = stub
}

func (fake *CertificationStorage) ExistsArgsForCall(i int) (context.Context, *token.ID) {
	fake.existsMutex.RLock()
	defer fake.existsMutex.RUnlock()
	argsForCall := fake.existsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *CertificationStorage) ExistsReturns(result1 bool) {
	fake.existsMutex.Lock()
	defer fake.existsMutex.Unlock()
	fake.ExistsStub = nil
	fake.existsReturns = struct {
		result1 bool
	}{result1}
}

func (fake *CertificationStorage) ExistsReturnsOnCall(i int, result1 bool) {
	fake.existsMutex.Lock()
	defer fake.existsMutex.Unlock()
	fake.ExistsStub = nil
	if fake.existsReturnsOnCall == nil {
		fake.existsReturnsOnCall = make(map[int]struct {
			result1 bool
		})
	}
	fake.existsReturnsOnCall[i] = struct {
		result1 bool
	}{result1}
}

func (fake *CertificationStorage) Store(arg1 context.Context, arg2 map[*token.ID][]byte) error {
	fake.storeMutex.Lock()
	ret, specificReturn := fake.storeReturnsOnCall[len(fake.storeArgsForCall)]
	fake.storeArgsForCall = append(fake.storeArgsForCall, struct {
		arg1 context.Context
		arg2 map[*token.ID][]byte
	}{arg1, arg2})
	stub := fake.StoreStub
	fakeReturns := fake.storeReturns
	fake.recordInvocation("Store", []interface{}{arg1, arg2})
	fake.storeMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *CertificationStorage) StoreCallCount() int {
	fake.storeMutex.RLock()
	defer fake.storeMutex.RUnlock()
	return len(fake.storeArgsForCall)
}

func (fake *CertificationStorage) StoreCalls(stub func(context.Context, map[*token.ID][]byte) error) {
	fake.storeMutex.Lock()
	defer fake.storeMutex.Unlock()
	fake.StoreStub = stub
}

func (fake *CertificationStorage) StoreArgsForCall(i int) (context.Context, map[*token.ID][]byte) {
	fake.storeMutex.RLock()
	defer fake.storeMutex.RUnlock()
	argsForCall := fake.storeArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *CertificationStorage) StoreReturns(result1 error) {
	fake.storeMutex.Lock()
	defer fake.storeMutex.Unlock()
	fake.StoreStub = nil
	fake.storeReturns = struct {
		result1 error
	}{result1}
}

func (fake *CertificationStorage) StoreReturnsOnCall(i int, result1 error) {
	fake.storeMutex.Lock()
	defer fake.storeMutex.Unlock()
	fake.StoreStub = nil
	if fake.storeReturnsOnCall == nil {
		fake.storeReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.storeReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *CertificationStorage) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.existsMutex.RLock()
	defer fake.existsMutex.RUnlock()
	fake.storeMutex.RLock()
	defer fake.storeMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *CertificationStorage) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ driver.CertificationStorage = new(CertificationStorage)
