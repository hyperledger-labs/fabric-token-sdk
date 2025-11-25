/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
)

var (
	zkIssueDurationOpts = metrics.HistogramOpts{
		Name:       "issue_duration",
		Help:       "Duration of zk issue token",
		LabelNames: []string{"network", "channel", "namespace"},
		Buckets:    utils.ExponentialBucketTimeRange(0, 1*time.Second, 10),
	}
	zkTransferDurationOpts = metrics.HistogramOpts{
		Name:       "transfer_duration",
		Help:       "Duration of zk transfer token",
		LabelNames: []string{"network", "channel", "namespace"},
		Buckets:    utils.ExponentialBucketTimeRange(0, 1*time.Second, 10),
	}
)

type Metrics struct {
	zkIssueDuration    metrics.Histogram
	zkTransferDuration metrics.Histogram
}

func NewMetrics(p metrics.Provider) *Metrics {
	return &Metrics{
		zkIssueDuration:    p.NewHistogram(zkIssueDurationOpts),
		zkTransferDuration: p.NewHistogram(zkTransferDurationOpts),
	}
}
