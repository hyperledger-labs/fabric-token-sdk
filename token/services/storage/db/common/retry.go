/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"errors"
	"time"

	logging2 "github.com/hyperledger-labs/fabric-token-sdk/token/utils/logging"
)

// RetryRunner receives a function that potentially fails and retries according to the specified strategy
type RetryRunner interface {
	Run(func() error) error
}

var ErrMaxRetriesExceeded = errors.New("maximum number of retries exceeded")

const Infinitely = -1

func NewRetryRunner(maxTimes int, delay time.Duration, expBackoff bool) *retryRunner {
	return &retryRunner{
		initialDelay: delay,
		expBackoff:   expBackoff,
		maxTimes:     maxTimes,
		logger:       logging2.MustGetLogger(),
	}
}

type retryRunner struct {
	initialDelay time.Duration
	expBackoff   bool
	maxTimes     int
	logger       logging2.Logger
}

func (f *retryRunner) nextDelay(delay time.Duration) time.Duration {
	if delay == 0 || !f.expBackoff {
		return f.initialDelay
	}
	return 2 * delay
}

func (f *retryRunner) Run(runner func() error) error {
	return f.RunWithErrors(func() (bool, error) {
		err := runner()
		return err == nil, err
	})
}

// RunWithErrors will retry until runner() returns true or until it returns maxTimes false.
// If it returns true, then the error or nil will be returned.
// If it returns maxTimes false, then it will always return an error: either a join of all errors it encountered or a ErrMaxRetriesExceeded.
func (f *retryRunner) RunWithErrors(runner func() (bool, error)) error {
	errs := make([]error, 0)
	var delay time.Duration
	for i := 0; f.maxTimes < 0 || i < f.maxTimes; i++ {
		terminate, err := runner()
		if terminate {
			return err
		}
		if err != nil {
			errs = append(errs, err)
		}
		delay = f.nextDelay(delay)
		f.logger.Warnf("Will retry iteration [%d] after a delay of [%v]. %d errors returned so far", i+1, delay, len(errs))
		time.Sleep(delay)
	}
	if len(errs) == 0 {
		return ErrMaxRetriesExceeded
	}
	return errors.Join(errs...)
}
