/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package selector

import (
	"time"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type QueryService interface {
	ListUnspentTokens() (*token2.UnspentTokens, error)
}

type QueryEngine interface {
	ListUnspentTokens() (*token2.UnspentTokens, error)
}

type CertificationClient interface {
	IsCertified(id *token2.Id) bool
	RequestCertification(ids ...*token2.Id) error
}

type CertClient interface {
	IsCertified(id *token2.Id) bool
	RequestCertification(ids ...*token2.Id) error
}

type Locker interface {
	Lock(id *token2.Id, txID string) error
	UnlockIDs(id ...*token2.Id)
	UnlockByTxID(txID string)
}

type selector struct {
	txID         string
	locker       Locker
	queryService QueryService
	certClient   CertClient
	precision    uint64

	numRetry             int
	timeout              time.Duration
	requestCertification bool
}

func newSelector(txID string, locker Locker, service QueryService, certClient CertClient, numRetry int, timeout time.Duration, requestCertification bool) *selector {
	return &selector{
		txID:                 txID,
		locker:               locker,
		queryService:         service,
		certClient:           certClient,
		precision:            keys.Precision,
		numRetry:             numRetry,
		timeout:              timeout,
		requestCertification: requestCertification,
	}
}

// Select selects tokens to be spent based on ownership, quantity, and type
func (s *selector) Select(ownerFilter token.OwnerFilter, q, tokenType string) ([]*token2.Id, token2.Quantity, error) {
	if ownerFilter == nil {
		ownerFilter = &allOwners{}
	}

	var toBeSpent []*token2.Id
	var sum token2.Quantity
	var potentialSumWithLocked token2.Quantity
	var potentialSumWithNonCertified token2.Quantity
	target, err := token2.ToQuantity(q, s.precision)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to convert quantity")
	}

	i := 0
	for {
		logger.Debugf("start token selection, iteration [%d/%d]", i, s.numRetry)
		unspentTokens, err := s.queryService.ListUnspentTokens()
		if err != nil {
			return nil, nil, errors.Wrap(err, "token selection failed")
		}
		logger.Debugf("select token for a quantity of [%s] of type [%s] from [%d] unspent tokens", q, tokenType, len(unspentTokens.Tokens))

		// First select only certified
		sum = token2.NewZeroQuantity(s.precision)
		potentialSumWithLocked = token2.NewZeroQuantity(s.precision)
		potentialSumWithNonCertified = token2.NewZeroQuantity(s.precision)
		toBeSpent = nil
		var toBeCertified []*token2.Id
		var locked []*token2.Id

		for _, t := range unspentTokens.Tokens {
			q, err := token2.ToQuantity(t.Quantity, s.precision)
			if err != nil {
				s.locker.UnlockIDs(toBeSpent...)
				s.locker.UnlockIDs(toBeCertified...)
				return nil, nil, errors.Wrap(err, "failed to convert quantity")
			}

			logger.Debugf("select token [%s,%s,%v]?", q, tokenType, ownerFilter.Contains(t.Owner.Raw))

			// check type and ownership
			if t.Type != tokenType {
				logger.Debugf("token [%s,%s,%v] type does not match", q, tokenType, ownerFilter.Contains(t.Owner.Raw))
				continue
			}

			if !ownerFilter.Contains(t.Owner.Raw) {
				logger.Debugf("token [%s,%s,%v] owner does not belong to the passed wallet", q, tokenType, ownerFilter.Contains(t.Owner.Raw))
				continue
			}

			// lock the token
			if err := s.locker.Lock(t.Id, s.txID); err != nil {
				locked = append(locked, t.Id)
				potentialSumWithLocked = potentialSumWithLocked.Add(q)

				logger.Debugf("token [%s,%s,%v] cannot be locked [%s]", q, tokenType, ownerFilter.Contains(t.Owner.Raw), err)
				continue
			}

			// check certification, if needed
			if s.certClient != nil && !s.certClient.IsCertified(t.Id) {
				toBeCertified = append(toBeCertified, t.Id)
				potentialSumWithNonCertified = potentialSumWithNonCertified.Add(q)

				logger.Debugf("token [%s,%s,%v] is not certified, skipping", q, tokenType, ownerFilter.Contains(t.Owner.Raw))
				continue
			}

			// Append token
			logger.Debugf("adding quantity [%s]", q.Decimal())
			toBeSpent = append(toBeSpent, t.Id)
			sum = sum.Add(q)
			potentialSumWithLocked = potentialSumWithLocked.Add(q)
			potentialSumWithNonCertified = potentialSumWithNonCertified.Add(q)

			if target.Cmp(sum) <= 0 {
				break
			}
		}

		if target.Cmp(sum) <= 0 {
			return toBeSpent, sum, nil
		}

		// if we reached this point is because there are not enough funds but why?

		// Maybe it is a certification issue?
		if target.Cmp(potentialSumWithNonCertified) <= 0 && s.requestCertification {
			logger.Warnf("token selection failed: missing certifications, request them for [%s]", toBeCertified)
			// request certification
			err := s.certClient.RequestCertification(toBeCertified...)
			if err == nil {
				// TODO: refine this
				return append(toBeSpent, toBeCertified...), potentialSumWithNonCertified, nil
			}
			logger.Warnf("token selection failed: failed requesting token certification for [%v]: [%s]", toBeCertified, err)
		}

		// Unlock and check the conditions for a retry
		s.locker.UnlockIDs(toBeSpent...)
		s.locker.UnlockIDs(toBeCertified...)

		if target.Cmp(potentialSumWithLocked) <= 0 && potentialSumWithLocked.Cmp(sum) != 0 {
			// funds are potentially enough but they are locked
			logger.Debugf("token selection: sufficient funds but partially locked")
		}
		if target.Cmp(potentialSumWithNonCertified) <= 0 && potentialSumWithNonCertified.Cmp(sum) != 0 {
			// funds are potentially enough but they are locked
			logger.Debugf("token selection: sufficient funds but partially not certified")
		}

		i++
		if i >= s.numRetry {
			// it is time to fail but how?
			if target.Cmp(potentialSumWithLocked) <= 0 && potentialSumWithLocked.Cmp(sum) != 0 {
				// funds are potentially enough but they are locked
				logger.Debugf("token selection: it is time to fail but how, sufficient funds but locked")
				return nil, nil, errors.WithMessagef(
					token.SelectorSufficientButLockedFunds,
					"token selection failed: sufficient but partially locked funds, potential [%s] tokens of type [%s] are available", potentialSumWithLocked, tokenType,
				)
			}

			if target.Cmp(potentialSumWithNonCertified) <= 0 && potentialSumWithNonCertified.Cmp(sum) != 0 {
				// funds are potentially enough but they are locked
				logger.Debugf("token selection: it is time to fail but how, sufficient funds but locked")
				return nil, nil, errors.WithMessagef(
					token.SelectorSufficientButNotCertifiedFunds,
					"token selection failed: sufficient but partially not certified, potential [%s] tokens of type [%s] are available", potentialSumWithNonCertified, tokenType,
				)
			}

			// funds are insufficient
			logger.Debugf("token selection: it is time to fail but how, insufficient funds")
			return nil, nil, errors.WithMessagef(
				token.SelectorInsufficientFunds,
				"token selection failed: insufficient funds, only [%s] tokens of type [%s] are available", sum, tokenType,
			)
		}

		logger.Debugf("token selection: let's wait [%v] before retry...", s.timeout)
		time.Sleep(s.timeout)
	}
}

type allOwners struct{}

func (a *allOwners) Contains(identity view.Identity) bool {
	return true
}
