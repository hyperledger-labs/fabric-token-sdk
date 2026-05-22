/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditdb

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

// Locker coordinates exclusive access to enrollment IDs during auditor processing.
// Implementations must guarantee that AcquireLocks with intersecting EID sets
// on different replicas are serialized (at most one holder at a time).
type Locker interface {
	// AcquireLocks acquires locks for the given anchor and enrollment IDs.
	// EIDs are deduplicated and sorted internally to prevent deadlocks.
	AcquireLocks(ctx context.Context, anchor string, eIDs ...string) error
	// ReleaseLocks releases the locks associated with the given anchor.
	ReleaseLocks(ctx context.Context, anchor string)
	// AssertLocksHeld verifies that this replica still holds all locks for the anchor.
	// Returns ErrLockNotHeld if one or more leases expired or were reclaimed.
	AssertLocksHeld(ctx context.Context, anchor string) error
}

var (
	ErrLockContention     = errors.New("auditor enrollment id lock contention")
	ErrLockAcquireTimeout = errors.New("auditor enrollment id lock acquire timeout")
	ErrLockLost           = errors.New("auditor enrollment id lock lost")
	ErrLockNotHeld        = errors.New("auditor enrollment id locks not held")
)
