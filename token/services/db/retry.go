/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"errors"
	"time"
)

// RetryRunner receives a function that potentially fails and retries according to the specified strategy
type RetryRunner interface {
	Run(func() error) error
}

var ErrMaxRetriesExceeded = errors.New("maximum number of retries exceeded")

const Infinitely = -1

func NewRetryRunner(maxTimes int, delay time.Duration, expBackoff bool) *retryRunner {
	return &retryRunner{
		delay:      delay,
		expBackoff: expBackoff,
		maxTimes:   maxTimes,
	}
}

type retryRunner struct {
	delay      time.Duration
	expBackoff bool
	maxTimes   int
}

func (f *retryRunner) nextDelay() time.Duration {
	if f.expBackoff {
		f.delay = 2 * f.delay
	}
	return f.delay
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
	for i := 0; f.maxTimes < 0 || i < f.maxTimes; i++ {
		terminate, err := runner()
		if terminate {
			return err
		}
		if err != nil {
			errs = append(errs, err)
		}
		time.Sleep(f.nextDelay())
	}
	if len(errs) == 0 {
		return ErrMaxRetriesExceeded
	}
	return errors.Join(errs...)
}
