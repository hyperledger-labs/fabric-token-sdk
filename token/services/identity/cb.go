/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultThreshold = 5
	defaultCooldown  = 30 * time.Second
)

// CircuitBreaker implements a lightweight in-memory circuit breaker.
type CircuitBreaker struct {
	sync.RWMutex
	failureCount int64
	lastFailure  time.Time
	open         bool
	threshold    int64
	cooldown     time.Duration
}

// CircuitBreakerConfig contains the configuration for the CircuitBreaker.
type CircuitBreakerConfig struct {
	// Threshold is the number of failures after which the circuit opens.
	Threshold int64
	// Cooldown is the duration after which the circuit closes automatically.
	Cooldown time.Duration
}

// NewCircuitBreaker returns a new instance of CircuitBreaker with the provided configuration.
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.Threshold <= 0 {
		config.Threshold = defaultThreshold
	}
	if config.Cooldown <= 0 {
		config.Cooldown = defaultCooldown
	}

	return &CircuitBreaker{
		threshold: config.Threshold,
		cooldown:  config.Cooldown,
	}
}

// Allow returns true if the circuit breaker allows the request to proceed.
func (circuitBreaker *CircuitBreaker) Allow() bool {
	circuitBreaker.RLock()
	if !circuitBreaker.open {
		circuitBreaker.RUnlock()

		return true
	}
	circuitBreaker.RUnlock()

	circuitBreaker.Lock()
	defer circuitBreaker.Unlock()
	if time.Since(circuitBreaker.lastFailure) > circuitBreaker.cooldown {
		circuitBreaker.open = false
		atomic.StoreInt64(&circuitBreaker.failureCount, 0)

		return true
	}

	return false
}

// RecordFailure increments the failure count and opens the circuit if the threshold is reached.
func (circuitBreaker *CircuitBreaker) RecordFailure() {
	count := atomic.AddInt64(&circuitBreaker.failureCount, 1)
	if count >= circuitBreaker.threshold {
		circuitBreaker.Lock()
		circuitBreaker.open = true
		circuitBreaker.lastFailure = time.Now()
		circuitBreaker.Unlock()
	}
}

// RecordSuccess resets the failure count and closes the circuit.
func (circuitBreaker *CircuitBreaker) RecordSuccess() {
	atomic.StoreInt64(&circuitBreaker.failureCount, 0)
	circuitBreaker.Lock()
	circuitBreaker.open = false
	circuitBreaker.Unlock()
}
