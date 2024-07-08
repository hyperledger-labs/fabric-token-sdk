// Code generated by counterfeiter. DO NOT EDIT.
package mock

import (
	"context"
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type QueryEngine struct {
	GetStatusStub        func(string) (int, string, error)
	getStatusMutex       sync.RWMutex
	getStatusArgsForCall []struct {
		arg1 string
	}
	getStatusReturns struct {
		result1 int
		result2 string
		result3 error
	}
	getStatusReturnsOnCall map[int]struct {
		result1 int
		result2 string
		result3 error
	}
	GetTokenInfoAndOutputsStub        func(context.Context, []*token.ID) ([]string, [][]byte, [][]byte, error)
	getTokenInfoAndOutputsMutex       sync.RWMutex
	getTokenInfoAndOutputsArgsForCall []struct {
		arg1 context.Context
		arg2 []*token.ID
	}
	getTokenInfoAndOutputsReturns struct {
		result1 []string
		result2 [][]byte
		result3 [][]byte
		result4 error
	}
	getTokenInfoAndOutputsReturnsOnCall map[int]struct {
		result1 []string
		result2 [][]byte
		result3 [][]byte
		result4 error
	}
	GetTokenInfosStub        func([]*token.ID) ([][]byte, error)
	getTokenInfosMutex       sync.RWMutex
	getTokenInfosArgsForCall []struct {
		arg1 []*token.ID
	}
	getTokenInfosReturns struct {
		result1 [][]byte
		result2 error
	}
	getTokenInfosReturnsOnCall map[int]struct {
		result1 [][]byte
		result2 error
	}
	GetTokenOutputsStub        func([]*token.ID, driver.QueryCallbackFunc) error
	getTokenOutputsMutex       sync.RWMutex
	getTokenOutputsArgsForCall []struct {
		arg1 []*token.ID
		arg2 driver.QueryCallbackFunc
	}
	getTokenOutputsReturns struct {
		result1 error
	}
	getTokenOutputsReturnsOnCall map[int]struct {
		result1 error
	}
	GetTokensStub        func(...*token.ID) ([]string, []*token.Token, error)
	getTokensMutex       sync.RWMutex
	getTokensArgsForCall []struct {
		arg1 []*token.ID
	}
	getTokensReturns struct {
		result1 []string
		result2 []*token.Token
		result3 error
	}
	getTokensReturnsOnCall map[int]struct {
		result1 []string
		result2 []*token.Token
		result3 error
	}
	IsMineStub        func(*token.ID) (bool, error)
	isMineMutex       sync.RWMutex
	isMineArgsForCall []struct {
		arg1 *token.ID
	}
	isMineReturns struct {
		result1 bool
		result2 error
	}
	isMineReturnsOnCall map[int]struct {
		result1 bool
		result2 error
	}
	IsPendingStub        func(*token.ID) (bool, error)
	isPendingMutex       sync.RWMutex
	isPendingArgsForCall []struct {
		arg1 *token.ID
	}
	isPendingReturns struct {
		result1 bool
		result2 error
	}
	isPendingReturnsOnCall map[int]struct {
		result1 bool
		result2 error
	}
	ListAuditTokensStub        func(...*token.ID) ([]*token.Token, error)
	listAuditTokensMutex       sync.RWMutex
	listAuditTokensArgsForCall []struct {
		arg1 []*token.ID
	}
	listAuditTokensReturns struct {
		result1 []*token.Token
		result2 error
	}
	listAuditTokensReturnsOnCall map[int]struct {
		result1 []*token.Token
		result2 error
	}
	ListHistoryIssuedTokensStub        func() (*token.IssuedTokens, error)
	listHistoryIssuedTokensMutex       sync.RWMutex
	listHistoryIssuedTokensArgsForCall []struct {
	}
	listHistoryIssuedTokensReturns struct {
		result1 *token.IssuedTokens
		result2 error
	}
	listHistoryIssuedTokensReturnsOnCall map[int]struct {
		result1 *token.IssuedTokens
		result2 error
	}
	ListUnspentTokensStub        func() (*token.UnspentTokens, error)
	listUnspentTokensMutex       sync.RWMutex
	listUnspentTokensArgsForCall []struct {
	}
	listUnspentTokensReturns struct {
		result1 *token.UnspentTokens
		result2 error
	}
	listUnspentTokensReturnsOnCall map[int]struct {
		result1 *token.UnspentTokens
		result2 error
	}
	PublicParamsStub        func() ([]byte, error)
	publicParamsMutex       sync.RWMutex
	publicParamsArgsForCall []struct {
	}
	publicParamsReturns struct {
		result1 []byte
		result2 error
	}
	publicParamsReturnsOnCall map[int]struct {
		result1 []byte
		result2 error
	}
	UnspentTokensIteratorStub        func() (driver.UnspentTokensIterator, error)
	unspentTokensIteratorMutex       sync.RWMutex
	unspentTokensIteratorArgsForCall []struct {
	}
	unspentTokensIteratorReturns struct {
		result1 driver.UnspentTokensIterator
		result2 error
	}
	unspentTokensIteratorReturnsOnCall map[int]struct {
		result1 driver.UnspentTokensIterator
		result2 error
	}
	UnspentTokensIteratorByStub        func(context.Context, string, string) (driver.UnspentTokensIterator, error)
	unspentTokensIteratorByMutex       sync.RWMutex
	unspentTokensIteratorByArgsForCall []struct {
		arg1 context.Context
		arg2 string
		arg3 string
	}
	unspentTokensIteratorByReturns struct {
		result1 driver.UnspentTokensIterator
		result2 error
	}
	unspentTokensIteratorByReturnsOnCall map[int]struct {
		result1 driver.UnspentTokensIterator
		result2 error
	}
	WhoDeletedTokensStub        func(...*token.ID) ([]string, []bool, error)
	whoDeletedTokensMutex       sync.RWMutex
	whoDeletedTokensArgsForCall []struct {
		arg1 []*token.ID
	}
	whoDeletedTokensReturns struct {
		result1 []string
		result2 []bool
		result3 error
	}
	whoDeletedTokensReturnsOnCall map[int]struct {
		result1 []string
		result2 []bool
		result3 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *QueryEngine) GetStatus(arg1 string) (int, string, error) {
	fake.getStatusMutex.Lock()
	ret, specificReturn := fake.getStatusReturnsOnCall[len(fake.getStatusArgsForCall)]
	fake.getStatusArgsForCall = append(fake.getStatusArgsForCall, struct {
		arg1 string
	}{arg1})
	stub := fake.GetStatusStub
	fakeReturns := fake.getStatusReturns
	fake.recordInvocation("GetStatus", []interface{}{arg1})
	fake.getStatusMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3
}

func (fake *QueryEngine) GetStatusCallCount() int {
	fake.getStatusMutex.RLock()
	defer fake.getStatusMutex.RUnlock()
	return len(fake.getStatusArgsForCall)
}

func (fake *QueryEngine) GetStatusCalls(stub func(string) (int, string, error)) {
	fake.getStatusMutex.Lock()
	defer fake.getStatusMutex.Unlock()
	fake.GetStatusStub = stub
}

func (fake *QueryEngine) GetStatusArgsForCall(i int) string {
	fake.getStatusMutex.RLock()
	defer fake.getStatusMutex.RUnlock()
	argsForCall := fake.getStatusArgsForCall[i]
	return argsForCall.arg1
}

func (fake *QueryEngine) GetStatusReturns(result1 int, result2 string, result3 error) {
	fake.getStatusMutex.Lock()
	defer fake.getStatusMutex.Unlock()
	fake.GetStatusStub = nil
	fake.getStatusReturns = struct {
		result1 int
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *QueryEngine) GetStatusReturnsOnCall(i int, result1 int, result2 string, result3 error) {
	fake.getStatusMutex.Lock()
	defer fake.getStatusMutex.Unlock()
	fake.GetStatusStub = nil
	if fake.getStatusReturnsOnCall == nil {
		fake.getStatusReturnsOnCall = make(map[int]struct {
			result1 int
			result2 string
			result3 error
		})
	}
	fake.getStatusReturnsOnCall[i] = struct {
		result1 int
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *QueryEngine) GetTokenInfoAndOutputs(arg1 context.Context, arg2 []*token.ID) ([]string, [][]byte, [][]byte, error) {
	var arg2Copy []*token.ID
	if arg2 != nil {
		arg2Copy = make([]*token.ID, len(arg2))
		copy(arg2Copy, arg2)
	}
	fake.getTokenInfoAndOutputsMutex.Lock()
	ret, specificReturn := fake.getTokenInfoAndOutputsReturnsOnCall[len(fake.getTokenInfoAndOutputsArgsForCall)]
	fake.getTokenInfoAndOutputsArgsForCall = append(fake.getTokenInfoAndOutputsArgsForCall, struct {
		arg1 context.Context
		arg2 []*token.ID
	}{arg1, arg2Copy})
	stub := fake.GetTokenInfoAndOutputsStub
	fakeReturns := fake.getTokenInfoAndOutputsReturns
	fake.recordInvocation("GetTokenInfoAndOutputs", []interface{}{arg1, arg2Copy})
	fake.getTokenInfoAndOutputsMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3, ret.result4
	}
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3, fakeReturns.result4
}

func (fake *QueryEngine) GetTokenInfoAndOutputsCallCount() int {
	fake.getTokenInfoAndOutputsMutex.RLock()
	defer fake.getTokenInfoAndOutputsMutex.RUnlock()
	return len(fake.getTokenInfoAndOutputsArgsForCall)
}

func (fake *QueryEngine) GetTokenInfoAndOutputsCalls(stub func(context.Context, []*token.ID) ([]string, [][]byte, [][]byte, error)) {
	fake.getTokenInfoAndOutputsMutex.Lock()
	defer fake.getTokenInfoAndOutputsMutex.Unlock()
	fake.GetTokenInfoAndOutputsStub = stub
}

func (fake *QueryEngine) GetTokenInfoAndOutputsArgsForCall(i int) (context.Context, []*token.ID) {
	fake.getTokenInfoAndOutputsMutex.RLock()
	defer fake.getTokenInfoAndOutputsMutex.RUnlock()
	argsForCall := fake.getTokenInfoAndOutputsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *QueryEngine) GetTokenInfoAndOutputsReturns(result1 []string, result2 [][]byte, result3 [][]byte, result4 error) {
	fake.getTokenInfoAndOutputsMutex.Lock()
	defer fake.getTokenInfoAndOutputsMutex.Unlock()
	fake.GetTokenInfoAndOutputsStub = nil
	fake.getTokenInfoAndOutputsReturns = struct {
		result1 []string
		result2 [][]byte
		result3 [][]byte
		result4 error
	}{result1, result2, result3, result4}
}

func (fake *QueryEngine) GetTokenInfoAndOutputsReturnsOnCall(i int, result1 []string, result2 [][]byte, result3 [][]byte, result4 error) {
	fake.getTokenInfoAndOutputsMutex.Lock()
	defer fake.getTokenInfoAndOutputsMutex.Unlock()
	fake.GetTokenInfoAndOutputsStub = nil
	if fake.getTokenInfoAndOutputsReturnsOnCall == nil {
		fake.getTokenInfoAndOutputsReturnsOnCall = make(map[int]struct {
			result1 []string
			result2 [][]byte
			result3 [][]byte
			result4 error
		})
	}
	fake.getTokenInfoAndOutputsReturnsOnCall[i] = struct {
		result1 []string
		result2 [][]byte
		result3 [][]byte
		result4 error
	}{result1, result2, result3, result4}
}

func (fake *QueryEngine) GetTokenInfos(arg1 []*token.ID) ([][]byte, error) {
	var arg1Copy []*token.ID
	if arg1 != nil {
		arg1Copy = make([]*token.ID, len(arg1))
		copy(arg1Copy, arg1)
	}
	fake.getTokenInfosMutex.Lock()
	ret, specificReturn := fake.getTokenInfosReturnsOnCall[len(fake.getTokenInfosArgsForCall)]
	fake.getTokenInfosArgsForCall = append(fake.getTokenInfosArgsForCall, struct {
		arg1 []*token.ID
	}{arg1Copy})
	stub := fake.GetTokenInfosStub
	fakeReturns := fake.getTokenInfosReturns
	fake.recordInvocation("GetTokenInfos", []interface{}{arg1Copy})
	fake.getTokenInfosMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *QueryEngine) GetTokenInfosCallCount() int {
	fake.getTokenInfosMutex.RLock()
	defer fake.getTokenInfosMutex.RUnlock()
	return len(fake.getTokenInfosArgsForCall)
}

func (fake *QueryEngine) GetTokenInfosCalls(stub func([]*token.ID) ([][]byte, error)) {
	fake.getTokenInfosMutex.Lock()
	defer fake.getTokenInfosMutex.Unlock()
	fake.GetTokenInfosStub = stub
}

func (fake *QueryEngine) GetTokenInfosArgsForCall(i int) []*token.ID {
	fake.getTokenInfosMutex.RLock()
	defer fake.getTokenInfosMutex.RUnlock()
	argsForCall := fake.getTokenInfosArgsForCall[i]
	return argsForCall.arg1
}

func (fake *QueryEngine) GetTokenInfosReturns(result1 [][]byte, result2 error) {
	fake.getTokenInfosMutex.Lock()
	defer fake.getTokenInfosMutex.Unlock()
	fake.GetTokenInfosStub = nil
	fake.getTokenInfosReturns = struct {
		result1 [][]byte
		result2 error
	}{result1, result2}
}

func (fake *QueryEngine) GetTokenInfosReturnsOnCall(i int, result1 [][]byte, result2 error) {
	fake.getTokenInfosMutex.Lock()
	defer fake.getTokenInfosMutex.Unlock()
	fake.GetTokenInfosStub = nil
	if fake.getTokenInfosReturnsOnCall == nil {
		fake.getTokenInfosReturnsOnCall = make(map[int]struct {
			result1 [][]byte
			result2 error
		})
	}
	fake.getTokenInfosReturnsOnCall[i] = struct {
		result1 [][]byte
		result2 error
	}{result1, result2}
}

func (fake *QueryEngine) GetTokenOutputs(arg1 []*token.ID, arg2 driver.QueryCallbackFunc) error {
	var arg1Copy []*token.ID
	if arg1 != nil {
		arg1Copy = make([]*token.ID, len(arg1))
		copy(arg1Copy, arg1)
	}
	fake.getTokenOutputsMutex.Lock()
	ret, specificReturn := fake.getTokenOutputsReturnsOnCall[len(fake.getTokenOutputsArgsForCall)]
	fake.getTokenOutputsArgsForCall = append(fake.getTokenOutputsArgsForCall, struct {
		arg1 []*token.ID
		arg2 driver.QueryCallbackFunc
	}{arg1Copy, arg2})
	stub := fake.GetTokenOutputsStub
	fakeReturns := fake.getTokenOutputsReturns
	fake.recordInvocation("GetTokenOutputs", []interface{}{arg1Copy, arg2})
	fake.getTokenOutputsMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *QueryEngine) GetTokenOutputsCallCount() int {
	fake.getTokenOutputsMutex.RLock()
	defer fake.getTokenOutputsMutex.RUnlock()
	return len(fake.getTokenOutputsArgsForCall)
}

func (fake *QueryEngine) GetTokenOutputsCalls(stub func([]*token.ID, driver.QueryCallbackFunc) error) {
	fake.getTokenOutputsMutex.Lock()
	defer fake.getTokenOutputsMutex.Unlock()
	fake.GetTokenOutputsStub = stub
}

func (fake *QueryEngine) GetTokenOutputsArgsForCall(i int) ([]*token.ID, driver.QueryCallbackFunc) {
	fake.getTokenOutputsMutex.RLock()
	defer fake.getTokenOutputsMutex.RUnlock()
	argsForCall := fake.getTokenOutputsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *QueryEngine) GetTokenOutputsReturns(result1 error) {
	fake.getTokenOutputsMutex.Lock()
	defer fake.getTokenOutputsMutex.Unlock()
	fake.GetTokenOutputsStub = nil
	fake.getTokenOutputsReturns = struct {
		result1 error
	}{result1}
}

func (fake *QueryEngine) GetTokenOutputsReturnsOnCall(i int, result1 error) {
	fake.getTokenOutputsMutex.Lock()
	defer fake.getTokenOutputsMutex.Unlock()
	fake.GetTokenOutputsStub = nil
	if fake.getTokenOutputsReturnsOnCall == nil {
		fake.getTokenOutputsReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.getTokenOutputsReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *QueryEngine) GetTokens(arg1 ...*token.ID) ([]string, []*token.Token, error) {
	fake.getTokensMutex.Lock()
	ret, specificReturn := fake.getTokensReturnsOnCall[len(fake.getTokensArgsForCall)]
	fake.getTokensArgsForCall = append(fake.getTokensArgsForCall, struct {
		arg1 []*token.ID
	}{arg1})
	stub := fake.GetTokensStub
	fakeReturns := fake.getTokensReturns
	fake.recordInvocation("GetTokens", []interface{}{arg1})
	fake.getTokensMutex.Unlock()
	if stub != nil {
		return stub(arg1...)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3
}

func (fake *QueryEngine) GetTokensCallCount() int {
	fake.getTokensMutex.RLock()
	defer fake.getTokensMutex.RUnlock()
	return len(fake.getTokensArgsForCall)
}

func (fake *QueryEngine) GetTokensCalls(stub func(...*token.ID) ([]string, []*token.Token, error)) {
	fake.getTokensMutex.Lock()
	defer fake.getTokensMutex.Unlock()
	fake.GetTokensStub = stub
}

func (fake *QueryEngine) GetTokensArgsForCall(i int) []*token.ID {
	fake.getTokensMutex.RLock()
	defer fake.getTokensMutex.RUnlock()
	argsForCall := fake.getTokensArgsForCall[i]
	return argsForCall.arg1
}

func (fake *QueryEngine) GetTokensReturns(result1 []string, result2 []*token.Token, result3 error) {
	fake.getTokensMutex.Lock()
	defer fake.getTokensMutex.Unlock()
	fake.GetTokensStub = nil
	fake.getTokensReturns = struct {
		result1 []string
		result2 []*token.Token
		result3 error
	}{result1, result2, result3}
}

func (fake *QueryEngine) GetTokensReturnsOnCall(i int, result1 []string, result2 []*token.Token, result3 error) {
	fake.getTokensMutex.Lock()
	defer fake.getTokensMutex.Unlock()
	fake.GetTokensStub = nil
	if fake.getTokensReturnsOnCall == nil {
		fake.getTokensReturnsOnCall = make(map[int]struct {
			result1 []string
			result2 []*token.Token
			result3 error
		})
	}
	fake.getTokensReturnsOnCall[i] = struct {
		result1 []string
		result2 []*token.Token
		result3 error
	}{result1, result2, result3}
}

func (fake *QueryEngine) IsMine(arg1 *token.ID) (bool, error) {
	fake.isMineMutex.Lock()
	ret, specificReturn := fake.isMineReturnsOnCall[len(fake.isMineArgsForCall)]
	fake.isMineArgsForCall = append(fake.isMineArgsForCall, struct {
		arg1 *token.ID
	}{arg1})
	stub := fake.IsMineStub
	fakeReturns := fake.isMineReturns
	fake.recordInvocation("IsMine", []interface{}{arg1})
	fake.isMineMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *QueryEngine) IsMineCallCount() int {
	fake.isMineMutex.RLock()
	defer fake.isMineMutex.RUnlock()
	return len(fake.isMineArgsForCall)
}

func (fake *QueryEngine) IsMineCalls(stub func(*token.ID) (bool, error)) {
	fake.isMineMutex.Lock()
	defer fake.isMineMutex.Unlock()
	fake.IsMineStub = stub
}

func (fake *QueryEngine) IsMineArgsForCall(i int) *token.ID {
	fake.isMineMutex.RLock()
	defer fake.isMineMutex.RUnlock()
	argsForCall := fake.isMineArgsForCall[i]
	return argsForCall.arg1
}

func (fake *QueryEngine) IsMineReturns(result1 bool, result2 error) {
	fake.isMineMutex.Lock()
	defer fake.isMineMutex.Unlock()
	fake.IsMineStub = nil
	fake.isMineReturns = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *QueryEngine) IsMineReturnsOnCall(i int, result1 bool, result2 error) {
	fake.isMineMutex.Lock()
	defer fake.isMineMutex.Unlock()
	fake.IsMineStub = nil
	if fake.isMineReturnsOnCall == nil {
		fake.isMineReturnsOnCall = make(map[int]struct {
			result1 bool
			result2 error
		})
	}
	fake.isMineReturnsOnCall[i] = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *QueryEngine) IsPending(arg1 *token.ID) (bool, error) {
	fake.isPendingMutex.Lock()
	ret, specificReturn := fake.isPendingReturnsOnCall[len(fake.isPendingArgsForCall)]
	fake.isPendingArgsForCall = append(fake.isPendingArgsForCall, struct {
		arg1 *token.ID
	}{arg1})
	stub := fake.IsPendingStub
	fakeReturns := fake.isPendingReturns
	fake.recordInvocation("IsPending", []interface{}{arg1})
	fake.isPendingMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *QueryEngine) IsPendingCallCount() int {
	fake.isPendingMutex.RLock()
	defer fake.isPendingMutex.RUnlock()
	return len(fake.isPendingArgsForCall)
}

func (fake *QueryEngine) IsPendingCalls(stub func(*token.ID) (bool, error)) {
	fake.isPendingMutex.Lock()
	defer fake.isPendingMutex.Unlock()
	fake.IsPendingStub = stub
}

func (fake *QueryEngine) IsPendingArgsForCall(i int) *token.ID {
	fake.isPendingMutex.RLock()
	defer fake.isPendingMutex.RUnlock()
	argsForCall := fake.isPendingArgsForCall[i]
	return argsForCall.arg1
}

func (fake *QueryEngine) IsPendingReturns(result1 bool, result2 error) {
	fake.isPendingMutex.Lock()
	defer fake.isPendingMutex.Unlock()
	fake.IsPendingStub = nil
	fake.isPendingReturns = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *QueryEngine) IsPendingReturnsOnCall(i int, result1 bool, result2 error) {
	fake.isPendingMutex.Lock()
	defer fake.isPendingMutex.Unlock()
	fake.IsPendingStub = nil
	if fake.isPendingReturnsOnCall == nil {
		fake.isPendingReturnsOnCall = make(map[int]struct {
			result1 bool
			result2 error
		})
	}
	fake.isPendingReturnsOnCall[i] = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *QueryEngine) ListAuditTokens(arg1 ...*token.ID) ([]*token.Token, error) {
	fake.listAuditTokensMutex.Lock()
	ret, specificReturn := fake.listAuditTokensReturnsOnCall[len(fake.listAuditTokensArgsForCall)]
	fake.listAuditTokensArgsForCall = append(fake.listAuditTokensArgsForCall, struct {
		arg1 []*token.ID
	}{arg1})
	stub := fake.ListAuditTokensStub
	fakeReturns := fake.listAuditTokensReturns
	fake.recordInvocation("ListAuditTokens", []interface{}{arg1})
	fake.listAuditTokensMutex.Unlock()
	if stub != nil {
		return stub(arg1...)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *QueryEngine) ListAuditTokensCallCount() int {
	fake.listAuditTokensMutex.RLock()
	defer fake.listAuditTokensMutex.RUnlock()
	return len(fake.listAuditTokensArgsForCall)
}

func (fake *QueryEngine) ListAuditTokensCalls(stub func(...*token.ID) ([]*token.Token, error)) {
	fake.listAuditTokensMutex.Lock()
	defer fake.listAuditTokensMutex.Unlock()
	fake.ListAuditTokensStub = stub
}

func (fake *QueryEngine) ListAuditTokensArgsForCall(i int) []*token.ID {
	fake.listAuditTokensMutex.RLock()
	defer fake.listAuditTokensMutex.RUnlock()
	argsForCall := fake.listAuditTokensArgsForCall[i]
	return argsForCall.arg1
}

func (fake *QueryEngine) ListAuditTokensReturns(result1 []*token.Token, result2 error) {
	fake.listAuditTokensMutex.Lock()
	defer fake.listAuditTokensMutex.Unlock()
	fake.ListAuditTokensStub = nil
	fake.listAuditTokensReturns = struct {
		result1 []*token.Token
		result2 error
	}{result1, result2}
}

func (fake *QueryEngine) ListAuditTokensReturnsOnCall(i int, result1 []*token.Token, result2 error) {
	fake.listAuditTokensMutex.Lock()
	defer fake.listAuditTokensMutex.Unlock()
	fake.ListAuditTokensStub = nil
	if fake.listAuditTokensReturnsOnCall == nil {
		fake.listAuditTokensReturnsOnCall = make(map[int]struct {
			result1 []*token.Token
			result2 error
		})
	}
	fake.listAuditTokensReturnsOnCall[i] = struct {
		result1 []*token.Token
		result2 error
	}{result1, result2}
}

func (fake *QueryEngine) ListHistoryIssuedTokens() (*token.IssuedTokens, error) {
	fake.listHistoryIssuedTokensMutex.Lock()
	ret, specificReturn := fake.listHistoryIssuedTokensReturnsOnCall[len(fake.listHistoryIssuedTokensArgsForCall)]
	fake.listHistoryIssuedTokensArgsForCall = append(fake.listHistoryIssuedTokensArgsForCall, struct {
	}{})
	stub := fake.ListHistoryIssuedTokensStub
	fakeReturns := fake.listHistoryIssuedTokensReturns
	fake.recordInvocation("ListHistoryIssuedTokens", []interface{}{})
	fake.listHistoryIssuedTokensMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *QueryEngine) ListHistoryIssuedTokensCallCount() int {
	fake.listHistoryIssuedTokensMutex.RLock()
	defer fake.listHistoryIssuedTokensMutex.RUnlock()
	return len(fake.listHistoryIssuedTokensArgsForCall)
}

func (fake *QueryEngine) ListHistoryIssuedTokensCalls(stub func() (*token.IssuedTokens, error)) {
	fake.listHistoryIssuedTokensMutex.Lock()
	defer fake.listHistoryIssuedTokensMutex.Unlock()
	fake.ListHistoryIssuedTokensStub = stub
}

func (fake *QueryEngine) ListHistoryIssuedTokensReturns(result1 *token.IssuedTokens, result2 error) {
	fake.listHistoryIssuedTokensMutex.Lock()
	defer fake.listHistoryIssuedTokensMutex.Unlock()
	fake.ListHistoryIssuedTokensStub = nil
	fake.listHistoryIssuedTokensReturns = struct {
		result1 *token.IssuedTokens
		result2 error
	}{result1, result2}
}

func (fake *QueryEngine) ListHistoryIssuedTokensReturnsOnCall(i int, result1 *token.IssuedTokens, result2 error) {
	fake.listHistoryIssuedTokensMutex.Lock()
	defer fake.listHistoryIssuedTokensMutex.Unlock()
	fake.ListHistoryIssuedTokensStub = nil
	if fake.listHistoryIssuedTokensReturnsOnCall == nil {
		fake.listHistoryIssuedTokensReturnsOnCall = make(map[int]struct {
			result1 *token.IssuedTokens
			result2 error
		})
	}
	fake.listHistoryIssuedTokensReturnsOnCall[i] = struct {
		result1 *token.IssuedTokens
		result2 error
	}{result1, result2}
}

func (fake *QueryEngine) ListUnspentTokens() (*token.UnspentTokens, error) {
	fake.listUnspentTokensMutex.Lock()
	ret, specificReturn := fake.listUnspentTokensReturnsOnCall[len(fake.listUnspentTokensArgsForCall)]
	fake.listUnspentTokensArgsForCall = append(fake.listUnspentTokensArgsForCall, struct {
	}{})
	stub := fake.ListUnspentTokensStub
	fakeReturns := fake.listUnspentTokensReturns
	fake.recordInvocation("ListUnspentTokens", []interface{}{})
	fake.listUnspentTokensMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *QueryEngine) ListUnspentTokensCallCount() int {
	fake.listUnspentTokensMutex.RLock()
	defer fake.listUnspentTokensMutex.RUnlock()
	return len(fake.listUnspentTokensArgsForCall)
}

func (fake *QueryEngine) ListUnspentTokensCalls(stub func() (*token.UnspentTokens, error)) {
	fake.listUnspentTokensMutex.Lock()
	defer fake.listUnspentTokensMutex.Unlock()
	fake.ListUnspentTokensStub = stub
}

func (fake *QueryEngine) ListUnspentTokensReturns(result1 *token.UnspentTokens, result2 error) {
	fake.listUnspentTokensMutex.Lock()
	defer fake.listUnspentTokensMutex.Unlock()
	fake.ListUnspentTokensStub = nil
	fake.listUnspentTokensReturns = struct {
		result1 *token.UnspentTokens
		result2 error
	}{result1, result2}
}

func (fake *QueryEngine) ListUnspentTokensReturnsOnCall(i int, result1 *token.UnspentTokens, result2 error) {
	fake.listUnspentTokensMutex.Lock()
	defer fake.listUnspentTokensMutex.Unlock()
	fake.ListUnspentTokensStub = nil
	if fake.listUnspentTokensReturnsOnCall == nil {
		fake.listUnspentTokensReturnsOnCall = make(map[int]struct {
			result1 *token.UnspentTokens
			result2 error
		})
	}
	fake.listUnspentTokensReturnsOnCall[i] = struct {
		result1 *token.UnspentTokens
		result2 error
	}{result1, result2}
}

func (fake *QueryEngine) PublicParams() ([]byte, error) {
	fake.publicParamsMutex.Lock()
	ret, specificReturn := fake.publicParamsReturnsOnCall[len(fake.publicParamsArgsForCall)]
	fake.publicParamsArgsForCall = append(fake.publicParamsArgsForCall, struct {
	}{})
	stub := fake.PublicParamsStub
	fakeReturns := fake.publicParamsReturns
	fake.recordInvocation("PublicParams", []interface{}{})
	fake.publicParamsMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *QueryEngine) PublicParamsCallCount() int {
	fake.publicParamsMutex.RLock()
	defer fake.publicParamsMutex.RUnlock()
	return len(fake.publicParamsArgsForCall)
}

func (fake *QueryEngine) PublicParamsCalls(stub func() ([]byte, error)) {
	fake.publicParamsMutex.Lock()
	defer fake.publicParamsMutex.Unlock()
	fake.PublicParamsStub = stub
}

func (fake *QueryEngine) PublicParamsReturns(result1 []byte, result2 error) {
	fake.publicParamsMutex.Lock()
	defer fake.publicParamsMutex.Unlock()
	fake.PublicParamsStub = nil
	fake.publicParamsReturns = struct {
		result1 []byte
		result2 error
	}{result1, result2}
}

func (fake *QueryEngine) PublicParamsReturnsOnCall(i int, result1 []byte, result2 error) {
	fake.publicParamsMutex.Lock()
	defer fake.publicParamsMutex.Unlock()
	fake.PublicParamsStub = nil
	if fake.publicParamsReturnsOnCall == nil {
		fake.publicParamsReturnsOnCall = make(map[int]struct {
			result1 []byte
			result2 error
		})
	}
	fake.publicParamsReturnsOnCall[i] = struct {
		result1 []byte
		result2 error
	}{result1, result2}
}

func (fake *QueryEngine) UnspentTokensIterator() (driver.UnspentTokensIterator, error) {
	fake.unspentTokensIteratorMutex.Lock()
	ret, specificReturn := fake.unspentTokensIteratorReturnsOnCall[len(fake.unspentTokensIteratorArgsForCall)]
	fake.unspentTokensIteratorArgsForCall = append(fake.unspentTokensIteratorArgsForCall, struct {
	}{})
	stub := fake.UnspentTokensIteratorStub
	fakeReturns := fake.unspentTokensIteratorReturns
	fake.recordInvocation("UnspentTokensIterator", []interface{}{})
	fake.unspentTokensIteratorMutex.Unlock()
	if stub != nil {
		return stub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *QueryEngine) UnspentTokensIteratorCallCount() int {
	fake.unspentTokensIteratorMutex.RLock()
	defer fake.unspentTokensIteratorMutex.RUnlock()
	return len(fake.unspentTokensIteratorArgsForCall)
}

func (fake *QueryEngine) UnspentTokensIteratorCalls(stub func() (driver.UnspentTokensIterator, error)) {
	fake.unspentTokensIteratorMutex.Lock()
	defer fake.unspentTokensIteratorMutex.Unlock()
	fake.UnspentTokensIteratorStub = stub
}

func (fake *QueryEngine) UnspentTokensIteratorReturns(result1 driver.UnspentTokensIterator, result2 error) {
	fake.unspentTokensIteratorMutex.Lock()
	defer fake.unspentTokensIteratorMutex.Unlock()
	fake.UnspentTokensIteratorStub = nil
	fake.unspentTokensIteratorReturns = struct {
		result1 driver.UnspentTokensIterator
		result2 error
	}{result1, result2}
}

func (fake *QueryEngine) UnspentTokensIteratorReturnsOnCall(i int, result1 driver.UnspentTokensIterator, result2 error) {
	fake.unspentTokensIteratorMutex.Lock()
	defer fake.unspentTokensIteratorMutex.Unlock()
	fake.UnspentTokensIteratorStub = nil
	if fake.unspentTokensIteratorReturnsOnCall == nil {
		fake.unspentTokensIteratorReturnsOnCall = make(map[int]struct {
			result1 driver.UnspentTokensIterator
			result2 error
		})
	}
	fake.unspentTokensIteratorReturnsOnCall[i] = struct {
		result1 driver.UnspentTokensIterator
		result2 error
	}{result1, result2}
}

func (fake *QueryEngine) UnspentTokensIteratorBy(arg1 context.Context, arg2 string, arg3 string) (driver.UnspentTokensIterator, error) {
	fake.unspentTokensIteratorByMutex.Lock()
	ret, specificReturn := fake.unspentTokensIteratorByReturnsOnCall[len(fake.unspentTokensIteratorByArgsForCall)]
	fake.unspentTokensIteratorByArgsForCall = append(fake.unspentTokensIteratorByArgsForCall, struct {
		arg1 context.Context
		arg2 string
		arg3 string
	}{arg1, arg2, arg3})
	stub := fake.UnspentTokensIteratorByStub
	fakeReturns := fake.unspentTokensIteratorByReturns
	fake.recordInvocation("UnspentTokensIteratorBy", []interface{}{arg1, arg2, arg3})
	fake.unspentTokensIteratorByMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *QueryEngine) UnspentTokensIteratorByCallCount() int {
	fake.unspentTokensIteratorByMutex.RLock()
	defer fake.unspentTokensIteratorByMutex.RUnlock()
	return len(fake.unspentTokensIteratorByArgsForCall)
}

func (fake *QueryEngine) UnspentTokensIteratorByCalls(stub func(context.Context, string, string) (driver.UnspentTokensIterator, error)) {
	fake.unspentTokensIteratorByMutex.Lock()
	defer fake.unspentTokensIteratorByMutex.Unlock()
	fake.UnspentTokensIteratorByStub = stub
}

func (fake *QueryEngine) UnspentTokensIteratorByArgsForCall(i int) (context.Context, string, string) {
	fake.unspentTokensIteratorByMutex.RLock()
	defer fake.unspentTokensIteratorByMutex.RUnlock()
	argsForCall := fake.unspentTokensIteratorByArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *QueryEngine) UnspentTokensIteratorByReturns(result1 driver.UnspentTokensIterator, result2 error) {
	fake.unspentTokensIteratorByMutex.Lock()
	defer fake.unspentTokensIteratorByMutex.Unlock()
	fake.UnspentTokensIteratorByStub = nil
	fake.unspentTokensIteratorByReturns = struct {
		result1 driver.UnspentTokensIterator
		result2 error
	}{result1, result2}
}

func (fake *QueryEngine) UnspentTokensIteratorByReturnsOnCall(i int, result1 driver.UnspentTokensIterator, result2 error) {
	fake.unspentTokensIteratorByMutex.Lock()
	defer fake.unspentTokensIteratorByMutex.Unlock()
	fake.UnspentTokensIteratorByStub = nil
	if fake.unspentTokensIteratorByReturnsOnCall == nil {
		fake.unspentTokensIteratorByReturnsOnCall = make(map[int]struct {
			result1 driver.UnspentTokensIterator
			result2 error
		})
	}
	fake.unspentTokensIteratorByReturnsOnCall[i] = struct {
		result1 driver.UnspentTokensIterator
		result2 error
	}{result1, result2}
}

func (fake *QueryEngine) WhoDeletedTokens(arg1 ...*token.ID) ([]string, []bool, error) {
	fake.whoDeletedTokensMutex.Lock()
	ret, specificReturn := fake.whoDeletedTokensReturnsOnCall[len(fake.whoDeletedTokensArgsForCall)]
	fake.whoDeletedTokensArgsForCall = append(fake.whoDeletedTokensArgsForCall, struct {
		arg1 []*token.ID
	}{arg1})
	stub := fake.WhoDeletedTokensStub
	fakeReturns := fake.whoDeletedTokensReturns
	fake.recordInvocation("WhoDeletedTokens", []interface{}{arg1})
	fake.whoDeletedTokensMutex.Unlock()
	if stub != nil {
		return stub(arg1...)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3
}

func (fake *QueryEngine) WhoDeletedTokensCallCount() int {
	fake.whoDeletedTokensMutex.RLock()
	defer fake.whoDeletedTokensMutex.RUnlock()
	return len(fake.whoDeletedTokensArgsForCall)
}

func (fake *QueryEngine) WhoDeletedTokensCalls(stub func(...*token.ID) ([]string, []bool, error)) {
	fake.whoDeletedTokensMutex.Lock()
	defer fake.whoDeletedTokensMutex.Unlock()
	fake.WhoDeletedTokensStub = stub
}

func (fake *QueryEngine) WhoDeletedTokensArgsForCall(i int) []*token.ID {
	fake.whoDeletedTokensMutex.RLock()
	defer fake.whoDeletedTokensMutex.RUnlock()
	argsForCall := fake.whoDeletedTokensArgsForCall[i]
	return argsForCall.arg1
}

func (fake *QueryEngine) WhoDeletedTokensReturns(result1 []string, result2 []bool, result3 error) {
	fake.whoDeletedTokensMutex.Lock()
	defer fake.whoDeletedTokensMutex.Unlock()
	fake.WhoDeletedTokensStub = nil
	fake.whoDeletedTokensReturns = struct {
		result1 []string
		result2 []bool
		result3 error
	}{result1, result2, result3}
}

func (fake *QueryEngine) WhoDeletedTokensReturnsOnCall(i int, result1 []string, result2 []bool, result3 error) {
	fake.whoDeletedTokensMutex.Lock()
	defer fake.whoDeletedTokensMutex.Unlock()
	fake.WhoDeletedTokensStub = nil
	if fake.whoDeletedTokensReturnsOnCall == nil {
		fake.whoDeletedTokensReturnsOnCall = make(map[int]struct {
			result1 []string
			result2 []bool
			result3 error
		})
	}
	fake.whoDeletedTokensReturnsOnCall[i] = struct {
		result1 []string
		result2 []bool
		result3 error
	}{result1, result2, result3}
}

func (fake *QueryEngine) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.getStatusMutex.RLock()
	defer fake.getStatusMutex.RUnlock()
	fake.getTokenInfoAndOutputsMutex.RLock()
	defer fake.getTokenInfoAndOutputsMutex.RUnlock()
	fake.getTokenInfosMutex.RLock()
	defer fake.getTokenInfosMutex.RUnlock()
	fake.getTokenOutputsMutex.RLock()
	defer fake.getTokenOutputsMutex.RUnlock()
	fake.getTokensMutex.RLock()
	defer fake.getTokensMutex.RUnlock()
	fake.isMineMutex.RLock()
	defer fake.isMineMutex.RUnlock()
	fake.isPendingMutex.RLock()
	defer fake.isPendingMutex.RUnlock()
	fake.listAuditTokensMutex.RLock()
	defer fake.listAuditTokensMutex.RUnlock()
	fake.listHistoryIssuedTokensMutex.RLock()
	defer fake.listHistoryIssuedTokensMutex.RUnlock()
	fake.listUnspentTokensMutex.RLock()
	defer fake.listUnspentTokensMutex.RUnlock()
	fake.publicParamsMutex.RLock()
	defer fake.publicParamsMutex.RUnlock()
	fake.unspentTokensIteratorMutex.RLock()
	defer fake.unspentTokensIteratorMutex.RUnlock()
	fake.unspentTokensIteratorByMutex.RLock()
	defer fake.unspentTokensIteratorByMutex.RUnlock()
	fake.whoDeletedTokensMutex.RLock()
	defer fake.whoDeletedTokensMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *QueryEngine) recordInvocation(key string, args []interface{}) {
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

var _ driver.QueryEngine = new(QueryEngine)
