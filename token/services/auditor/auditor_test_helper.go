/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditor

import "time"

// CalculateBackoff is a test helper that exposes the private calculateBackoff method for testing.
// This allows unit tests to verify the backoff calculation logic without making the method public.
func (a *Service) CalculateBackoff(attempt int) time.Duration {
	return a.calculateBackoff(attempt)
}

// Made with Bob
