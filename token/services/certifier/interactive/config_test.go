/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interactive

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDefaultConstants guards that the operational defaults don't change accidentally.
// Changing them is a behaviour-breaking deployment change, not just a refactor.
func TestDefaultConstants(t *testing.T) {
	assert.Equal(t, 3, DefaultMaxAttempts, "DefaultMaxAttempts")
	assert.Equal(t, 10*time.Second, DefaultWaitTime, "DefaultWaitTime")
	assert.Equal(t, 10, DefaultBatchSize, "DefaultBatchSize")
	assert.Equal(t, 1000, DefaultBufferSize, "DefaultBufferSize")
	assert.Equal(t, 5*time.Second, DefaultFlushInterval, "DefaultFlushInterval")
	assert.Equal(t, 1, DefaultWorkers, "DefaultWorkers")
}
