/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package inmemory

import (
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

var logger = flogging.MustGetLogger("token-sdk.selector.inmemory")

type Channel interface {
	Vault() *fabric.Vault
}

type dummyLocker struct {
}

func NewDummyLocker() selector.Locker {
	return &dummyLocker{}
}

func (d *dummyLocker) Lock(id *token2.Id, txID string) error {
	return nil
}

func (d *dummyLocker) UnlockIDs(id ...*token2.Id) {
	return
}

func (d *dummyLocker) UnlockByTxID(txID string) {
	return
}

type locker struct {
	ch      Channel
	lock    sync.RWMutex
	locked  map[string]string
	timeout time.Duration
}

func NewLocker(ch Channel, timeout time.Duration) selector.Locker {
	r := &locker{
		ch:      ch,
		timeout: timeout,
		lock:    sync.RWMutex{},
		locked:  map[string]string{},
	}
	r.Start()
	return r
}

func (d *locker) Lock(id *token2.Id, txID string) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	tx, ok := d.locked[id.String()]
	if ok {
		// Second chance
		logger.Debugf("[%s] already locked by [%s], try to reclaim...", id, tx)
		reclaimed, status := d.reclaim(id, tx)
		if !reclaimed {
			logger.Debugf("[%s] already locked by [%s], reclaim failed, tx status [%d]", id, tx, status)
			return errors.Errorf("already locked by [%s]", tx)
		}
		logger.Debugf("[%s] already locked by [%s], reclaimed successful, tx status [%d]", id, tx, status)
	}
	logger.Debugf("locking [%s] for [%s]", id, txID)
	d.locked[id.String()] = txID
	return nil
}

func (d *locker) UnlockIDs(ids ...*token2.Id) {
	d.lock.Lock()
	defer d.lock.Unlock()

	logger.Debugf("unlocking tokens [%v]", ids)
	for _, id := range ids {
		txID, ok := d.locked[id.String()]
		if !ok {
			// TODO: shall we panic
			// return errors.Errorf("already locked by [%s]", tx)
			logger.Warnf("unlocking [%s] hold by no one, skipping", id, txID)
			continue
		}
		logger.Debugf("unlocking [%s] hold by [%s]", id, txID)
		delete(d.locked, id.String())
	}
}

func (d *locker) UnlockByTxID(txID string) {
	d.lock.Lock()
	defer d.lock.Unlock()

	logger.Debugf("unlocking tokens hold by [%s]", txID)
	for id, tx := range d.locked {
		if tx == txID {
			logger.Debugf("unlocking [%s] hold by [%s]", id, txID)
			delete(d.locked, id)
		}
	}
}

func (d *locker) reclaim(id *token2.Id, txID string) (bool, fabric.ValidationCode) {
	status, _, err := d.ch.Vault().Status(txID)
	if err != nil {
		return false, status
	}
	if status == fabric.Invalid {
		delete(d.locked, id.String())
		return true, status
	}
	return false, status
}

func (d *locker) Start() {
	go d.scan()
}

func (d *locker) scan() {
	for {
		logger.Debugf("garbage collector: scan locked tokens")
		var removeList []string
		d.lock.RLock()
		for id, txID := range d.locked {
			status, _, err := d.ch.Vault().Status(txID)
			if err != nil {
				continue
			}
			switch status {
			case fabric.Valid:
				removeList = append(removeList, id)
			case fabric.Invalid:
				removeList = append(removeList, id)
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
			time.Sleep(d.timeout)
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
