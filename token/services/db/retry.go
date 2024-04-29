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
	errs := make([]error, 0)
	for i := 0; f.maxTimes < 0 || i < f.maxTimes; i++ {
		if err := runner(); err != nil {
			errs = append(errs, err)
			time.Sleep(f.nextDelay())
		} else {
			return nil
		}
	}
	return errors.Join(errs...)
}
