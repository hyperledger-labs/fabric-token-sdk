/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package inmemory

import (
	"fmt"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

var logger = flogging.MustGetLogger("token-sdk.selector.inmemory")

type Channel interface {
	Vault() *fabric.Vault
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
	ch                           Channel
	lock                         sync.RWMutex
	locked                       map[string]*lockEntry
	sleepTimeout                 time.Duration
	validTxEvictionTimeoutMillis int64
}

func NewLocker(ch Channel, timeout time.Duration, validTxEvictionTimeoutMillis int64) selector.Locker {
	r := &locker{
		ch:                           ch,
		sleepTimeout:                 timeout,
		lock:                         sync.RWMutex{},
		locked:                       map[string]*lockEntry{},
		validTxEvictionTimeoutMillis: validTxEvictionTimeoutMillis,
	}
	r.Start()
	return r
}

func (d *locker) Lock(id *token2.Id, txID string) (string, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	e, ok := d.locked[id.String()]
	if ok {
		e.LastAccess = time.Now()
		// Second chance
		logger.Debugf("[%s] already locked by [%s], try to reclaim...", id, e)
		reclaimed, status := d.reclaim(id, e.TxID)
		if !reclaimed {
			logger.Debugf("[%s] already locked by [%s], reclaim failed, tx status [%s]", id, e, status)
			return e.TxID, errors.Errorf("already locked by [%s]", e)
		}
		logger.Debugf("[%s] already locked by [%s], reclaimed successful, tx status [%s]", id, e, status)
	}
	logger.Debugf("locking [%s] for [%s]", id, txID)
	now := time.Now()
	d.locked[id.String()] = &lockEntry{TxID: txID, Created: now, LastAccess: now}
	return "", nil
}

func (d *locker) UnlockIDs(ids ...*token2.Id) {
	d.lock.Lock()
	defer d.lock.Unlock()

	logger.Debugf("unlocking tokens [%v]", ids)
	for _, id := range ids {
		entry, ok := d.locked[id.String()]
		if !ok {
			// TODO: shall we panic
			// return errors.Errorf("already locked by [%s]", tx)
			logger.Warnf("unlocking [%s] hold by no one, skipping", id, entry)
			continue
		}
		logger.Debugf("unlocking [%s] hold by [%s]", id, entry)
		delete(d.locked, id.String())
	}
}

func (d *locker) UnlockByTxID(txID string) {
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

func (d *locker) reclaim(id *token2.Id, txID string) (bool, fabric.ValidationCode) {
	status, _, err := d.ch.Vault().Status(txID)
	if err != nil {
		return false, status
	}
	switch status {
	case fabric.Invalid:
		delete(d.locked, id.String())
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
		logger.Debugf("garbage collector: scan locked tokens")
		var removeList []string
		d.lock.RLock()
		for id, entry := range d.locked {
			status, _, err := d.ch.Vault().Status(entry.TxID)
			if err != nil {
				logger.Warnf("failed getting status for token [%s] locked by [%s], remove", id, entry)
				removeList = append(removeList, id)
				continue
			}
			switch status {
			case fabric.Valid:
				// remove only if elapsed enough time from last access, to avoid concurrency issue
				if time.Now().Sub(entry.LastAccess).Milliseconds() > d.validTxEvictionTimeoutMillis {
					removeList = append(removeList, id)
					logger.Debugf("token [%s] locked by [%s] in status [%s], time elapsed, remove", id, entry, status)
				}
			case fabric.Invalid:
				removeList = append(removeList, id)
				logger.Debugf("token [%s] locked by [%s] in status [%s], remove", id, entry, status)
			default:
				logger.Debugf("token [%s] locked by [%s] in status [%s], skip", id, entry, status)
			}
		}
		d.lock.RUnlock()

		d.lock.Lock()
		logger.Debugf("garbage collector: freeing [%d] items", len(removeList))
		for _, s := range removeList {
			delete(d.locked, s)
		}
		d.lock.Unlock()

		for {
			logger.Debugf("garbage collector: sleep for some time...")
			time.Sleep(d.sleepTimeout)
			d.lock.RLock()
			l := len(d.locked)
			d.lock.RUnlock()
			if l > 0 {
				// time to do some garbage collection
				logger.Debugf("garbage collector: time to do some garbage collection, [%d] locked", l)
				break
			}
		}
	}
}
