/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interactive

import (
	"context"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/processor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type Op uint8

const (
	Add Op = iota
)

var logger = flogging.MustGetLogger("token-sdk.certifier.interactive")

type QueryEngine interface {
	UnspentTokensIterator() (network.UnspentTokensIterator, error)
}

type CertificationStorage interface {
	Exists(id *token.ID) bool
	Store(certifications map[*token.ID][]byte) error
}

type ViewManager interface {
	InitiateView(view view.View) (interface{}, error)
}

// CertificationClient scans the vault for tokens not yet certified and asks the certification.
type CertificationClient struct {
	ctx                  context.Context
	channel, namespace   string
	queryEngine          QueryEngine
	certificationStorage CertificationStorage
	viewManager          ViewManager
	certifiers           []view2.Identity
	eventOperationMap    map[string]Op
	// waitTime is used in case of a failure. It tells how much time to wait before retrying.
	waitTime    time.Duration
	maxAttempts int

	tokens    chan *token.ID
	batchSize int
}

func NewCertificationClient(
	ctx context.Context,
	channel string,
	namespace string,
	qe QueryEngine,
	cm CertificationStorage,
	fm ViewManager,
	certifiers []view2.Identity,
	notifier events.Subscriber,
	maxAttempts int,
	waitTime time.Duration,
) *CertificationClient {
	cc := &CertificationClient{
		ctx:                  ctx,
		channel:              channel,
		namespace:            namespace,
		queryEngine:          qe,
		certificationStorage: cm,
		viewManager:          fm,
		certifiers:           certifiers,
		waitTime:             waitTime,
		tokens:               make(chan *token.ID, 1000),
		batchSize:            10,
		maxAttempts:          maxAttempts,
	}

	eventOperationMap := make(map[string]Op)
	eventOperationMap[processor.AddToken] = Add
	if notifier != nil {
		for topic := range eventOperationMap {
			notifier.Subscribe(topic, cc)
		}
	}
	cc.eventOperationMap = eventOperationMap
	return cc
}

func (cc *CertificationClient) IsCertified(id *token.ID) bool {
	return cc.certificationStorage.Exists(id)
}

func (cc *CertificationClient) RequestCertification(ids ...*token.ID) error {
	var toBeCertified []*token.ID
	for _, id := range ids {
		if !cc.IsCertified(id) {
			toBeCertified = append(toBeCertified, id)
		}
	}
	if len(toBeCertified) == 0 {
		// all tokens already certified.
		return nil
	}

	var resultBoxed interface{}
	var err error
	for i := 0; i < cc.maxAttempts; i++ {
		resultBoxed, err = cc.viewManager.InitiateView(NewCertificationRequestView(cc.channel, cc.namespace, cc.certifiers[0], toBeCertified...))
		if err != nil {
			logger.Errorf("failed to request certification, try again [%d] after [%s]...", i, cc.waitTime)
			time.Sleep(cc.waitTime)
			continue
		}
		break
	}
	if err != nil {
		return err
	}
	certifications, ok := resultBoxed.(map[*token.ID][]byte)
	if !ok {
		return errors.Errorf("invalid type, expected map[token.ID][]byte")
	}
	if err := cc.certificationStorage.Store(certifications); err != nil {
		return err
	}
	return nil
}

func (cc *CertificationClient) Scan() error {
	logger.Debugf("check the certification of unspent tokens from the vault...")
	// Check the unspent tokens
	var err error
	tokens, err := cc.queryEngine.UnspentTokensIterator()
	if err != nil {
		return errors.WithMessagef(err, "failed to get an iterator over unspent tokens")
	}
	defer tokens.Close()

	var toBeCertified []*token.ID
	for {
		token, err := tokens.Next()
		if err != nil {
			logger.Errorf("failed to get next unspent tokens, stop here [%s]", err)
			break
		}
		if token == nil {
			break
		}

		// does token have a certification?
		if !cc.certificationStorage.Exists(token.Id) {
			// if no, batch it
			toBeCertified = append(toBeCertified, token.Id)
		}
	}

	if len(toBeCertified) != 0 {
		// Request certification
		logger.Debugf("request certification of [%v]", toBeCertified)
		if err := cc.RequestCertification(toBeCertified...); err != nil {
			return errors.WithMessagef(err, "failed retrieving certification")
		}
		logger.Debugf("request certification of [%v] satisfied with no error", toBeCertified)
	}

	return nil
}

func (cc *CertificationClient) Start() {
	go cc.accumulatorCutter()
}

func (cc *CertificationClient) OnReceive(event events.Event) {
	t, ok := event.Message().(processor.TokenMessage)
	if !ok {
		logger.Warnf("cannot cast to TokenMessage %v", event.Message())
		// drop this event
		return
	}

	// sanity check that we really registered for this type of event
	_, ok = cc.eventOperationMap[event.Topic()]
	if !ok {
		logger.Warnf("receive an event we did not registered for %v", event.Message())
		// drop this event
		return
	}

	// accumulate token
	if len(cc.tokens) >= cap(cc.tokens) {
		// skip this
		logger.Warnf("certification pipeline filled up, skipping id [%s:%d]", t.TxID, t.Index)
		return
	}
	cc.tokens <- &token.ID{
		TxId:  t.TxID,
		Index: t.Index,
	}
}

func (cc *CertificationClient) accumulatorCutter() {
	// TODO: introduce workers
	timeout := time.NewTimer(5 * time.Second)
	var accumulator []*token.ID
	for {
		select {
		case id := <-cc.tokens:
			logger.Debugf("Accumulate token [%s]", id)
			accumulator = append(accumulator, id)
			if len(accumulator) >= cc.batchSize {
				logger.Debugf("Limit reached, certify accumulator...")
				toCertify := accumulator
				accumulator = nil
				go cc.requestCertification(toCertify...)
			}
		case <-timeout.C:
			logger.Debugf("Timeout, certify accumulator...")
			toCertify := accumulator
			accumulator = nil
			go cc.requestCertification(toCertify...)
		case <-cc.ctx.Done():
			// time to close
			return
		}
	}
}

func (cc *CertificationClient) requestCertification(tokens ...*token.ID) {
	if len(tokens) == 0 {
		// no tokens passed, check the vault
		logger.Debugf("request certification of 0 tokens, check the vault...")
		if err := cc.Scan(); err != nil {
			logger.Errorf("failed to scan the vault for token to be certified [%s]", err)
		}
		return
	}
	logger.Debugf("request certification of [%v]", tokens)
	if err := cc.RequestCertification(tokens...); err != nil {
		// push back the ids
		logger.Warnf("failed retrieving certification [%s], push back token ids [%s]", err, tokens)
		for _, id := range tokens {
			cc.tokens <- id
		}
		return
	}
	logger.Debugf("request certification of [%v] satisfied with no error", tokens)
}
