/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package metrics

import (
	"time"

	c "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/txgen/model/constants"
)

type Reporter interface {
	GetTotalRequests() string
	GetActiveRequests() string
	Summary() string
}

type Collector interface {
	IncrementRequests()
	DecrementRequests()
	AddDuration(millisDuration time.Duration, requestType c.ApiRequestType, success bool)
}
