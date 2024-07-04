// Code generated by counterfeiter. DO NOT EDIT.
package mock

import (
	"context"
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/tcc"
)

type Validator struct {
	UnmarshallAndVerifyWithMetadataStub        func(context.Context, driver.ValidatorLedger, string, []byte) ([]interface{}, map[string][]byte, error)
	unmarshallAndVerifyWithMetadataMutex       sync.RWMutex
	unmarshallAndVerifyWithMetadataArgsForCall []struct {
		arg1 context.Context
		arg2 driver.ValidatorLedger
		arg3 string
		arg4 []byte
	}
	unmarshallAndVerifyWithMetadataReturns struct {
		result1 []interface{}
		result2 map[string][]byte
		result3 error
	}
	unmarshallAndVerifyWithMetadataReturnsOnCall map[int]struct {
		result1 []interface{}
		result2 map[string][]byte
		result3 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *Validator) UnmarshallAndVerifyWithMetadata(arg1 context.Context, arg2 driver.ValidatorLedger, arg3 string, arg4 []byte) ([]interface{}, map[string][]byte, error) {
	var arg4Copy []byte
	if arg4 != nil {
		arg4Copy = make([]byte, len(arg4))
		copy(arg4Copy, arg4)
	}
	fake.unmarshallAndVerifyWithMetadataMutex.Lock()
	ret, specificReturn := fake.unmarshallAndVerifyWithMetadataReturnsOnCall[len(fake.unmarshallAndVerifyWithMetadataArgsForCall)]
	fake.unmarshallAndVerifyWithMetadataArgsForCall = append(fake.unmarshallAndVerifyWithMetadataArgsForCall, struct {
		arg1 context.Context
		arg2 driver.ValidatorLedger
		arg3 string
		arg4 []byte
	}{arg1, arg2, arg3, arg4Copy})
	stub := fake.UnmarshallAndVerifyWithMetadataStub
	fakeReturns := fake.unmarshallAndVerifyWithMetadataReturns
	fake.recordInvocation("UnmarshallAndVerifyWithMetadata", []interface{}{arg1, arg2, arg3, arg4Copy})
	fake.unmarshallAndVerifyWithMetadataMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3, arg4)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3
}

func (fake *Validator) UnmarshallAndVerifyWithMetadataCallCount() int {
	fake.unmarshallAndVerifyWithMetadataMutex.RLock()
	defer fake.unmarshallAndVerifyWithMetadataMutex.RUnlock()
	return len(fake.unmarshallAndVerifyWithMetadataArgsForCall)
}

func (fake *Validator) UnmarshallAndVerifyWithMetadataCalls(stub func(context.Context, driver.ValidatorLedger, string, []byte) ([]interface{}, map[string][]byte, error)) {
	fake.unmarshallAndVerifyWithMetadataMutex.Lock()
	defer fake.unmarshallAndVerifyWithMetadataMutex.Unlock()
	fake.UnmarshallAndVerifyWithMetadataStub = stub
}

func (fake *Validator) UnmarshallAndVerifyWithMetadataArgsForCall(i int) (context.Context, driver.ValidatorLedger, string, []byte) {
	fake.unmarshallAndVerifyWithMetadataMutex.RLock()
	defer fake.unmarshallAndVerifyWithMetadataMutex.RUnlock()
	argsForCall := fake.unmarshallAndVerifyWithMetadataArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3, argsForCall.arg4
}

func (fake *Validator) UnmarshallAndVerifyWithMetadataReturns(result1 []interface{}, result2 map[string][]byte, result3 error) {
	fake.unmarshallAndVerifyWithMetadataMutex.Lock()
	defer fake.unmarshallAndVerifyWithMetadataMutex.Unlock()
	fake.UnmarshallAndVerifyWithMetadataStub = nil
	fake.unmarshallAndVerifyWithMetadataReturns = struct {
		result1 []interface{}
		result2 map[string][]byte
		result3 error
	}{result1, result2, result3}
}

func (fake *Validator) UnmarshallAndVerifyWithMetadataReturnsOnCall(i int, result1 []interface{}, result2 map[string][]byte, result3 error) {
	fake.unmarshallAndVerifyWithMetadataMutex.Lock()
	defer fake.unmarshallAndVerifyWithMetadataMutex.Unlock()
	fake.UnmarshallAndVerifyWithMetadataStub = nil
	if fake.unmarshallAndVerifyWithMetadataReturnsOnCall == nil {
		fake.unmarshallAndVerifyWithMetadataReturnsOnCall = make(map[int]struct {
			result1 []interface{}
			result2 map[string][]byte
			result3 error
		})
	}
	fake.unmarshallAndVerifyWithMetadataReturnsOnCall[i] = struct {
		result1 []interface{}
		result2 map[string][]byte
		result3 error
	}{result1, result2, result3}
}

func (fake *Validator) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.unmarshallAndVerifyWithMetadataMutex.RLock()
	defer fake.unmarshallAndVerifyWithMetadataMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *Validator) recordInvocation(key string, args []interface{}) {
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

var _ tcc.Validator = new(Validator)
