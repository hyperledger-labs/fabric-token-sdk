/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
)

type Metrics struct {
	*metrics.Metrics
}

func NewMetrics(provider metrics.Provider, tmsID token.TMSID) *Metrics {
	m := &Metrics{Metrics: metrics.New(provider, tmsID)}

	return m
}
