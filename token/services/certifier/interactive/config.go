/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interactive

import "time"

// Default operational parameters for the CertificationClient.
const (
	// DefaultMaxAttempts is the default number of times a certification request
	// is retried before giving up and pushing the tokens back to the queue.
	DefaultMaxAttempts = 3
	// DefaultWaitTime is the default backoff duration between retry attempts.
	DefaultWaitTime = 10 * time.Second
	// DefaultBatchSize is the default maximum number of tokens per certification batch.
	DefaultBatchSize = 10
	// DefaultBufferSize is the default capacity of the incoming token channel.
	DefaultBufferSize = 1000
	// DefaultFlushInterval is the default period after which a partial batch is
	// flushed to the worker pool even if it has not reached BatchSize.
	DefaultFlushInterval = 5 * time.Second
	// DefaultWorkers is the default number of worker goroutines processing certification batches.
	DefaultWorkers = 1
)
