/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package session

import (
	"strconv"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
)

const (
	versionLabel = "version"
	typeLabel    = "type"
	errorLabel   = "error"
)

var (
	envelopeSentOpts = metrics.CounterOpts{
		Name:       "ttx_envelope_sent_total",
		Help:       "Total number of versioned envelopes sent",
		LabelNames: []string{versionLabel, typeLabel},
	}
	envelopeReceivedOpts = metrics.CounterOpts{
		Name:       "ttx_envelope_received_total",
		Help:       "Total number of versioned envelopes received",
		LabelNames: []string{versionLabel, typeLabel},
	}
	envelopeErrorsOpts = metrics.CounterOpts{
		Name:       "ttx_envelope_errors_total",
		Help:       "Total number of envelope validation errors",
		LabelNames: []string{errorLabel},
	}
	envelopeSizeOpts = metrics.HistogramOpts{
		Name:       "ttx_envelope_body_bytes",
		Help:       "Size of envelope body in bytes",
		LabelNames: []string{typeLabel},
	}
)

// EnvelopeMetrics holds the counters and histograms for envelope operations.
// Instantiate via NewEnvelopeMetrics and pass to SendTypedWithMetrics /
// ReceiveTypedWithMetrics. A nil *EnvelopeMetrics is safe and disables metrics.
type EnvelopeMetrics struct {
	Sent     metrics.Counter
	Received metrics.Counter
	Errors   metrics.Counter
	Size     metrics.Histogram
}

// NewEnvelopeMetrics registers and returns metrics using the given provider.
func NewEnvelopeMetrics(p metrics.Provider) *EnvelopeMetrics {
	return &EnvelopeMetrics{
		Sent:     p.NewCounter(envelopeSentOpts),
		Received: p.NewCounter(envelopeReceivedOpts),
		Errors:   p.NewCounter(envelopeErrorsOpts),
		Size:     p.NewHistogram(envelopeSizeOpts),
	}
}

// pkgMetrics holds the process-wide envelope metrics. It is registered once at
// service startup via RegisterMetrics and read by the typed send/receive helpers
// through envelopeMetrics(). Registering at startup (rather than per session)
// keeps the metrics provider out of the per-message hot path.
var (
	pkgMetricsOnce sync.Once
	pkgMetrics     *EnvelopeMetrics
)

// RegisterMetrics registers the envelope metrics against the given provider,
// once per process. It is safe to call repeatedly and from multiple goroutines;
// only the first non-nil call performs registration. Pass the metrics provider
// already held at service construction (e.g. obtained via FSC
// platform/view/services/metrics#GetProvider in the SDK wiring).
func RegisterMetrics(p metrics.Provider) {
	if p == nil {
		return
	}
	pkgMetricsOnce.Do(func() {
		pkgMetrics = NewEnvelopeMetrics(p)
	})
}

// envelopeMetrics returns the process-wide metrics, or nil when none were
// registered. A nil *EnvelopeMetrics is safe to use (all observe* methods no-op).
func envelopeMetrics() *EnvelopeMetrics {
	return pkgMetrics
}

func (m *EnvelopeMetrics) observeSend(msgType string, bodySize int) {
	if m == nil {
		return
	}
	m.Sent.With(versionLabel, fmtVersion(CurrentVersion), typeLabel, msgType).Add(1)
	m.Size.With(typeLabel, msgType).Observe(float64(bodySize))
}

func (m *EnvelopeMetrics) observeReceive(env *Envelope) {
	if m == nil {
		return
	}
	m.Received.With(versionLabel, fmtVersion(env.Version), typeLabel, env.Type).Add(1)
	m.Size.With(typeLabel, env.Type).Observe(float64(len(env.Body)))
}

func (m *EnvelopeMetrics) observeError(errType string) {
	if m == nil {
		return
	}
	m.Errors.With(errorLabel, errType).Add(1)
}

func fmtVersion(v uint32) string {
	return strconv.FormatUint(uint64(v), 10)
}
