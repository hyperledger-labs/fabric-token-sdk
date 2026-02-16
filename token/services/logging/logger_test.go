/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package logging_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/stretchr/testify/assert"
)

var logger = logging.MustGetLogger()

func logDebug() {
	logger.Debug("this is a debug message", "key", "value", "key2", "value2", "key3", "value3", "key4", "value4")
}

func logDebugf() {
	logger.Debugf("this is a debug message", "key", "value", "key2", "value2", "key3", "value3", "key4", "value4")
}

func TestLoggerDebugAllocs(t *testing.T) {
	const runs = 10000
	allocs := testing.AllocsPerRun(runs, func() {
		logDebug()
	})
	assert.GreaterOrEqual(t, allocs, float64(2))
	allocs = testing.AllocsPerRun(runs, func() {
		logDebugf()
	})
	assert.InDeltaf(t, float64(1), allocs, 0.01, "expected no delta from allocs")
}
