/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package metrics

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
)

type (
	CounterOpts   = metrics.CounterOpts
	Counter       = metrics.Counter
	GaugeOpts     = metrics.GaugeOpts
	Gauge         = metrics.Gauge
	HistogramOpts = metrics.HistogramOpts
	Histogram     = metrics.Histogram
	Provider      = metrics.Provider
	MetricLabel   = string
)

var GetProvider = metrics.GetProvider
