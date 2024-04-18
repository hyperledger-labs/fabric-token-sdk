// Code generated by counterfeiter. DO NOT EDIT.
package mock

import (
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TokenService struct {
	DeserializeTokenStub        func([]byte, []byte) (*token.Token, view.Identity, error)
	deserializeTokenMutex       sync.RWMutex
	deserializeTokenArgsForCall []struct {
		arg1 []byte
		arg2 []byte
	}
	deserializeTokenReturns struct {
		result1 *token.Token
		result2 view.Identity
		result3 error
	}
	deserializeTokenReturnsOnCall map[int]struct {
		result1 *token.Token
		result2 view.Identity
		result3 error
	}
	GetTokenInfoStub        func(*driver.TokenRequestMetadata, []byte) ([]byte, error)
	getTokenInfoMutex       sync.RWMutex
	getTokenInfoArgsForCall []struct {
		arg1 *driver.TokenRequestMetadata
		arg2 []byte
	}
	getTokenInfoReturns struct {
		result1 []byte
		result2 error
	}
	getTokenInfoReturnsOnCall map[int]struct {
		result1 []byte
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *TokenService) DeserializeToken(arg1 []byte, arg2 []byte) (*token.Token, view.Identity, error) {
	var arg1Copy []byte
	if arg1 != nil {
		arg1Copy = make([]byte, len(arg1))
		copy(arg1Copy, arg1)
	}
	var arg2Copy []byte
	if arg2 != nil {
		arg2Copy = make([]byte, len(arg2))
		copy(arg2Copy, arg2)
	}
	fake.deserializeTokenMutex.Lock()
	ret, specificReturn := fake.deserializeTokenReturnsOnCall[len(fake.deserializeTokenArgsForCall)]
	fake.deserializeTokenArgsForCall = append(fake.deserializeTokenArgsForCall, struct {
		arg1 []byte
		arg2 []byte
	}{arg1Copy, arg2Copy})
	stub := fake.DeserializeTokenStub
	fakeReturns := fake.deserializeTokenReturns
	fake.recordInvocation("DeserializeToken", []interface{}{arg1Copy, arg2Copy})
	fake.deserializeTokenMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3
}

func (fake *TokenService) DeserializeTokenCallCount() int {
	fake.deserializeTokenMutex.RLock()
	defer fake.deserializeTokenMutex.RUnlock()
	return len(fake.deserializeTokenArgsForCall)
}

func (fake *TokenService) DeserializeTokenCalls(stub func([]byte, []byte) (*token.Token, view.Identity, error)) {
	fake.deserializeTokenMutex.Lock()
	defer fake.deserializeTokenMutex.Unlock()
	fake.DeserializeTokenStub = stub
}

func (fake *TokenService) DeserializeTokenArgsForCall(i int) ([]byte, []byte) {
	fake.deserializeTokenMutex.RLock()
	defer fake.deserializeTokenMutex.RUnlock()
	argsForCall := fake.deserializeTokenArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *TokenService) DeserializeTokenReturns(result1 *token.Token, result2 view.Identity, result3 error) {
	fake.deserializeTokenMutex.Lock()
	defer fake.deserializeTokenMutex.Unlock()
	fake.DeserializeTokenStub = nil
	fake.deserializeTokenReturns = struct {
		result1 *token.Token
		result2 view.Identity
		result3 error
	}{result1, result2, result3}
}

func (fake *TokenService) DeserializeTokenReturnsOnCall(i int, result1 *token.Token, result2 view.Identity, result3 error) {
	fake.deserializeTokenMutex.Lock()
	defer fake.deserializeTokenMutex.Unlock()
	fake.DeserializeTokenStub = nil
	if fake.deserializeTokenReturnsOnCall == nil {
		fake.deserializeTokenReturnsOnCall = make(map[int]struct {
			result1 *token.Token
			result2 view.Identity
			result3 error
		})
	}
	fake.deserializeTokenReturnsOnCall[i] = struct {
		result1 *token.Token
		result2 view.Identity
		result3 error
	}{result1, result2, result3}
}

func (fake *TokenService) GetTokenInfo(arg1 *driver.TokenRequestMetadata, arg2 []byte) ([]byte, error) {
	var arg2Copy []byte
	if arg2 != nil {
		arg2Copy = make([]byte, len(arg2))
		copy(arg2Copy, arg2)
	}
	fake.getTokenInfoMutex.Lock()
	ret, specificReturn := fake.getTokenInfoReturnsOnCall[len(fake.getTokenInfoArgsForCall)]
	fake.getTokenInfoArgsForCall = append(fake.getTokenInfoArgsForCall, struct {
		arg1 *driver.TokenRequestMetadata
		arg2 []byte
	}{arg1, arg2Copy})
	stub := fake.GetTokenInfoStub
	fakeReturns := fake.getTokenInfoReturns
	fake.recordInvocation("GetTokenInfo", []interface{}{arg1, arg2Copy})
	fake.getTokenInfoMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *TokenService) GetTokenInfoCallCount() int {
	fake.getTokenInfoMutex.RLock()
	defer fake.getTokenInfoMutex.RUnlock()
	return len(fake.getTokenInfoArgsForCall)
}

func (fake *TokenService) GetTokenInfoCalls(stub func(*driver.TokenRequestMetadata, []byte) ([]byte, error)) {
	fake.getTokenInfoMutex.Lock()
	defer fake.getTokenInfoMutex.Unlock()
	fake.GetTokenInfoStub = stub
}

func (fake *TokenService) GetTokenInfoArgsForCall(i int) (*driver.TokenRequestMetadata, []byte) {
	fake.getTokenInfoMutex.RLock()
	defer fake.getTokenInfoMutex.RUnlock()
	argsForCall := fake.getTokenInfoArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *TokenService) GetTokenInfoReturns(result1 []byte, result2 error) {
	fake.getTokenInfoMutex.Lock()
	defer fake.getTokenInfoMutex.Unlock()
	fake.GetTokenInfoStub = nil
	fake.getTokenInfoReturns = struct {
		result1 []byte
		result2 error
	}{result1, result2}
}

func (fake *TokenService) GetTokenInfoReturnsOnCall(i int, result1 []byte, result2 error) {
	fake.getTokenInfoMutex.Lock()
	defer fake.getTokenInfoMutex.Unlock()
	fake.GetTokenInfoStub = nil
	if fake.getTokenInfoReturnsOnCall == nil {
		fake.getTokenInfoReturnsOnCall = make(map[int]struct {
			result1 []byte
			result2 error
		})
	}
	fake.getTokenInfoReturnsOnCall[i] = struct {
		result1 []byte
		result2 error
	}{result1, result2}
}

func (fake *TokenService) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.deserializeTokenMutex.RLock()
	defer fake.deserializeTokenMutex.RUnlock()
	fake.getTokenInfoMutex.RLock()
	defer fake.getTokenInfoMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *TokenService) recordInvocation(key string, args []interface{}) {
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

var _ driver.TokensService = new(TokenService)
