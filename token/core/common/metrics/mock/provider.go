// Code generated by counterfeiter. DO NOT EDIT.
package mock

import (
	"sync"

	metricsa "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
)

type Provider struct {
	NewCounterStub        func(metricsa.CounterOpts) metricsa.Counter
	newCounterMutex       sync.RWMutex
	newCounterArgsForCall []struct {
		arg1 metricsa.CounterOpts
	}
	newCounterReturns struct {
		result1 metricsa.Counter
	}
	newCounterReturnsOnCall map[int]struct {
		result1 metricsa.Counter
	}
	NewGaugeStub        func(metricsa.GaugeOpts) metricsa.Gauge
	newGaugeMutex       sync.RWMutex
	newGaugeArgsForCall []struct {
		arg1 metricsa.GaugeOpts
	}
	newGaugeReturns struct {
		result1 metricsa.Gauge
	}
	newGaugeReturnsOnCall map[int]struct {
		result1 metricsa.Gauge
	}
	NewHistogramStub        func(metricsa.HistogramOpts) metricsa.Histogram
	newHistogramMutex       sync.RWMutex
	newHistogramArgsForCall []struct {
		arg1 metricsa.HistogramOpts
	}
	newHistogramReturns struct {
		result1 metricsa.Histogram
	}
	newHistogramReturnsOnCall map[int]struct {
		result1 metricsa.Histogram
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *Provider) NewCounter(arg1 metricsa.CounterOpts) metricsa.Counter {
	fake.newCounterMutex.Lock()
	ret, specificReturn := fake.newCounterReturnsOnCall[len(fake.newCounterArgsForCall)]
	fake.newCounterArgsForCall = append(fake.newCounterArgsForCall, struct {
		arg1 metricsa.CounterOpts
	}{arg1})
	stub := fake.NewCounterStub
	fakeReturns := fake.newCounterReturns
	fake.recordInvocation("NewCounter", []interface{}{arg1})
	fake.newCounterMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *Provider) NewCounterCallCount() int {
	fake.newCounterMutex.RLock()
	defer fake.newCounterMutex.RUnlock()
	return len(fake.newCounterArgsForCall)
}

func (fake *Provider) NewCounterCalls(stub func(metricsa.CounterOpts) metricsa.Counter) {
	fake.newCounterMutex.Lock()
	defer fake.newCounterMutex.Unlock()
	fake.NewCounterStub = stub
}

func (fake *Provider) NewCounterArgsForCall(i int) metricsa.CounterOpts {
	fake.newCounterMutex.RLock()
	defer fake.newCounterMutex.RUnlock()
	argsForCall := fake.newCounterArgsForCall[i]
	return argsForCall.arg1
}

func (fake *Provider) NewCounterReturns(result1 metricsa.Counter) {
	fake.newCounterMutex.Lock()
	defer fake.newCounterMutex.Unlock()
	fake.NewCounterStub = nil
	fake.newCounterReturns = struct {
		result1 metricsa.Counter
	}{result1}
}

func (fake *Provider) NewCounterReturnsOnCall(i int, result1 metricsa.Counter) {
	fake.newCounterMutex.Lock()
	defer fake.newCounterMutex.Unlock()
	fake.NewCounterStub = nil
	if fake.newCounterReturnsOnCall == nil {
		fake.newCounterReturnsOnCall = make(map[int]struct {
			result1 metricsa.Counter
		})
	}
	fake.newCounterReturnsOnCall[i] = struct {
		result1 metricsa.Counter
	}{result1}
}

func (fake *Provider) NewGauge(arg1 metricsa.GaugeOpts) metricsa.Gauge {
	fake.newGaugeMutex.Lock()
	ret, specificReturn := fake.newGaugeReturnsOnCall[len(fake.newGaugeArgsForCall)]
	fake.newGaugeArgsForCall = append(fake.newGaugeArgsForCall, struct {
		arg1 metricsa.GaugeOpts
	}{arg1})
	stub := fake.NewGaugeStub
	fakeReturns := fake.newGaugeReturns
	fake.recordInvocation("NewGauge", []interface{}{arg1})
	fake.newGaugeMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *Provider) NewGaugeCallCount() int {
	fake.newGaugeMutex.RLock()
	defer fake.newGaugeMutex.RUnlock()
	return len(fake.newGaugeArgsForCall)
}

func (fake *Provider) NewGaugeCalls(stub func(metricsa.GaugeOpts) metricsa.Gauge) {
	fake.newGaugeMutex.Lock()
	defer fake.newGaugeMutex.Unlock()
	fake.NewGaugeStub = stub
}

func (fake *Provider) NewGaugeArgsForCall(i int) metricsa.GaugeOpts {
	fake.newGaugeMutex.RLock()
	defer fake.newGaugeMutex.RUnlock()
	argsForCall := fake.newGaugeArgsForCall[i]
	return argsForCall.arg1
}

func (fake *Provider) NewGaugeReturns(result1 metricsa.Gauge) {
	fake.newGaugeMutex.Lock()
	defer fake.newGaugeMutex.Unlock()
	fake.NewGaugeStub = nil
	fake.newGaugeReturns = struct {
		result1 metricsa.Gauge
	}{result1}
}

func (fake *Provider) NewGaugeReturnsOnCall(i int, result1 metricsa.Gauge) {
	fake.newGaugeMutex.Lock()
	defer fake.newGaugeMutex.Unlock()
	fake.NewGaugeStub = nil
	if fake.newGaugeReturnsOnCall == nil {
		fake.newGaugeReturnsOnCall = make(map[int]struct {
			result1 metricsa.Gauge
		})
	}
	fake.newGaugeReturnsOnCall[i] = struct {
		result1 metricsa.Gauge
	}{result1}
}

func (fake *Provider) NewHistogram(arg1 metricsa.HistogramOpts) metricsa.Histogram {
	fake.newHistogramMutex.Lock()
	ret, specificReturn := fake.newHistogramReturnsOnCall[len(fake.newHistogramArgsForCall)]
	fake.newHistogramArgsForCall = append(fake.newHistogramArgsForCall, struct {
		arg1 metricsa.HistogramOpts
	}{arg1})
	stub := fake.NewHistogramStub
	fakeReturns := fake.newHistogramReturns
	fake.recordInvocation("NewHistogram", []interface{}{arg1})
	fake.newHistogramMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *Provider) NewHistogramCallCount() int {
	fake.newHistogramMutex.RLock()
	defer fake.newHistogramMutex.RUnlock()
	return len(fake.newHistogramArgsForCall)
}

func (fake *Provider) NewHistogramCalls(stub func(metricsa.HistogramOpts) metricsa.Histogram) {
	fake.newHistogramMutex.Lock()
	defer fake.newHistogramMutex.Unlock()
	fake.NewHistogramStub = stub
}

func (fake *Provider) NewHistogramArgsForCall(i int) metricsa.HistogramOpts {
	fake.newHistogramMutex.RLock()
	defer fake.newHistogramMutex.RUnlock()
	argsForCall := fake.newHistogramArgsForCall[i]
	return argsForCall.arg1
}

func (fake *Provider) NewHistogramReturns(result1 metricsa.Histogram) {
	fake.newHistogramMutex.Lock()
	defer fake.newHistogramMutex.Unlock()
	fake.NewHistogramStub = nil
	fake.newHistogramReturns = struct {
		result1 metricsa.Histogram
	}{result1}
}

func (fake *Provider) NewHistogramReturnsOnCall(i int, result1 metricsa.Histogram) {
	fake.newHistogramMutex.Lock()
	defer fake.newHistogramMutex.Unlock()
	fake.NewHistogramStub = nil
	if fake.newHistogramReturnsOnCall == nil {
		fake.newHistogramReturnsOnCall = make(map[int]struct {
			result1 metricsa.Histogram
		})
	}
	fake.newHistogramReturnsOnCall[i] = struct {
		result1 metricsa.Histogram
	}{result1}
}

func (fake *Provider) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.newCounterMutex.RLock()
	defer fake.newCounterMutex.RUnlock()
	fake.newGaugeMutex.RLock()
	defer fake.newGaugeMutex.RUnlock()
	fake.newHistogramMutex.RLock()
	defer fake.newHistogramMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *Provider) recordInvocation(key string, args []interface{}) {
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

var _ metrics.Provider = new(Provider)
