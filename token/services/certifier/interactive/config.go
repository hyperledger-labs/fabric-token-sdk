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
	// DefaultResponseTimeout is the maximum time the client waits for the certifier
	// to respond before treating the request as failed.
	DefaultResponseTimeout = 60 * time.Second

	// MaxTokensPerRequest is the maximum number of token IDs accepted in a single
	// certification request. Requests exceeding this limit are rejected to prevent
	// resource exhaustion on the certifier node.
	MaxTokensPerRequest = 500

	// MaxRequestBytes is the maximum byte-length of the cryptographic request payload
	// in a CertificationRequest. Requests exceeding this limit are rejected to prevent
	// memory exhaustion on the certifier node.
	MaxRequestBytes = 1 << 20 // 1 MiB

	// MaxWireMessageBytes is the maximum byte-length of the entire JSON-encoded
	// certification request as received from the wire. This guard fires before
	// JSON deserialisation so that an oversized message is dropped without ever
	// allocating the decoded struct — preventing memory exhaustion from large
	// payloads. It is set to 2 MiB to accommodate the base64 overhead of
	// MaxRequestBytes plus the JSON-encoded IDs and header fields.
	MaxWireMessageBytes = MaxRequestBytes * 2 // 2 MiB
)
