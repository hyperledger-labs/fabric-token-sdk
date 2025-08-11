/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package metrics

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
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
			Name:       "sent",
			Help:       "Total transfer requests executed",
			LabelNames: []string{OperationLabel},
		}),
		RequestsReceived: p.NewCounter(metrics.CounterOpts{
			Name:       "received",
			Help:       "Success transfer requests executed",
			LabelNames: []string{OperationLabel, SuccessLabel},
		}),
		RequestDuration: p.NewHistogram(metrics.HistogramOpts{
			Name:       "duration",
			Help:       "Duration of transfer requests executed",
			Buckets:    utils.ExponentialBucketTimeRange(0, 50*time.Second, 15),
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
