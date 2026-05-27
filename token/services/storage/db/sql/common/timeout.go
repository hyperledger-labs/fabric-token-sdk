/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"time"
)

// DBTimeoutConfig holds timeout configuration for different types of database operations
type DBTimeoutConfig struct {
	// ShortOpTimeout is for quick operations like locks, simple inserts/deletes
	ShortOpTimeout time.Duration
	// MediumOpTimeout is for queries and updates
	MediumOpTimeout time.Duration
	// LongOpTimeout is for batch operations and complex queries
	LongOpTimeout time.Duration
}

// DefaultDBTimeoutConfig returns the default timeout configuration
func DefaultDBTimeoutConfig() *DBTimeoutConfig {
	return &DBTimeoutConfig{
		ShortOpTimeout:  5 * time.Second,
		MediumOpTimeout: 15 * time.Second,
		LongOpTimeout:   30 * time.Second,
	}
}

// WithShortTimeout wraps the context with a timeout for short operations
// Returns the new context and a cancel function that MUST be called to release resources
func WithShortTimeout(ctx context.Context, config *DBTimeoutConfig) (context.Context, context.CancelFunc) {
	if config == nil {
		config = DefaultDBTimeoutConfig()
	}
	return context.WithTimeout(ctx, config.ShortOpTimeout)
}

// WithMediumTimeout wraps the context with a timeout for medium operations
// Returns the new context and a cancel function that MUST be called to release resources
func WithMediumTimeout(ctx context.Context, config *DBTimeoutConfig) (context.Context, context.CancelFunc) {
	if config == nil {
		config = DefaultDBTimeoutConfig()
	}
	return context.WithTimeout(ctx, config.MediumOpTimeout)
}

// WithLongTimeout wraps the context with a timeout for long operations
// Returns the new context and a cancel function that MUST be called to release resources
func WithLongTimeout(ctx context.Context, config *DBTimeoutConfig) (context.Context, context.CancelFunc) {
	if config == nil {
		config = DefaultDBTimeoutConfig()
	}
	return context.WithTimeout(ctx, config.LongOpTimeout)
}

// WithCustomTimeout wraps the context with a custom timeout duration
// Returns the new context and a cancel function that MUST be called to release resources
func WithCustomTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// Made with Bob
