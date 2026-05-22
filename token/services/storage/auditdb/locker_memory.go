/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditdb

import (
	"context"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"golang.org/x/sync/semaphore"
)

// memoryLocker is the default in-memory Locker. It uses weighted semaphores
// (weight 1) so that AcquireLocks respects context cancellation and deadlines.
// Suitable for single-replica deployments.
type memoryLocker struct {
	locks sync.Map
}

func newMemoryLocker() Locker {
	return &memoryLocker{}
}

func (m *memoryLocker) AcquireLocks(ctx context.Context, anchor string, eIDs ...string) error {
	dedup := deduplicateAndSort(eIDs)

	acquired := make([]string, 0, len(dedup))
	for _, id := range dedup {
		sem, _ := m.locks.LoadOrStore(id, semaphore.NewWeighted(1))
		if err := sem.(*semaphore.Weighted).Acquire(ctx, 1); err != nil {
			for _, aid := range acquired {
				if s, ok := m.locks.Load(aid); ok {
					s.(*semaphore.Weighted).Release(1)
				}
			}

			return errors.Wrapf(err, "failed to acquire lock for enrollment ID [%s]", id)
		}
		acquired = append(acquired, id)
	}

	m.locks.Store(anchor, dedup)

	return nil
}

func (m *memoryLocker) ReleaseLocks(_ context.Context, anchor string) {
	dedupBoxed, ok := m.locks.LoadAndDelete(anchor)
	if !ok {
		return
	}
	dedup := dedupBoxed.([]string)
	for _, id := range dedup {
		lock, ok := m.locks.Load(id)
		if !ok {
			continue
		}
		lock.(*semaphore.Weighted).Release(1)
	}
}

// AssertLocksHeld is a no-op for the memory locker. Semaphores cannot be
// "lost" the way a Postgres lease can expire, so the check is always satisfied.
func (m *memoryLocker) AssertLocksHeld(_ context.Context, _ string) error {
	return nil
}
