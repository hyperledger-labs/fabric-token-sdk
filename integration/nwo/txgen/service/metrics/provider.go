/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package metrics

import (
	"math"
	"sync"

	"github.com/hyperledger/fabric-lib-go/common/metrics"
)

func NewProvider() metrics.Provider { return &Provider{} }

type Provider struct{}

func (p *Provider) NewCounter(metrics.CounterOpts) metrics.Counter {
	return &counter{singleAccess: &singleAccess{}}
}

func (p *Provider) NewGauge(metrics.GaugeOpts) metrics.Gauge {
	return &gauge{counter: &counter{singleAccess: &singleAccess{}}}
}

func (p *Provider) NewHistogram(metrics.HistogramOpts) metrics.Histogram {
	return &histogram{singleAccess: &singleAccess{}, min: math.MaxFloat64, max: -math.MaxFloat64}
}

type counter struct {
	*singleAccess
	v float64
}

func (c *counter) With(...string) metrics.Counter { return c }
func (c *counter) Add(v float64)                  { c.do(func() { c.v += v }) }
func (c *counter) Get() float64                   { return c.v }

type gauge struct{ *counter }

func (g *gauge) With(...string) metrics.Gauge { return g }
func (g *gauge) Set(v float64)                { g.do(func() { g.v = v }) }

type histogram struct {
	*singleAccess
	min, max, sum float64
	n             uint64
}

func (h *histogram) With(...string) metrics.Histogram { return h }
func (h *histogram) Observe(v float64) {
	h.do(func() {
		h.min = min(v, h.min)
		h.max = max(v, h.max)
		h.sum += v
		h.n++
	})
}
func (h *histogram) Min() float64 { return h.min }
func (h *histogram) Max() float64 { return h.max }
func (h *histogram) Avg() float64 { return h.sum / float64(h.n) }

type singleAccess struct{ m sync.Mutex }

func (c *singleAccess) do(f func()) {
	c.m.Lock()
	defer c.m.Unlock()
	f()
}
