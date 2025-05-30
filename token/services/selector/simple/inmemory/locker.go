/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package inmemory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/simple"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var (
	logger             = logging.MustGetLogger()
	AlreadyLockedError = errors.New("already locked")
)

type TXStatusProvider interface {
	GetStatus(ctx context.Context, txID string) (ttxdb.TxStatus, string, error)
}

type lockEntry struct {
	TxID       string
	Created    time.Time
	LastAccess time.Time
}

func (l *lockEntry) String() string {
	return fmt.Sprintf("[[%s] since [%s], last access [%s]]", l.TxID, l.Created, l.LastAccess)
}

type locker struct {
	ttxdb                  TXStatusProvider
	lock                   *sync.RWMutex
	locked                 map[token2.ID]*lockEntry
	sleepTimeout           time.Duration
	validTxEvictionTimeout time.Duration
}

func NewLocker(ttxdb TXStatusProvider, timeout time.Duration, validTxEvictionTimeout time.Duration) simple.Locker {
	r := &locker{
		ttxdb:                  ttxdb,
		sleepTimeout:           timeout,
		lock:                   &sync.RWMutex{},
		locked:                 map[token2.ID]*lockEntry{},
		validTxEvictionTimeout: validTxEvictionTimeout,
	}
	r.Start()
	return r
}

func (d *locker) Lock(ctx context.Context, id *token2.ID, txID string, reclaim bool) (string, error) {
	k := *id

	// check quickly if the token is locked
	d.lock.RLock()
	if _, ok := d.locked[k]; ok && !reclaim {
		// return immediately
		d.lock.RUnlock()
		return "", AlreadyLockedError
	}
	d.lock.RUnlock()

	// it is either not locked or we are reclaiming
	d.lock.Lock()
	defer d.lock.Unlock()
	e, ok := d.locked[k]
	if ok {
		e.LastAccess = time.Now()

		if reclaim {
			// Second chance
			logger.Debugf("[%s] already locked by [%s], try to reclaim...", id, e)
			reclaimed, status := d.reclaim(context.Background(), id, e.TxID)
			if !reclaimed {
				logger.Debugf("[%s] already locked by [%s], reclaim failed, tx status [%s]", id, e, ttxdb.TxStatusMessage[status])
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					return e.TxID, errors.Errorf("already locked by [%s]", e)
				}
				return e.TxID, AlreadyLockedError
			}
			logger.Debugf("[%s] already locked by [%s], reclaimed successful, tx status [%s]", id, e, ttxdb.TxStatusMessage[status])
		} else {
			logger.Debugf("[%s] already locked by [%s], no reclaim", id, e)
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				return e.TxID, errors.Errorf("already locked by [%s]", e)
			}
			return e.TxID, AlreadyLockedError
		}
	}
	logger.Debugf("locking [%s] for [%s]", id, txID)
	now := time.Now()
	d.locked[k] = &lockEntry{TxID: txID, Created: now, LastAccess: now}
	return "", nil
}

// UnlockIDs unlocks the passed IDS. It returns the list of tokens that were not locked in the first place among
// those passed.
func (d *locker) UnlockIDs(ids ...*token2.ID) []*token2.ID {
	d.lock.Lock()
	defer d.lock.Unlock()

	logger.Debugf("unlocking tokens [%v]", ids)
	var notFound []*token2.ID
	for _, id := range ids {
		k := *id
		entry, ok := d.locked[k]
		if !ok {
			notFound = append(notFound, &k)
			logger.Warnf("unlocking [%s] hold by no one, skipping [%s]", id, entry)
			continue
		}
		logger.Debugf("unlocking [%s] hold by [%s]", id, entry)
		delete(d.locked, k)
	}
	return notFound
}

func (d *locker) UnlockByTxID(ctx context.Context, txID string) {
	d.lock.Lock()
	defer d.lock.Unlock()

	logger.Debugf("unlocking tokens hold by [%s]", txID)
	for id, entry := range d.locked {
		if entry.TxID == txID {
			logger.Debugf("unlocking [%s] hold by [%s]", id, entry)
			delete(d.locked, id)
		}
	}
}

func (d *locker) IsLocked(id *token2.ID) bool {
	d.lock.Lock()
	defer d.lock.Unlock()

	_, ok := d.locked[*id]
	return ok
}

func (d *locker) reclaim(ctx context.Context, id *token2.ID, txID string) (bool, int) {
	status, _, err := d.ttxdb.GetStatus(ctx, txID)
	if err != nil {
		return false, status
	}
	switch status {
	case ttxdb.Deleted:
		delete(d.locked, *id)
		return true, status
	default:
		return false, status
	}
}

func (d *locker) Start() {
	go d.scan(context.Background())
}

func (d *locker) scan(ctx context.Context) {
	for {
		logger.Debugf("token collector: scan locked tokens")
		var removeList []token2.ID
		d.lock.RLock()
		for id, entry := range d.locked {
			status, _, err := d.ttxdb.GetStatus(ctx, entry.TxID)
			if err != nil {
				logger.Warnf("failed getting status for token [%s] locked by [%s], remove", id, entry)
				removeList = append(removeList, id)
				continue
			}
			switch status {
			case ttxdb.Confirmed:
				// remove only if elapsed enough time from last access, to avoid concurrency issue
				if time.Since(entry.LastAccess) > d.validTxEvictionTimeout {
					removeList = append(removeList, id)
					logger.Debugf("token [%s] locked by [%s] in status [%s], time elapsed, remove", id, entry, ttxdb.TxStatusMessage[status])
				}
			case ttxdb.Deleted:
				removeList = append(removeList, id)
				logger.Debugf("token [%s] locked by [%s] in status [%s], remove", id, entry, ttxdb.TxStatusMessage[status])
			default:
				logger.Debugf("token [%s] locked by [%s] in status [%s], skip", id, entry, ttxdb.TxStatusMessage[status])
			}
		}
		d.lock.RUnlock()

		d.lock.Lock()
		logger.Debugf("token collector: freeing [%d] items", len(removeList))
		for _, s := range removeList {
			delete(d.locked, s)
		}
		d.lock.Unlock()

		for {
			logger.Debugf("token collector: sleep for some time...")
			time.Sleep(d.sleepTimeout)
			d.lock.RLock()
			l := len(d.locked)
			d.lock.RUnlock()
			if l > 0 {
				// time to do some token collection
				logger.Debugf("token collector: time to do some token collection, [%d] locked", l)
				break
			}
		}
	}
}
