/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditor

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestMetricsProviderCall(t *testing.T) {
	// Call the Add, Set, Observe methods
	m := newMetrics(&noopProvider{})
	
	// Those are returning pointers to struct that implemented interfaces, let's call the interface methods
	assert.NotPanics(t, func() {
		m.AuditLockConflicts.Add(1)
		m.AppendErrors.Add(1)
		m.ReleasesTotal.Add(1)
		
		m.AuditDuration.Observe(1.0)
		m.AppendDuration.Observe(1.0)
	})
	
	nc := &noopCounter{}
	assert.NotPanics(t, func() {
		nc.Add(12)
	})
	
	ng := &noopGauge{}
	assert.NotPanics(t, func() {
		ng.Add(12)
		ng.Set(12)
	})

	nh := &noopHistogram{}
	assert.NotPanics(t, func() {
		nh.Observe(12)
	})
}
