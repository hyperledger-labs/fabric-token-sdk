/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditor

import "time"

const (
	// Default lock acquisition retry configuration constants
	defaultMaxLockRetries        = 10                    // Maximum number of retry attempts
	defaultInitialLockBackoff    = 10 * time.Millisecond // Initial backoff delay
	defaultMaxLockBackoff        = 5 * time.Second       // Maximum backoff delay
	defaultLockBackoffMultiplier = 2.0                   // Exponential backoff multiplier
	defaultLockJitterFactor      = 0.3                   // Randomization factor (30%)
)

// LockConfig holds the configuration for lock acquisition retry logic
type LockConfig struct {
	// MaxRetries is the maximum number of retry attempts for lock acquisition
	MaxRetries int
	// InitialBackoff is the initial backoff delay before the first retry
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff delay between retries
	MaxBackoff time.Duration
	// BackoffMultiplier is the exponential backoff multiplier
	BackoffMultiplier float64
	// JitterFactor is the randomization factor to prevent thundering herd (0.0 to 1.0)
	JitterFactor float64
}

// DefaultLockConfig returns the default lock configuration
func DefaultLockConfig() *LockConfig {
	return &LockConfig{
		MaxRetries:        defaultMaxLockRetries,
		InitialBackoff:    defaultInitialLockBackoff,
		MaxBackoff:        defaultMaxLockBackoff,
		BackoffMultiplier: defaultLockBackoffMultiplier,
		JitterFactor:      defaultLockJitterFactor,
	}
}

// Made with Bob
