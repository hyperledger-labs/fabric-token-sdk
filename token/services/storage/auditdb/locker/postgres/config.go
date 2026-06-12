/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import "time"

const (
	defaultTTL             = 30 * time.Second
	defaultAcquireBackoff  = 100 * time.Millisecond
	defaultAcquireDeadline = time.Minute
	defaultHeartbeat       = 10 * time.Second
)

// Config holds Postgres lease-table locking settings.
type Config struct {
	// TTL is the lease duration for each EID lock row.
	TTL time.Duration `yaml:"ttl"`
	// AcquireBackoff is the wait between retry attempts when a lock is contended.
	AcquireBackoff time.Duration `yaml:"acquireBackoff"`
	// AcquireDeadline is the total time allowed to acquire all EID locks.
	AcquireDeadline time.Duration `yaml:"acquireDeadline"`
	// Heartbeat is the interval at which held leases are renewed (~TTL/3).
	Heartbeat time.Duration `yaml:"heartbeat"`
	// Owner identifies this replica. Defaults to the FSC node ID when empty.
	Owner string `yaml:"owner"`
}

func (c Config) withDefaults(owner string) Config {
	if c.TTL <= 0 {
		c.TTL = defaultTTL
	}
	if c.AcquireBackoff <= 0 {
		c.AcquireBackoff = defaultAcquireBackoff
	}
	if c.AcquireDeadline <= 0 {
		c.AcquireDeadline = defaultAcquireDeadline
	}
	if c.Heartbeat <= 0 {
		c.Heartbeat = defaultHeartbeat
	}
	if c.Owner == "" {
		c.Owner = owner
	}

	return c
}
