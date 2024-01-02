/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mailman

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

var logger = flogging.MustGetLogger("mailman")

// Mailman implements a token selector service that is targeting high concurrent use
type Mailman struct {
	isRunning       bool
	queryChannel    chan query
	updateChannel   chan batch
	stopRequest     chan bool
	hasStopped      chan bool
	availableTokens Queue
}

func NewMailman(tokenIDs ...*token.ID) *Mailman {
	availableTokens := NewQueue(len(tokenIDs))
	for _, k := range tokenIDs {
		availableTokens.Push(k)
	}

	return &Mailman{
		availableTokens: availableTokens,
	}
}

func (m *Mailman) Start() {
	if m.isRunning {
		logger.Warn("Mailman already running")
		return
	}

	m.queryChannel = make(chan query, 1024)
	m.updateChannel = make(chan batch, 1024)
	m.stopRequest = make(chan bool)
	m.hasStopped = make(chan bool)

	// start storage
	go func() {
		defer func() {
			// close all our communication channels
			close(m.queryChannel)
			close(m.updateChannel)
			close(m.hasStopped)
		}()
		for {
			select {
			case <-m.stopRequest:
				return
			case upt := <-m.updateChannel:
				// process mailman update
				m.processBatchUpdate(upt.updates...)
			case req := <-m.queryChannel:
				// process mailman query
				m.processQuery(&req)
			}
		}
	}()

	m.isRunning = true
	logger.Debugf("Mailman started")
}

func (m *Mailman) processQuery(req *query) {
	defer close(req.responseChanel)

	// find free token in rotating caching
	next := m.findNext()

	if next == nil {
		logger.Debugf("no tokens available, return empty response")
		req.responseChanel <- queryResponse{}
		return
	}

	logger.Debugf("processQuery returns %v", *next)
	req.responseChanel <- queryResponse{tokenID: *next}
}

func (m *Mailman) findNext() *token.ID {
	next, ok := m.availableTokens.Pop()
	if !ok {
		// no available token found
		return nil
	}

	return next
}

func (m *Mailman) processUpdate(update *update) {
	logger.Debugf("processUpdate %v", update)
	switch update.op {
	case Add, Unlock:
		//if m.getUnspentToken(&update.tokenID) != nil {
		m.availableTokens.Push(&update.tokenID)
		//}
	case Del:
		m.availableTokens.Remove(&update.tokenID)
	default:
		// just ignore update
	}
}

func (m *Mailman) processBatchUpdate(updates ...update) {
	for _, upd := range updates {
		m.processUpdate(&upd)
	}
}

func (m *Mailman) Stop() {
	close(m.stopRequest)
	<-m.hasStopped
	m.isRunning = false
	logger.Debugf("Mailman stopped")
}

func (m *Mailman) Query(r *query) {
	if !m.isRunning {
		return
	}

	m.queryChannel <- *r
}

func (m *Mailman) Update(u ...update) {
	if !m.isRunning {
		return
	}

	b := batch{u}
	m.updateChannel <- b
}

type query struct {
	responseChanel chan queryResponse
}

type queryResponse struct {
	tokenID token.ID
	err     error
}

type Op uint8

const (
	Add Op = iota
	Del
	Unlock
)

type update struct {
	op      Op
	tokenID token.ID
}

type batch struct {
	updates []update
}
