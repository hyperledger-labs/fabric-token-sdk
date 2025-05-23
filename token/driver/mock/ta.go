// Code generated by counterfeiter. DO NOT EDIT.
package mock

import (
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TransferAction struct {
	ExtraSignersStub        func() []identity.Identity
	extraSignersMutex       sync.RWMutex
	extraSignersArgsForCall []struct {
	}
	extraSignersReturns struct {
		result1 []identity.Identity
	}
	extraSignersReturnsOnCall map[int]struct {
		result1 []identity.Identity
	}
	GetInputsStub        func() []*token.ID
	getInputsMutex       sync.RWMutex
	getInputsArgsForCall []struct {
	}
	getInputsReturns struct {
		result1 []*token.ID
	}
	getInputsReturnsOnCall map[int]struct {
		result1 []*token.ID
	}
	GetIssuerStub        func() identity.Identity
	getIssuerMutex       sync.RWMutex
	getIssuerArgsForCall []struct {
	}
	getIssuerReturns struct {
		result1 identity.Identity
	}
	getIssuerReturnsOnCall map[int]struct {
		result1 identity.Identity
	}
	GetMetadataStub        func() map[string][]byte
	getMetadataMutex       sync.RWMutex
	getMetadataArgsForCall []struct {
	}
	getMetadataReturns struct {
		result1 map[string][]byte
	}
	getMetadataReturnsOnCall map[int]struct {
		result1 map[string][]byte
	}
	GetOutputsStub        func() []driver.Output
	getOutputsMutex       sync.RWMutex
	getOutputsArgsForCall []struct {
	}
	getOutputsReturns struct {
		result1 []driver.Output
	}
	getOutputsReturnsOnCall map[int]struct {
		result1 []driver.Output
	}
	GetSerialNumbersStub        func() []string
	getSerialNumbersMutex       sync.RWMutex
	getSerialNumbersArgsForCall []struct {
	}
	getSerialNumbersReturns struct {
		result1 []string
	}
	getSerialNumbersReturnsOnCall map[int]struct {
		result1 []string
	}
	GetSerializedInputsStub        func() ([][]byte, error)
	getSerializedInputsMutex       sync.RWMutex
	getSerializedInputsArgsForCall []struct {
	}
	getSerializedInputsReturns struct {
		result1 [][]byte
		result2 error
	}
	getSerializedInputsReturnsOnCall map[int]struct {
		result1 [][]byte
		result2 error
	}
	GetSerializedOutputsStub        func() ([][]byte, error)
	getSerializedOutputsMutex       sync.RWMutex
	getSerializedOutputsArgsForCall []struct {
	}
	getSerializedOutputsReturns struct {
		result1 [][]byte
		result2 error
	}
	getSerializedOutputsReturnsOnCall map[int]struct {
		result1 [][]byte
		result2 error
	}
	IsGraphHidingStub        func() bool
	isGraphHidingMutex       sync.RWMutex
	isGraphHidingArgsForCall []struct {
	}
	isGraphHidingReturns struct {
		result1 bool
	}
	isGraphHidingReturnsOnCall map[int]struct {
		result1 bool
	}
	IsRedeemAtStub        func(int) bool
	isRedeemAtMutex       sync.RWMutex
	isRedeemAtArgsForCall []struct {
		arg1 int
	}
	isRedeemAtReturns struct {
		result1 bool
	}
	isRedeemAtReturnsOnCall map[int]struct {
		result1 bool
	}
	NumInputsStub        func() int
	numInputsMutex       sync.RWMutex
	numInputsArgsForCall []struct {
	}
	numInputsReturns struct {
		result1 int
	}
	numInputsReturnsOnCall map[int]struct {
		result1 int
	}
	NumOutputsStub        func() int
	numOutputsMutex       sync.RWMutex
	numOutputsArgsForCall []struct {
	}
	numOutputsReturns struct {
		result1 int
	}
	numOutputsReturnsOnCall map[int]struct {
		result1 int
	}
	SerializeStub        func() ([]byte, error)
	serializeMutex       sync.RWMutex
	serializeArgsForCall []struct {
	}
	serializeReturns struct {
		result1 []byte
		result2 error
	}
	serializeReturnsOnCall map[int]struct {
		result1 []byte
		result2 error
	}
	SerializeOutputAtStub        func(int) ([]byte, error)
	serializeOutputAtMutex       sync.RWMutex
	serializeOutputAtArgsForCall []struct {
		arg1 int
	}
	serializeOutputAtReturns struct {
		result1 []byte
		result2 error
	}
	serializeOutputAtReturnsOnCall map[int]struct {
		result1 []byte
		result2 error
	}
	ValidateStub        func() error
	validateMutex       sync.RWMutex
	validateArgsForCall []struct {
	}
	validateReturns struct {
		result1 error
	}
	validateReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *TransferAction) ExtraSigners() []identity.Identity {
	fake.extraSignersMutex.Lock()
	ret, specificReturn := fake.extraSignersReturnsOnCall[len(fake.extraSignersArgsForCall)]
	fake.extraSignersArgsForCall = append(fake.extraSignersArgsForCall, struct {
	}{})
	stub := fake.ExtraSignersStub
	fakeReturns := fake.extraSignersReturns
	fake.recordInvocation("ExtraSigners", []interface{}{})
	fake.extraSignersMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *TransferAction) ExtraSignersCallCount() int {
	fake.extraSignersMutex.RLock()
	defer fake.extraSignersMutex.RUnlock()
	return len(fake.extraSignersArgsForCall)
}

func (fake *TransferAction) ExtraSignersCalls(stub func() []identity.Identity) {
	fake.extraSignersMutex.Lock()
	defer fake.extraSignersMutex.Unlock()
	fake.ExtraSignersStub = stub
}

func (fake *TransferAction) ExtraSignersReturns(result1 []identity.Identity) {
	fake.extraSignersMutex.Lock()
	defer fake.extraSignersMutex.Unlock()
	fake.ExtraSignersStub = nil
	fake.extraSignersReturns = struct {
		result1 []identity.Identity
	}{result1}
}

func (fake *TransferAction) ExtraSignersReturnsOnCall(i int, result1 []identity.Identity) {
	fake.extraSignersMutex.Lock()
	defer fake.extraSignersMutex.Unlock()
	fake.ExtraSignersStub = nil
	if fake.extraSignersReturnsOnCall == nil {
		fake.extraSignersReturnsOnCall = make(map[int]struct {
			result1 []identity.Identity
		})
	}
	fake.extraSignersReturnsOnCall[i] = struct {
		result1 []identity.Identity
	}{result1}
}

func (fake *TransferAction) GetInputs() []*token.ID {
	fake.getInputsMutex.Lock()
	ret, specificReturn := fake.getInputsReturnsOnCall[len(fake.getInputsArgsForCall)]
	fake.getInputsArgsForCall = append(fake.getInputsArgsForCall, struct {
	}{})
	stub := fake.GetInputsStub
	fakeReturns := fake.getInputsReturns
	fake.recordInvocation("GetInputs", []interface{}{})
	fake.getInputsMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *TransferAction) GetInputsCallCount() int {
	fake.getInputsMutex.RLock()
	defer fake.getInputsMutex.RUnlock()
	return len(fake.getInputsArgsForCall)
}

func (fake *TransferAction) GetInputsCalls(stub func() []*token.ID) {
	fake.getInputsMutex.Lock()
	defer fake.getInputsMutex.Unlock()
	fake.GetInputsStub = stub
}

func (fake *TransferAction) GetInputsReturns(result1 []*token.ID) {
	fake.getInputsMutex.Lock()
	defer fake.getInputsMutex.Unlock()
	fake.GetInputsStub = nil
	fake.getInputsReturns = struct {
		result1 []*token.ID
	}{result1}
}

func (fake *TransferAction) GetInputsReturnsOnCall(i int, result1 []*token.ID) {
	fake.getInputsMutex.Lock()
	defer fake.getInputsMutex.Unlock()
	fake.GetInputsStub = nil
	if fake.getInputsReturnsOnCall == nil {
		fake.getInputsReturnsOnCall = make(map[int]struct {
			result1 []*token.ID
		})
	}
	fake.getInputsReturnsOnCall[i] = struct {
		result1 []*token.ID
	}{result1}
}

func (fake *TransferAction) GetIssuer() identity.Identity {
	fake.getIssuerMutex.Lock()
	ret, specificReturn := fake.getIssuerReturnsOnCall[len(fake.getIssuerArgsForCall)]
	fake.getIssuerArgsForCall = append(fake.getIssuerArgsForCall, struct {
	}{})
	stub := fake.GetIssuerStub
	fakeReturns := fake.getIssuerReturns
	fake.recordInvocation("GetIssuer", []interface{}{})
	fake.getIssuerMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *TransferAction) GetIssuerCallCount() int {
	fake.getIssuerMutex.RLock()
	defer fake.getIssuerMutex.RUnlock()
	return len(fake.getIssuerArgsForCall)
}

func (fake *TransferAction) GetIssuerCalls(stub func() identity.Identity) {
	fake.getIssuerMutex.Lock()
	defer fake.getIssuerMutex.Unlock()
	fake.GetIssuerStub = stub
}

func (fake *TransferAction) GetIssuerReturns(result1 identity.Identity) {
	fake.getIssuerMutex.Lock()
	defer fake.getIssuerMutex.Unlock()
	fake.GetIssuerStub = nil
	fake.getIssuerReturns = struct {
		result1 identity.Identity
	}{result1}
}

func (fake *TransferAction) GetIssuerReturnsOnCall(i int, result1 identity.Identity) {
	fake.getIssuerMutex.Lock()
	defer fake.getIssuerMutex.Unlock()
	fake.GetIssuerStub = nil
	if fake.getIssuerReturnsOnCall == nil {
		fake.getIssuerReturnsOnCall = make(map[int]struct {
			result1 identity.Identity
		})
	}
	fake.getIssuerReturnsOnCall[i] = struct {
		result1 identity.Identity
	}{result1}
}

func (fake *TransferAction) GetMetadata() map[string][]byte {
	fake.getMetadataMutex.Lock()
	ret, specificReturn := fake.getMetadataReturnsOnCall[len(fake.getMetadataArgsForCall)]
	fake.getMetadataArgsForCall = append(fake.getMetadataArgsForCall, struct {
	}{})
	stub := fake.GetMetadataStub
	fakeReturns := fake.getMetadataReturns
	fake.recordInvocation("GetMetadata", []interface{}{})
	fake.getMetadataMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *TransferAction) GetMetadataCallCount() int {
	fake.getMetadataMutex.RLock()
	defer fake.getMetadataMutex.RUnlock()
	return len(fake.getMetadataArgsForCall)
}

func (fake *TransferAction) GetMetadataCalls(stub func() map[string][]byte) {
	fake.getMetadataMutex.Lock()
	defer fake.getMetadataMutex.Unlock()
	fake.GetMetadataStub = stub
}

func (fake *TransferAction) GetMetadataReturns(result1 map[string][]byte) {
	fake.getMetadataMutex.Lock()
	defer fake.getMetadataMutex.Unlock()
	fake.GetMetadataStub = nil
	fake.getMetadataReturns = struct {
		result1 map[string][]byte
	}{result1}
}

func (fake *TransferAction) GetMetadataReturnsOnCall(i int, result1 map[string][]byte) {
	fake.getMetadataMutex.Lock()
	defer fake.getMetadataMutex.Unlock()
	fake.GetMetadataStub = nil
	if fake.getMetadataReturnsOnCall == nil {
		fake.getMetadataReturnsOnCall = make(map[int]struct {
			result1 map[string][]byte
		})
	}
	fake.getMetadataReturnsOnCall[i] = struct {
		result1 map[string][]byte
	}{result1}
}

func (fake *TransferAction) GetOutputs() []driver.Output {
	fake.getOutputsMutex.Lock()
	ret, specificReturn := fake.getOutputsReturnsOnCall[len(fake.getOutputsArgsForCall)]
	fake.getOutputsArgsForCall = append(fake.getOutputsArgsForCall, struct {
	}{})
	stub := fake.GetOutputsStub
	fakeReturns := fake.getOutputsReturns
	fake.recordInvocation("GetOutputs", []interface{}{})
	fake.getOutputsMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *TransferAction) GetOutputsCallCount() int {
	fake.getOutputsMutex.RLock()
	defer fake.getOutputsMutex.RUnlock()
	return len(fake.getOutputsArgsForCall)
}

func (fake *TransferAction) GetOutputsCalls(stub func() []driver.Output) {
	fake.getOutputsMutex.Lock()
	defer fake.getOutputsMutex.Unlock()
	fake.GetOutputsStub = stub
}

func (fake *TransferAction) GetOutputsReturns(result1 []driver.Output) {
	fake.getOutputsMutex.Lock()
	defer fake.getOutputsMutex.Unlock()
	fake.GetOutputsStub = nil
	fake.getOutputsReturns = struct {
		result1 []driver.Output
	}{result1}
}

func (fake *TransferAction) GetOutputsReturnsOnCall(i int, result1 []driver.Output) {
	fake.getOutputsMutex.Lock()
	defer fake.getOutputsMutex.Unlock()
	fake.GetOutputsStub = nil
	if fake.getOutputsReturnsOnCall == nil {
		fake.getOutputsReturnsOnCall = make(map[int]struct {
			result1 []driver.Output
		})
	}
	fake.getOutputsReturnsOnCall[i] = struct {
		result1 []driver.Output
	}{result1}
}

func (fake *TransferAction) GetSerialNumbers() []string {
	fake.getSerialNumbersMutex.Lock()
	ret, specificReturn := fake.getSerialNumbersReturnsOnCall[len(fake.getSerialNumbersArgsForCall)]
	fake.getSerialNumbersArgsForCall = append(fake.getSerialNumbersArgsForCall, struct {
	}{})
	stub := fake.GetSerialNumbersStub
	fakeReturns := fake.getSerialNumbersReturns
	fake.recordInvocation("GetSerialNumbers", []interface{}{})
	fake.getSerialNumbersMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *TransferAction) GetSerialNumbersCallCount() int {
	fake.getSerialNumbersMutex.RLock()
	defer fake.getSerialNumbersMutex.RUnlock()
	return len(fake.getSerialNumbersArgsForCall)
}

func (fake *TransferAction) GetSerialNumbersCalls(stub func() []string) {
	fake.getSerialNumbersMutex.Lock()
	defer fake.getSerialNumbersMutex.Unlock()
	fake.GetSerialNumbersStub = stub
}

func (fake *TransferAction) GetSerialNumbersReturns(result1 []string) {
	fake.getSerialNumbersMutex.Lock()
	defer fake.getSerialNumbersMutex.Unlock()
	fake.GetSerialNumbersStub = nil
	fake.getSerialNumbersReturns = struct {
		result1 []string
	}{result1}
}

func (fake *TransferAction) GetSerialNumbersReturnsOnCall(i int, result1 []string) {
	fake.getSerialNumbersMutex.Lock()
	defer fake.getSerialNumbersMutex.Unlock()
	fake.GetSerialNumbersStub = nil
	if fake.getSerialNumbersReturnsOnCall == nil {
		fake.getSerialNumbersReturnsOnCall = make(map[int]struct {
			result1 []string
		})
	}
	fake.getSerialNumbersReturnsOnCall[i] = struct {
		result1 []string
	}{result1}
}

func (fake *TransferAction) GetSerializedInputs() ([][]byte, error) {
	fake.getSerializedInputsMutex.Lock()
	ret, specificReturn := fake.getSerializedInputsReturnsOnCall[len(fake.getSerializedInputsArgsForCall)]
	fake.getSerializedInputsArgsForCall = append(fake.getSerializedInputsArgsForCall, struct {
	}{})
	stub := fake.GetSerializedInputsStub
	fakeReturns := fake.getSerializedInputsReturns
	fake.recordInvocation("GetSerializedInputs", []interface{}{})
	fake.getSerializedInputsMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *TransferAction) GetSerializedInputsCallCount() int {
	fake.getSerializedInputsMutex.RLock()
	defer fake.getSerializedInputsMutex.RUnlock()
	return len(fake.getSerializedInputsArgsForCall)
}

func (fake *TransferAction) GetSerializedInputsCalls(stub func() ([][]byte, error)) {
	fake.getSerializedInputsMutex.Lock()
	defer fake.getSerializedInputsMutex.Unlock()
	fake.GetSerializedInputsStub = stub
}

func (fake *TransferAction) GetSerializedInputsReturns(result1 [][]byte, result2 error) {
	fake.getSerializedInputsMutex.Lock()
	defer fake.getSerializedInputsMutex.Unlock()
	fake.GetSerializedInputsStub = nil
	fake.getSerializedInputsReturns = struct {
		result1 [][]byte
		result2 error
	}{result1, result2}
}

func (fake *TransferAction) GetSerializedInputsReturnsOnCall(i int, result1 [][]byte, result2 error) {
	fake.getSerializedInputsMutex.Lock()
	defer fake.getSerializedInputsMutex.Unlock()
	fake.GetSerializedInputsStub = nil
	if fake.getSerializedInputsReturnsOnCall == nil {
		fake.getSerializedInputsReturnsOnCall = make(map[int]struct {
			result1 [][]byte
			result2 error
		})
	}
	fake.getSerializedInputsReturnsOnCall[i] = struct {
		result1 [][]byte
		result2 error
	}{result1, result2}
}

func (fake *TransferAction) GetSerializedOutputs() ([][]byte, error) {
	fake.getSerializedOutputsMutex.Lock()
	ret, specificReturn := fake.getSerializedOutputsReturnsOnCall[len(fake.getSerializedOutputsArgsForCall)]
	fake.getSerializedOutputsArgsForCall = append(fake.getSerializedOutputsArgsForCall, struct {
	}{})
	stub := fake.GetSerializedOutputsStub
	fakeReturns := fake.getSerializedOutputsReturns
	fake.recordInvocation("GetSerializedOutputs", []interface{}{})
	fake.getSerializedOutputsMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *TransferAction) GetSerializedOutputsCallCount() int {
	fake.getSerializedOutputsMutex.RLock()
	defer fake.getSerializedOutputsMutex.RUnlock()
	return len(fake.getSerializedOutputsArgsForCall)
}

func (fake *TransferAction) GetSerializedOutputsCalls(stub func() ([][]byte, error)) {
	fake.getSerializedOutputsMutex.Lock()
	defer fake.getSerializedOutputsMutex.Unlock()
	fake.GetSerializedOutputsStub = stub
}

func (fake *TransferAction) GetSerializedOutputsReturns(result1 [][]byte, result2 error) {
	fake.getSerializedOutputsMutex.Lock()
	defer fake.getSerializedOutputsMutex.Unlock()
	fake.GetSerializedOutputsStub = nil
	fake.getSerializedOutputsReturns = struct {
		result1 [][]byte
		result2 error
	}{result1, result2}
}

func (fake *TransferAction) GetSerializedOutputsReturnsOnCall(i int, result1 [][]byte, result2 error) {
	fake.getSerializedOutputsMutex.Lock()
	defer fake.getSerializedOutputsMutex.Unlock()
	fake.GetSerializedOutputsStub = nil
	if fake.getSerializedOutputsReturnsOnCall == nil {
		fake.getSerializedOutputsReturnsOnCall = make(map[int]struct {
			result1 [][]byte
			result2 error
		})
	}
	fake.getSerializedOutputsReturnsOnCall[i] = struct {
		result1 [][]byte
		result2 error
	}{result1, result2}
}

func (fake *TransferAction) IsGraphHiding() bool {
	fake.isGraphHidingMutex.Lock()
	ret, specificReturn := fake.isGraphHidingReturnsOnCall[len(fake.isGraphHidingArgsForCall)]
	fake.isGraphHidingArgsForCall = append(fake.isGraphHidingArgsForCall, struct {
	}{})
	stub := fake.IsGraphHidingStub
	fakeReturns := fake.isGraphHidingReturns
	fake.recordInvocation("IsGraphHiding", []interface{}{})
	fake.isGraphHidingMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *TransferAction) IsGraphHidingCallCount() int {
	fake.isGraphHidingMutex.RLock()
	defer fake.isGraphHidingMutex.RUnlock()
	return len(fake.isGraphHidingArgsForCall)
}

func (fake *TransferAction) IsGraphHidingCalls(stub func() bool) {
	fake.isGraphHidingMutex.Lock()
	defer fake.isGraphHidingMutex.Unlock()
	fake.IsGraphHidingStub = stub
}

func (fake *TransferAction) IsGraphHidingReturns(result1 bool) {
	fake.isGraphHidingMutex.Lock()
	defer fake.isGraphHidingMutex.Unlock()
	fake.IsGraphHidingStub = nil
	fake.isGraphHidingReturns = struct {
		result1 bool
	}{result1}
}

func (fake *TransferAction) IsGraphHidingReturnsOnCall(i int, result1 bool) {
	fake.isGraphHidingMutex.Lock()
	defer fake.isGraphHidingMutex.Unlock()
	fake.IsGraphHidingStub = nil
	if fake.isGraphHidingReturnsOnCall == nil {
		fake.isGraphHidingReturnsOnCall = make(map[int]struct {
			result1 bool
		})
	}
	fake.isGraphHidingReturnsOnCall[i] = struct {
		result1 bool
	}{result1}
}

func (fake *TransferAction) IsRedeemAt(arg1 int) bool {
	fake.isRedeemAtMutex.Lock()
	ret, specificReturn := fake.isRedeemAtReturnsOnCall[len(fake.isRedeemAtArgsForCall)]
	fake.isRedeemAtArgsForCall = append(fake.isRedeemAtArgsForCall, struct {
		arg1 int
	}{arg1})
	stub := fake.IsRedeemAtStub
	fakeReturns := fake.isRedeemAtReturns
	fake.recordInvocation("IsRedeemAt", []interface{}{arg1})
	fake.isRedeemAtMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *TransferAction) IsRedeemAtCallCount() int {
	fake.isRedeemAtMutex.RLock()
	defer fake.isRedeemAtMutex.RUnlock()
	return len(fake.isRedeemAtArgsForCall)
}

func (fake *TransferAction) IsRedeemAtCalls(stub func(int) bool) {
	fake.isRedeemAtMutex.Lock()
	defer fake.isRedeemAtMutex.Unlock()
	fake.IsRedeemAtStub = stub
}

func (fake *TransferAction) IsRedeemAtArgsForCall(i int) int {
	fake.isRedeemAtMutex.RLock()
	defer fake.isRedeemAtMutex.RUnlock()
	argsForCall := fake.isRedeemAtArgsForCall[i]
	return argsForCall.arg1
}

func (fake *TransferAction) IsRedeemAtReturns(result1 bool) {
	fake.isRedeemAtMutex.Lock()
	defer fake.isRedeemAtMutex.Unlock()
	fake.IsRedeemAtStub = nil
	fake.isRedeemAtReturns = struct {
		result1 bool
	}{result1}
}

func (fake *TransferAction) IsRedeemAtReturnsOnCall(i int, result1 bool) {
	fake.isRedeemAtMutex.Lock()
	defer fake.isRedeemAtMutex.Unlock()
	fake.IsRedeemAtStub = nil
	if fake.isRedeemAtReturnsOnCall == nil {
		fake.isRedeemAtReturnsOnCall = make(map[int]struct {
			result1 bool
		})
	}
	fake.isRedeemAtReturnsOnCall[i] = struct {
		result1 bool
	}{result1}
}

func (fake *TransferAction) NumInputs() int {
	fake.numInputsMutex.Lock()
	ret, specificReturn := fake.numInputsReturnsOnCall[len(fake.numInputsArgsForCall)]
	fake.numInputsArgsForCall = append(fake.numInputsArgsForCall, struct {
	}{})
	stub := fake.NumInputsStub
	fakeReturns := fake.numInputsReturns
	fake.recordInvocation("NumInputs", []interface{}{})
	fake.numInputsMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *TransferAction) NumInputsCallCount() int {
	fake.numInputsMutex.RLock()
	defer fake.numInputsMutex.RUnlock()
	return len(fake.numInputsArgsForCall)
}

func (fake *TransferAction) NumInputsCalls(stub func() int) {
	fake.numInputsMutex.Lock()
	defer fake.numInputsMutex.Unlock()
	fake.NumInputsStub = stub
}

func (fake *TransferAction) NumInputsReturns(result1 int) {
	fake.numInputsMutex.Lock()
	defer fake.numInputsMutex.Unlock()
	fake.NumInputsStub = nil
	fake.numInputsReturns = struct {
		result1 int
	}{result1}
}

func (fake *TransferAction) NumInputsReturnsOnCall(i int, result1 int) {
	fake.numInputsMutex.Lock()
	defer fake.numInputsMutex.Unlock()
	fake.NumInputsStub = nil
	if fake.numInputsReturnsOnCall == nil {
		fake.numInputsReturnsOnCall = make(map[int]struct {
			result1 int
		})
	}
	fake.numInputsReturnsOnCall[i] = struct {
		result1 int
	}{result1}
}

func (fake *TransferAction) NumOutputs() int {
	fake.numOutputsMutex.Lock()
	ret, specificReturn := fake.numOutputsReturnsOnCall[len(fake.numOutputsArgsForCall)]
	fake.numOutputsArgsForCall = append(fake.numOutputsArgsForCall, struct {
	}{})
	stub := fake.NumOutputsStub
	fakeReturns := fake.numOutputsReturns
	fake.recordInvocation("NumOutputs", []interface{}{})
	fake.numOutputsMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *TransferAction) NumOutputsCallCount() int {
	fake.numOutputsMutex.RLock()
	defer fake.numOutputsMutex.RUnlock()
	return len(fake.numOutputsArgsForCall)
}

func (fake *TransferAction) NumOutputsCalls(stub func() int) {
	fake.numOutputsMutex.Lock()
	defer fake.numOutputsMutex.Unlock()
	fake.NumOutputsStub = stub
}

func (fake *TransferAction) NumOutputsReturns(result1 int) {
	fake.numOutputsMutex.Lock()
	defer fake.numOutputsMutex.Unlock()
	fake.NumOutputsStub = nil
	fake.numOutputsReturns = struct {
		result1 int
	}{result1}
}

func (fake *TransferAction) NumOutputsReturnsOnCall(i int, result1 int) {
	fake.numOutputsMutex.Lock()
	defer fake.numOutputsMutex.Unlock()
	fake.NumOutputsStub = nil
	if fake.numOutputsReturnsOnCall == nil {
		fake.numOutputsReturnsOnCall = make(map[int]struct {
			result1 int
		})
	}
	fake.numOutputsReturnsOnCall[i] = struct {
		result1 int
	}{result1}
}

func (fake *TransferAction) Serialize() ([]byte, error) {
	fake.serializeMutex.Lock()
	ret, specificReturn := fake.serializeReturnsOnCall[len(fake.serializeArgsForCall)]
	fake.serializeArgsForCall = append(fake.serializeArgsForCall, struct {
	}{})
	stub := fake.SerializeStub
	fakeReturns := fake.serializeReturns
	fake.recordInvocation("Serialize", []interface{}{})
	fake.serializeMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *TransferAction) SerializeCallCount() int {
	fake.serializeMutex.RLock()
	defer fake.serializeMutex.RUnlock()
	return len(fake.serializeArgsForCall)
}

func (fake *TransferAction) SerializeCalls(stub func() ([]byte, error)) {
	fake.serializeMutex.Lock()
	defer fake.serializeMutex.Unlock()
	fake.SerializeStub = stub
}

func (fake *TransferAction) SerializeReturns(result1 []byte, result2 error) {
	fake.serializeMutex.Lock()
	defer fake.serializeMutex.Unlock()
	fake.SerializeStub = nil
	fake.serializeReturns = struct {
		result1 []byte
		result2 error
	}{result1, result2}
}

func (fake *TransferAction) SerializeReturnsOnCall(i int, result1 []byte, result2 error) {
	fake.serializeMutex.Lock()
	defer fake.serializeMutex.Unlock()
	fake.SerializeStub = nil
	if fake.serializeReturnsOnCall == nil {
		fake.serializeReturnsOnCall = make(map[int]struct {
			result1 []byte
			result2 error
		})
	}
	fake.serializeReturnsOnCall[i] = struct {
		result1 []byte
		result2 error
	}{result1, result2}
}

func (fake *TransferAction) SerializeOutputAt(arg1 int) ([]byte, error) {
	fake.serializeOutputAtMutex.Lock()
	ret, specificReturn := fake.serializeOutputAtReturnsOnCall[len(fake.serializeOutputAtArgsForCall)]
	fake.serializeOutputAtArgsForCall = append(fake.serializeOutputAtArgsForCall, struct {
		arg1 int
	}{arg1})
	stub := fake.SerializeOutputAtStub
	fakeReturns := fake.serializeOutputAtReturns
	fake.recordInvocation("SerializeOutputAt", []interface{}{arg1})
	fake.serializeOutputAtMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *TransferAction) SerializeOutputAtCallCount() int {
	fake.serializeOutputAtMutex.RLock()
	defer fake.serializeOutputAtMutex.RUnlock()
	return len(fake.serializeOutputAtArgsForCall)
}

func (fake *TransferAction) SerializeOutputAtCalls(stub func(int) ([]byte, error)) {
	fake.serializeOutputAtMutex.Lock()
	defer fake.serializeOutputAtMutex.Unlock()
	fake.SerializeOutputAtStub = stub
}

func (fake *TransferAction) SerializeOutputAtArgsForCall(i int) int {
	fake.serializeOutputAtMutex.RLock()
	defer fake.serializeOutputAtMutex.RUnlock()
	argsForCall := fake.serializeOutputAtArgsForCall[i]
	return argsForCall.arg1
}

func (fake *TransferAction) SerializeOutputAtReturns(result1 []byte, result2 error) {
	fake.serializeOutputAtMutex.Lock()
	defer fake.serializeOutputAtMutex.Unlock()
	fake.SerializeOutputAtStub = nil
	fake.serializeOutputAtReturns = struct {
		result1 []byte
		result2 error
	}{result1, result2}
}

func (fake *TransferAction) SerializeOutputAtReturnsOnCall(i int, result1 []byte, result2 error) {
	fake.serializeOutputAtMutex.Lock()
	defer fake.serializeOutputAtMutex.Unlock()
	fake.SerializeOutputAtStub = nil
	if fake.serializeOutputAtReturnsOnCall == nil {
		fake.serializeOutputAtReturnsOnCall = make(map[int]struct {
			result1 []byte
			result2 error
		})
	}
	fake.serializeOutputAtReturnsOnCall[i] = struct {
		result1 []byte
		result2 error
	}{result1, result2}
}

func (fake *TransferAction) Validate() error {
	fake.validateMutex.Lock()
	ret, specificReturn := fake.validateReturnsOnCall[len(fake.validateArgsForCall)]
	fake.validateArgsForCall = append(fake.validateArgsForCall, struct {
	}{})
	stub := fake.ValidateStub
	fakeReturns := fake.validateReturns
	fake.recordInvocation("Validate", []interface{}{})
	fake.validateMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *TransferAction) ValidateCallCount() int {
	fake.validateMutex.RLock()
	defer fake.validateMutex.RUnlock()
	return len(fake.validateArgsForCall)
}

func (fake *TransferAction) ValidateCalls(stub func() error) {
	fake.validateMutex.Lock()
	defer fake.validateMutex.Unlock()
	fake.ValidateStub = stub
}

func (fake *TransferAction) ValidateReturns(result1 error) {
	fake.validateMutex.Lock()
	defer fake.validateMutex.Unlock()
	fake.ValidateStub = nil
	fake.validateReturns = struct {
		result1 error
	}{result1}
}

func (fake *TransferAction) ValidateReturnsOnCall(i int, result1 error) {
	fake.validateMutex.Lock()
	defer fake.validateMutex.Unlock()
	fake.ValidateStub = nil
	if fake.validateReturnsOnCall == nil {
		fake.validateReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.validateReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *TransferAction) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.extraSignersMutex.RLock()
	defer fake.extraSignersMutex.RUnlock()
	fake.getInputsMutex.RLock()
	defer fake.getInputsMutex.RUnlock()
	fake.getIssuerMutex.RLock()
	defer fake.getIssuerMutex.RUnlock()
	fake.getMetadataMutex.RLock()
	defer fake.getMetadataMutex.RUnlock()
	fake.getOutputsMutex.RLock()
	defer fake.getOutputsMutex.RUnlock()
	fake.getSerialNumbersMutex.RLock()
	defer fake.getSerialNumbersMutex.RUnlock()
	fake.getSerializedInputsMutex.RLock()
	defer fake.getSerializedInputsMutex.RUnlock()
	fake.getSerializedOutputsMutex.RLock()
	defer fake.getSerializedOutputsMutex.RUnlock()
	fake.isGraphHidingMutex.RLock()
	defer fake.isGraphHidingMutex.RUnlock()
	fake.isRedeemAtMutex.RLock()
	defer fake.isRedeemAtMutex.RUnlock()
	fake.numInputsMutex.RLock()
	defer fake.numInputsMutex.RUnlock()
	fake.numOutputsMutex.RLock()
	defer fake.numOutputsMutex.RUnlock()
	fake.serializeMutex.RLock()
	defer fake.serializeMutex.RUnlock()
	fake.serializeOutputAtMutex.RLock()
	defer fake.serializeOutputAtMutex.RUnlock()
	fake.validateMutex.RLock()
	defer fake.validateMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *TransferAction) recordInvocation(key string, args []interface{}) {
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

var _ driver.TransferAction = new(TransferAction)
