/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package selector

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"

	"github.com/hashicorp/go-uuid"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type QueryService interface {
	UnspentTokensIterator() (*token.UnspentTokensIterator, error)
	UnspentTokensIteratorBy(id, typ string) (*token.UnspentTokensIterator, error)
	GetTokens(inputs ...*token2.ID) ([]*token2.Token, error)
}

type Locker interface {
	Lock(id *token2.ID, txID string, reclaim bool) (string, error)
	// UnlockIDs unlocks the passed IDS. It returns the list of tokens that were not locked in the first place among
	// those passed.
	UnlockIDs(ids ...*token2.ID) []*token2.ID
	UnlockByTxID(txID string)
	IsLocked(id *token2.ID) bool
}

type Tracer tracing.Tracer

type selector struct {
	txID         string
	locker       Locker
	queryService QueryService
	precision    uint64

	numRetry             int
	timeout              time.Duration
	requestCertification bool

	tracer Tracer
}

// Select selects tokens to be spent based on ownership, quantity, and type
func (s *selector) Select(ownerFilter token.OwnerFilter, q, tokenType string) ([]*token2.ID, token2.Quantity, error) {
	if ownerFilter == nil {
		ownerFilter = &allOwners{}
	}

	if len(ownerFilter.ID()) != 0 {
		return s.selectByID(ownerFilter, q, tokenType)
	}

	return s.selectByOwner(ownerFilter, q, tokenType)
}

func (s *selector) concurrencyCheck(ids []*token2.ID) error {
	_, err := s.queryService.GetTokens(ids...)
	return err
}

func (s *selector) selectByID(ownerFilter token.OwnerFilter, q string, tokenType string) ([]*token2.ID, token2.Quantity, error) {
	uuid, err := uuid.GenerateUUID()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate UUID")
	}
	s.tracer.Start("selector.selectByID" + uuid)
	defer s.tracer.End("selector.selectByID" + uuid)

	var toBeSpent []*token2.ID
	var sum token2.Quantity
	var potentialSumWithLocked token2.Quantity
	var potentialSumWithNonCertified token2.Quantity
	target, err := token2.ToQuantity(q, s.precision)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to convert quantity")
	}
	id := ownerFilter.ID()

	i := 0
	var unspentTokens *token.UnspentTokensIterator
	defer func() {
		if unspentTokens != nil {
			unspentTokens.Close()
		}
	}()
	for {
		if unspentTokens != nil {
			unspentTokens.Close()
		}
		logger.Debugf("start token selection, iteration [%d/%d]", i, s.numRetry)
		unspentTokens, err = s.queryService.UnspentTokensIteratorBy(id, tokenType)
		if err != nil {
			return nil, nil, errors.Wrap(err, "token selection failed")
		}
		logger.Debugf("select token for a quantity of [%s] of type [%s]", q, tokenType)

		// First select only certified
		sum = token2.NewZeroQuantity(s.precision)
		potentialSumWithLocked = token2.NewZeroQuantity(s.precision)
		potentialSumWithNonCertified = token2.NewZeroQuantity(s.precision)
		toBeSpent = nil
		var toBeCertified []*token2.ID

		reclaim := s.numRetry == 1 || i > 0
		numNext := 0
		for {
			t, err := unspentTokens.Next()
			numNext++
			if err != nil {
				return nil, nil, errors.Wrap(err, "token selection failed")
			}
			if t == nil {
				break
			}

			q, err := token2.ToQuantity(t.Quantity, s.precision)
			if err != nil {
				s.locker.UnlockIDs(toBeSpent...)
				s.locker.UnlockIDs(toBeCertified...)
				return nil, nil, errors.Wrap(err, "failed to convert quantity")
			}

			// lock the token
			if _, err := s.locker.Lock(t.Id, s.txID, reclaim); err != nil {
				potentialSumWithLocked = potentialSumWithLocked.Add(q)

				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("token [%s,%v] cannot be locked [%s]", q, tokenType, err)
				}
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

		concurrencyIssue := false
		if target.Cmp(sum) <= 0 {
			err := s.concurrencyCheck(toBeSpent)
			if err == nil {
				return toBeSpent, sum, nil
			}
			concurrencyIssue = true
			logger.Errorf("concurrency issue, some of the tokens might not exist anymore [%s]", err)
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
			if concurrencyIssue {
				logger.Debugf("concurrency issue, some of the tokens might not exist anymore")
				return nil, nil, errors.WithMessagef(
					token.SelectorSufficientFundsButConcurrencyIssue,
					"token selection failed: sufficient funs but concurrency issue, potential [%s] tokens of type [%s] were available", potentialSumWithLocked, tokenType,
				)
			}

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
				"token selection failed: insufficient funds, only [%s] tokens of type [%s] are available", sum.Decimal(), tokenType,
			)
		}

		logger.Debugf("token selection: let's wait [%v] before retry...", s.timeout)
		time.Sleep(s.timeout)
	}
}

func (s *selector) selectByOwner(ownerFilter token.OwnerFilter, q string, tokenType string) ([]*token2.ID, token2.Quantity, error) {
	var toBeSpent []*token2.ID
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
		unspentTokens, err := s.queryService.UnspentTokensIterator()
		if err != nil {
			return nil, nil, errors.Wrap(err, "token selection failed")
		}
		defer unspentTokens.Close()
		logger.Debugf("select token for a quantity of [%s] of type [%s]", q, tokenType)

		// First select only certified
		sum = token2.NewZeroQuantity(s.precision)
		potentialSumWithLocked = token2.NewZeroQuantity(s.precision)
		potentialSumWithNonCertified = token2.NewZeroQuantity(s.precision)
		toBeSpent = nil
		var toBeCertified []*token2.ID

		reclaim := s.numRetry == 1 || i > 0
		for {
			t, err := unspentTokens.Next()
			if err != nil {
				return nil, nil, errors.Wrap(err, "token selection failed")
			}
			if t == nil {
				break
			}

			q, err := token2.ToQuantity(t.Quantity, s.precision)
			if err != nil {
				s.locker.UnlockIDs(toBeSpent...)
				s.locker.UnlockIDs(toBeCertified...)
				return nil, nil, errors.Wrap(err, "failed to convert quantity")
			}

			// check type and ownership
			if t.Type != tokenType {
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("token [%s,%s] type does not match", q, tokenType)
				}
				continue
			}

			rightOwner := ownerFilter.ContainsToken(t)

			if !rightOwner {
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("token [%s,%s,%s,%v] owner does not belong to the passed wallet", view.Identity(t.Owner.Raw), q, tokenType, rightOwner)
				}
				continue
			}

			// lock the token
			if _, err := s.locker.Lock(t.Id, s.txID, reclaim); err != nil {
				potentialSumWithLocked = potentialSumWithLocked.Add(q)

				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("token [%s,%s,%v] cannot be locked [%s]", q, tokenType, rightOwner, err)
				}
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

		concurrencyIssue := false
		if target.Cmp(sum) <= 0 {
			err := s.concurrencyCheck(toBeSpent)
			if err == nil {
				return toBeSpent, sum, nil
			}
			concurrencyIssue = true
			logger.Errorf("concurrency issue, some of the tokens might not exist anymore [%s]", err)
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
			if concurrencyIssue {
				logger.Debugf("concurrency issue, some of the tokens might not exist anymore")
				return nil, nil, errors.WithMessagef(
					token.SelectorSufficientFundsButConcurrencyIssue,
					"token selection failed: sufficient funs but concurrency issue, potential [%s] tokens of type [%s] were available", potentialSumWithLocked, tokenType,
				)
			}

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

func (a *allOwners) ID() string {
	return ""
}

func (a *allOwners) ContainsToken(token *token2.UnspentToken) bool {
	return true
}
