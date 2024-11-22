/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"time"
)

const (
	Sherdlock Driver = "sherdlock"
	Simple    Driver = "simple"
)

type SelectorConfig interface {
	GetDriver() Driver
	GetNumRetries() int
	GetRetryInterval() time.Duration
	GetLeaseExpiry() time.Duration
	GetLeaseCleanupTickPeriod() time.Duration
}

type Driver string
