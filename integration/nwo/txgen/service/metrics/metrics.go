/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package metrics

import (
	"time"

	"github.com/hyperledger/fabric-lib-go/common/metrics"
)

type OperationType = string

const (
	Transfer OperationType = "transfer"
	Balance  OperationType = "balance"
	Withdraw OperationType = "withdraw"
)

type SuccessType = string

var SuccessValues = map[bool]SuccessType{
	true:  "success",
	false: "failure",
}

const (
	OperationLabel = "operation"
	SuccessLabel   = "success"
)

func NewMetrics(p metrics.Provider) *Metrics {
	_, supportsGetters := p.(*Provider)
	return &Metrics{
		supportsGetters: supportsGetters,

		RequestsSent: p.NewCounter(metrics.CounterOpts{
			Namespace:  "tx_gen",
			Name:       "sent",
			Help:       "Total transfer requests executed",
			LabelNames: []string{OperationLabel},
		}),
		RequestsReceived: p.NewCounter(metrics.CounterOpts{
			Namespace:  "tx_gen",
			Name:       "received",
			Help:       "Success transfer requests executed",
			LabelNames: []string{OperationLabel, SuccessLabel},
		}),
		RequestDuration: p.NewHistogram(metrics.HistogramOpts{
			Namespace:  "tx_gen",
			Name:       "duration",
			Help:       "Duration of transfer requests executed",
			Buckets:    bucketRange(0, 15*time.Second, 100),
			LabelNames: []string{OperationLabel, SuccessLabel},
		}),
	}
}

type Metrics struct {
	supportsGetters bool

	RequestsSent     metrics.Counter
	RequestsReceived metrics.Counter
	RequestDuration  metrics.Histogram
}

func bucketRange(start, end time.Duration, buckets int) []float64 {
	bs := make([]float64, 0, buckets+1)
	step := (end.Seconds() - start.Seconds()) / float64(buckets)
	for v := start.Seconds(); v <= end.Seconds(); v += step {
		bs = append(bs, v)
	}
	return bs
}
