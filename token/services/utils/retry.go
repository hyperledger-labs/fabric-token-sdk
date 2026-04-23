/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// This file provides retry logic with exponential backoff and context cancellation support.
// RetryRunner executes functions with configurable retry strategies, supporting both
// fixed and exponential backoff delays with a maximum delay cap.
package utils

import (
	"context"
	"errors"
	"time"

	logging2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

// RetryRunner receives a function that potentially fails and retries according to the specified strategy
type RetryRunner interface {
	Run(func() error) error
	// RunWithContext retries like Run but stops early if ctx is canceled.
	RunWithContext(ctx context.Context, runner func() error) error
	// RunWithErrors retries until runner returns true or maxTimes is exhausted.
	RunWithErrors(runner func() (bool, error)) error
	// RunWithErrorsContext retries like RunWithErrors but stops early if ctx is canceled.
	RunWithErrorsContext(ctx context.Context, runner func() (bool, error)) error
}

var ErrMaxRetriesExceeded = errors.New("maximum number of retries exceeded")

const Infinitely = -1

// defaultMaxDelay caps exponential backoff to prevent workers from sleeping for
// unbounded durations (hours/days) after a burst of transient failures.
const defaultMaxDelay = 30 * time.Second

func NewRetryRunner(logger logging2.Logger, maxTimes int, delay time.Duration, expBackoff bool) *retryRunner {
	return &retryRunner{
		initialDelay: delay,
		maxDelay:     defaultMaxDelay,
		expBackoff:   expBackoff,
		maxTimes:     maxTimes,
		logger:       logger,
	}
}

type retryRunner struct {
	initialDelay time.Duration
	// maxDelay caps the exponential backoff. Zero means no cap.
	maxDelay   time.Duration
	expBackoff bool
	maxTimes   int
	logger     logging2.Logger
}

func (f *retryRunner) nextDelay(delay time.Duration) time.Duration {
	if delay == 0 || !f.expBackoff {
		return f.initialDelay
	}
	next := 2 * delay
	if f.maxDelay > 0 && next > f.maxDelay {
		return f.maxDelay
	}

	return next
}

// Run retries runner until it succeeds or maxTimes is exhausted.
// Uses context.Background() — prefer RunWithContext for cancelable retries.
func (f *retryRunner) Run(runner func() error) error {
	return f.RunWithContext(context.Background(), runner)
}

// RunWithContext retries runner until it succeeds, ctx is canceled, or maxTimes
// attempts are exhausted. The backoff sleep respects ctx cancellation so callers
// (e.g. queue workers shutting down) are not blocked for the full sleep duration.
func (f *retryRunner) RunWithContext(ctx context.Context, runner func() error) error {
	var (
		errs  []error
		delay time.Duration
	)
	for i := 0; f.maxTimes < 0 || i < f.maxTimes; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := runner(); err == nil {
			return nil
		} else {
			errs = append(errs, err)
		}
		delay = f.nextDelay(delay)
		f.logger.Warnf("Will retry iteration [%d] after a delay of [%v]. %d errors returned so far", i+1, delay, len(errs))
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if len(errs) == 0 {
		return ErrMaxRetriesExceeded
	}

	return errors.Join(errs...)
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

// RunWithErrorsContext retries until runner() returns true, ctx is canceled, or maxTimes
// attempts are exhausted. The backoff sleep respects ctx cancellation so callers
// are not blocked for the full sleep duration.
func (f *retryRunner) RunWithErrorsContext(ctx context.Context, runner func() (bool, error)) error {
	var (
		errs  []error
		delay time.Duration
	)
	for i := 0; f.maxTimes < 0 || i < f.maxTimes; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		terminate, err := runner()
		if terminate {
			return err
		}
		if err != nil {
			errs = append(errs, err)
		}
		delay = f.nextDelay(delay)
		f.logger.Warnf("Will retry iteration [%d] after a delay of [%v]. %d errors returned so far", i+1, delay, len(errs))
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if len(errs) == 0 {
		return ErrMaxRetriesExceeded
	}
	return errors.Join(errs...)
}
