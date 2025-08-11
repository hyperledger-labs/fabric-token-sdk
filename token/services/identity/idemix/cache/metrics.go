/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
)

var (
	LevelOpts = metrics.GaugeOpts{
		Namespace:  "idemix",
		Name:       "cache_level",
		Help:       "Level of the idemix cache",
		LabelNames: []string{"network", "channel", "namespace"},
	}
)

// Metrics contains the metrics for this package
type Metrics struct {
	CacheLevelGauge metrics.Gauge
}

// NewMetrics instantiate the metrics for this package
func NewMetrics(p metrics.Provider) *Metrics {
	return &Metrics{
		CacheLevelGauge: p.NewGauge(LevelOpts),
	}
}
