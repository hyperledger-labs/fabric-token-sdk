// Code generated by counterfeiter. DO NOT EDIT.
package mock

import (
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/tms"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
)

type ConfigManager struct {
	TMSStub        func() *config.TMS
	tMSMutex       sync.RWMutex
	tMSArgsForCall []struct {
	}
	tMSReturns struct {
		result1 *config.TMS
	}
	tMSReturnsOnCall map[int]struct {
		result1 *config.TMS
	}
	TranslatePathStub        func(string) string
	translatePathMutex       sync.RWMutex
	translatePathArgsForCall []struct {
		arg1 string
	}
	translatePathReturns struct {
		result1 string
	}
	translatePathReturnsOnCall map[int]struct {
		result1 string
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *ConfigManager) TMS() *config.TMS {
	fake.tMSMutex.Lock()
	ret, specificReturn := fake.tMSReturnsOnCall[len(fake.tMSArgsForCall)]
	fake.tMSArgsForCall = append(fake.tMSArgsForCall, struct {
	}{})
	stub := fake.TMSStub
	fakeReturns := fake.tMSReturns
	fake.recordInvocation("TMS", []interface{}{})
	fake.tMSMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *ConfigManager) TMSCallCount() int {
	fake.tMSMutex.RLock()
	defer fake.tMSMutex.RUnlock()
	return len(fake.tMSArgsForCall)
}

func (fake *ConfigManager) TMSCalls(stub func() *config.TMS) {
	fake.tMSMutex.Lock()
	defer fake.tMSMutex.Unlock()
	fake.TMSStub = stub
}

func (fake *ConfigManager) TMSReturns(result1 *config.TMS) {
	fake.tMSMutex.Lock()
	defer fake.tMSMutex.Unlock()
	fake.TMSStub = nil
	fake.tMSReturns = struct {
		result1 *config.TMS
	}{result1}
}

func (fake *ConfigManager) TMSReturnsOnCall(i int, result1 *config.TMS) {
	fake.tMSMutex.Lock()
	defer fake.tMSMutex.Unlock()
	fake.TMSStub = nil
	if fake.tMSReturnsOnCall == nil {
		fake.tMSReturnsOnCall = make(map[int]struct {
			result1 *config.TMS
		})
	}
	fake.tMSReturnsOnCall[i] = struct {
		result1 *config.TMS
	}{result1}
}

func (fake *ConfigManager) TranslatePath(arg1 string) string {
	fake.translatePathMutex.Lock()
	ret, specificReturn := fake.translatePathReturnsOnCall[len(fake.translatePathArgsForCall)]
	fake.translatePathArgsForCall = append(fake.translatePathArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.TranslatePathStub
	fakeReturns := fake.translatePathReturns
	fake.recordInvocation("TranslatePath", []interface{}{arg1})
	fake.translatePathMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *ConfigManager) TranslatePathCallCount() int {
	fake.translatePathMutex.RLock()
	defer fake.translatePathMutex.RUnlock()
	return len(fake.translatePathArgsForCall)
}

func (fake *ConfigManager) TranslatePathCalls(stub func(string) string) {
	fake.translatePathMutex.Lock()
	defer fake.translatePathMutex.Unlock()
	fake.TranslatePathStub = stub
}

func (fake *ConfigManager) TranslatePathArgsForCall(i int) string {
	fake.translatePathMutex.RLock()
	defer fake.translatePathMutex.RUnlock()
	argsForCall := fake.translatePathArgsForCall[i]
	return argsForCall.arg1
}

func (fake *ConfigManager) TranslatePathReturns(result1 string) {
	fake.translatePathMutex.Lock()
	defer fake.translatePathMutex.Unlock()
	fake.TranslatePathStub = nil
	fake.translatePathReturns = struct {
		result1 string
	}{result1}
}

func (fake *ConfigManager) TranslatePathReturnsOnCall(i int, result1 string) {
	fake.translatePathMutex.Lock()
	defer fake.translatePathMutex.Unlock()
	fake.TranslatePathStub = nil
	if fake.translatePathReturnsOnCall == nil {
		fake.translatePathReturnsOnCall = make(map[int]struct {
			result1 string
		})
	}
	fake.translatePathReturnsOnCall[i] = struct {
		result1 string
	}{result1}
}

func (fake *ConfigManager) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.tMSMutex.RLock()
	defer fake.tMSMutex.RUnlock()
	fake.translatePathMutex.RLock()
	defer fake.translatePathMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *ConfigManager) recordInvocation(key string, args []interface{}) {
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

var _ tms.ConfigManager = new(ConfigManager)
