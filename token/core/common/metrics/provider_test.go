package metrics_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics/mock"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestTMSProvider_NewCounter(t *testing.T) {
	tmsID := token.TMSID{
		Network:   "network",
		Channel:   "channel",
		Namespace: "namespace",
	}
	tmsLabels := []string{
		metrics.NetworkLabel, tmsID.Network,
		metrics.ChannelLabel, tmsID.Channel,
		metrics.NamespaceLabel, tmsID.Namespace,
	}
	tests := []struct {
		name      string
		setup     func(*mock.Provider)
		wantPanic bool
	}{
		{
			name: "successful counter creation",
			setup: func(mp *mock.Provider) {
				mp.NewCounterStub = func(opts metrics.CounterOpts) metrics.Counter {
					return &mockCounter{}
				}
			},
			wantPanic: false,
		},
		{
			name: "counter creation with error",
			setup: func(mp *mock.Provider) {
				mp.NewCounterStub = func(opts metrics.CounterOpts) metrics.Counter {
					panic(prometheus.AlreadyRegisteredError{})
				}
			},
			wantPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mp := &mock.Provider{}
			if tt.setup != nil {
				tt.setup(mp)
			}
			p := metrics.NewTMSProvider(tmsID, mp)
			if tt.wantPanic {
				assert.Panics(t, func() {
					counter := p.NewCounter(metrics.CounterOpts{})
					assert.Equal(t, tmsLabels, counter.(*mockCounter).labelValues)
				})
			} else {
				assert.NotPanics(t, func() { p.NewCounter(metrics.CounterOpts{}) })
			}
		})
	}
}

func TestTMSProvider_NewGauge(t *testing.T) {
	tmsID := token.TMSID{
		Network:   "network",
		Channel:   "channel",
		Namespace: "namespace",
	}
	tmsLabels := []string{
		metrics.NetworkLabel, tmsID.Network,
		metrics.ChannelLabel, tmsID.Channel,
		metrics.NamespaceLabel, tmsID.Namespace,
	}
	tests := []struct {
		name      string
		setup     func(*mock.Provider)
		wantPanic bool
	}{
		{
			name: "successful gauge creation",
			setup: func(mp *mock.Provider) {
				mp.NewGaugeStub = func(opts metrics.GaugeOpts) metrics.Gauge {
					return &mockGauge{}
				}
			},
			wantPanic: false,
		},
		{
			name: "gauge creation with error",
			setup: func(mp *mock.Provider) {
				mp.NewGaugeStub = func(opts metrics.GaugeOpts) metrics.Gauge {
					panic(prometheus.AlreadyRegisteredError{})
				}
			},
			wantPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mp := &mock.Provider{}
			if tt.setup != nil {
				tt.setup(mp)
			}
			p := metrics.NewTMSProvider(tmsID, mp)
			if tt.wantPanic {
				assert.Panics(t, func() {
					gauge := p.NewGauge(metrics.GaugeOpts{})
					assert.Equal(t, tmsLabels, gauge.(*mockGauge).labelValues)
				})
			} else {
				assert.NotPanics(t, func() { p.NewGauge(metrics.GaugeOpts{}) })
			}
		})
	}
}

func TestTMSProvider_NewHistogram(t *testing.T) {
	tmsID := token.TMSID{
		Network:   "network",
		Channel:   "channel",
		Namespace: "namespace",
	}
	tmsLabels := []string{
		metrics.NetworkLabel, tmsID.Network,
		metrics.ChannelLabel, tmsID.Channel,
		metrics.NamespaceLabel, tmsID.Namespace,
	}
	tests := []struct {
		name      string
		setup     func(*mock.Provider)
		wantPanic bool
	}{
		{
			name: "successful histogram creation",
			setup: func(mp *mock.Provider) {
				mp.NewHistogramStub = func(opts metrics.HistogramOpts) metrics.Histogram {
					return &mockHistogram{}
				}
			},
			wantPanic: false,
		},
		{
			name: "histogram creation with error",
			setup: func(mp *mock.Provider) {
				mp.NewHistogramStub = func(opts metrics.HistogramOpts) metrics.Histogram {
					panic(prometheus.AlreadyRegisteredError{})
				}
			},
			wantPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mp := &mock.Provider{}
			if tt.setup != nil {
				tt.setup(mp)
			}
			p := metrics.NewTMSProvider(tmsID, mp)
			if tt.wantPanic {
				assert.Panics(t, func() {
					histogram := p.NewHistogram(metrics.HistogramOpts{})
					assert.Equal(t, tmsLabels, histogram.(*mockHistogram).labelValues)
				})
			} else {
				assert.NotPanics(t, func() { p.NewHistogram(metrics.HistogramOpts{}) })
			}
		})
	}
}

func TestAllLabelNames(t *testing.T) {
	tests := []struct {
		name   string
		extras []string
		want   []string
	}{
		{
			name:   "no extras",
			extras: nil,
			want:   []string{"network", "channel", "namespace"},
		},
		{
			name:   "with extras",
			extras: []string{"extra1", "extra2"},
			want:   []string{"network", "channel", "namespace", "extra1", "extra2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := metrics.AllLabelNames(tt.extras...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStatsdFormat(t *testing.T) {
	tests := []struct {
		name   string
		labels []string
		values []string
		want   string
	}{
		{
			name:   "empty labels",
			labels: nil,
			want:   "%{#fqname}.%{network}.%{channel}.%{namespace}",
		},
		{
			name:   "single label",
			labels: []string{"label1"},
			want:   "%{#fqname}.%{network}.%{channel}.%{namespace}.%{label1}",
		},
		{
			name:   "multiple labels",
			labels: []string{"l1", "l2"},
			want:   "%{#fqname}.%{network}.%{channel}.%{namespace}.%{l1}.%{l2}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := metrics.StatsdFormat(tt.labels...)
			assert.Equal(t, tt.want, got)
		})
	}
}

type mockCounter struct {
	labelValues []string
}

func (m *mockCounter) With(labelValues ...string) metrics.Counter {
	m.labelValues = labelValues
	return m
}

func (m *mockCounter) Add(delta float64) {
	return
}

type mockGauge struct {
	labelValues []string
}

func (m *mockGauge) Set(value float64) {
	return
}

func (m *mockGauge) With(labelValues ...string) metrics.Gauge {
	m.labelValues = labelValues
	return m
}

func (m *mockGauge) Add(delta float64) {
	return
}

type mockHistogram struct {
	labelValues []string
}

func (m *mockHistogram) With(labelValues ...string) metrics.Histogram {
	m.labelValues = labelValues
	return m
}

func (m *mockHistogram) Observe(value float64) {
	return
}
