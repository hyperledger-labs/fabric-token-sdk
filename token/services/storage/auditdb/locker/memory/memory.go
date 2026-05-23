/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"context"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/auditdb/locker/dedup"
	"golang.org/x/sync/semaphore"
)

// Locker is the default in-memory Locker. It uses weighted semaphores
// (weight 1) so that AcquireLocks respects context cancellation and deadlines.
// Suitable for single-replica deployments.
type Locker struct {
	locks sync.Map
}

func New() *Locker {
	return &Locker{}
}

func (m *Locker) AcquireLocks(ctx context.Context, anchor string, eIDs ...string) error {
	deduped := dedup.AndSort(eIDs)

	acquired := make([]string, 0, len(deduped))
	for _, id := range deduped {
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

	m.locks.Store(anchor, deduped)

	return nil
}

func (m *Locker) ReleaseLocks(_ context.Context, anchor string) {
	dedupBoxed, ok := m.locks.LoadAndDelete(anchor)
	if !ok {
		return
	}
	deduped := dedupBoxed.([]string)
	for _, id := range deduped {
		lock, ok := m.locks.Load(id)
		if !ok {
			continue
		}
		lock.(*semaphore.Weighted).Release(1)
	}
}

func (m *Locker) AssertLocksHeld(_ context.Context, _ string) error {
	return nil
}
