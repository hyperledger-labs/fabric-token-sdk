/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditdb

import "time"

// LockerBackend identifies which Locker implementation to use.
type LockerBackend string

const (
	LockerBackendMemory   LockerBackend = "memory"
	LockerBackendPostgres LockerBackend = "postgres"
)

// LockerConfig is the top-level configuration for the auditor EID locker.
// It is read from the TMS configuration under the key "auditor.locker".
type LockerConfig struct {
	Backend  LockerBackend        `yaml:"backend"`
	Postgres PostgresLockerConfig `yaml:"postgres"`
}

// DefaultLockerConfig returns the configuration that preserves
// the existing single-replica behaviour (in-memory locking).
func DefaultLockerConfig() LockerConfig {
	return LockerConfig{
		Backend: LockerBackendMemory,
		Postgres: PostgresLockerConfig{
			TTL:             30 * time.Second,
			AcquireBackoff:  100 * time.Millisecond,
			AcquireDeadline: time.Minute,
			Heartbeat:       10 * time.Second,
		},
	}
}
