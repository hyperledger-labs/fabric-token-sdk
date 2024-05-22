/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package inmemory

import (
	"fmt"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/simple"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var (
	logger             = logging.MustGetLogger("token-sdk.selector.inmemory")
	AlreadyLockedError = errors.New("already locked")
)

const (
	_       int = iota
	Valid       // Transaction is valid and committed
	Invalid     // Transaction is invalid and has been discarded
)

type Vault interface {
	Status(id string) (int, error)
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
	vault                  Vault
	lock                   *sync.RWMutex
	locked                 map[token2.ID]*lockEntry
	sleepTimeout           time.Duration
	validTxEvictionTimeout time.Duration
}

func NewLocker(vault Vault, timeout time.Duration, validTxEvictionTimeout time.Duration) simple.Locker {
	r := &locker{
		vault:                  vault,
		sleepTimeout:           timeout,
		lock:                   &sync.RWMutex{},
		locked:                 map[token2.ID]*lockEntry{},
		validTxEvictionTimeout: validTxEvictionTimeout,
	}
	r.Start()
	return r
}

func (d *locker) Lock(id *token2.ID, txID string, reclaim bool) (string, error) {
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
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[%s] already locked by [%s], try to reclaim...", id, e)
			}
			reclaimed, status := d.reclaim(id, e.TxID)
			if !reclaimed {
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("[%s] already locked by [%s], reclaim failed, tx status [%s]", id, e, status)
				}
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					return e.TxID, errors.Errorf("already locked by [%s]", e)
				}
				return e.TxID, AlreadyLockedError
			}
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[%s] already locked by [%s], reclaimed successful, tx status [%s]", id, e, status)
			}
		} else {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("[%s] already locked by [%s], no reclaim", id, e)
			}
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				return e.TxID, errors.Errorf("already locked by [%s]", e)
			}
			return e.TxID, AlreadyLockedError
		}
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("locking [%s] for [%s]", id, txID)
	}
	now := time.Now()
	d.locked[k] = &lockEntry{TxID: txID, Created: now, LastAccess: now}
	return "", nil
}

// UnlockIDs unlocks the passed IDS. It returns the list of tokens that were not locked in the first place among
// those passed.
func (d *locker) UnlockIDs(ids ...*token2.ID) []*token2.ID {
	d.lock.Lock()
	defer d.lock.Unlock()

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("unlocking tokens [%v]", ids)
	}
	var notFound []*token2.ID
	for _, id := range ids {
		k := *id
		entry, ok := d.locked[k]
		if !ok {
			notFound = append(notFound, &k)
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Warnf("unlocking [%s] hold by no one, skipping", id, entry)
			}
			continue
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("unlocking [%s] hold by [%s]", id, entry)
		}
		delete(d.locked, k)
	}
	return notFound
}

func (d *locker) UnlockByTxID(txID string) {
	d.lock.Lock()
	defer d.lock.Unlock()

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("unlocking tokens hold by [%s]", txID)
	}
	for id, entry := range d.locked {
		if entry.TxID == txID {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("unlocking [%s] hold by [%s]", id, entry)
			}
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

func (d *locker) reclaim(id *token2.ID, txID string) (bool, int) {
	status, err := d.vault.Status(txID)
	if err != nil {
		return false, status
	}
	switch status {
	case Invalid:
		delete(d.locked, *id)
		return true, status
	default:
		return false, status
	}
}

func (d *locker) Start() {
	go d.scan()
}

func (d *locker) scan() {
	for {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("token collector: scan locked tokens")
		}
		var removeList []token2.ID
		d.lock.RLock()
		for id, entry := range d.locked {
			status, err := d.vault.Status(entry.TxID)
			if err != nil {
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Warnf("failed getting status for token [%s] locked by [%s], remove", id, entry)
				}
				removeList = append(removeList, id)
				continue
			}
			switch status {
			case Valid:
				// remove only if elapsed enough time from last access, to avoid concurrency issue
				if time.Since(entry.LastAccess) > d.validTxEvictionTimeout {
					removeList = append(removeList, id)
					if logger.IsEnabledFor(zapcore.DebugLevel) {
						logger.Debugf("token [%s] locked by [%s] in status [%s], time elapsed, remove", id, entry, status)
					}
				}
			case Invalid:
				removeList = append(removeList, id)
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("token [%s] locked by [%s] in status [%s], remove", id, entry, status)
				}
			default:
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("token [%s] locked by [%s] in status [%s], skip", id, entry, status)
				}
			}
		}
		d.lock.RUnlock()

		d.lock.Lock()
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("token collector: freeing [%d] items", len(removeList))
		}
		for _, s := range removeList {
			delete(d.locked, s)
		}
		d.lock.Unlock()

		for {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("token collector: sleep for some time...")
			}
			time.Sleep(d.sleepTimeout)
			d.lock.RLock()
			l := len(d.locked)
			d.lock.RUnlock()
			if l > 0 {
				// time to do some token collection
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("token collector: time to do some token collection, [%d] locked", l)
				}
				break
			}
		}
	}
}
