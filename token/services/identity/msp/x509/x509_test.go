/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeserializer(t *testing.T) {
	p, err := NewProvider("./testdata/msp", "", "apple", nil)
	assert.NoError(t, err)
	id, auditInfo, err := p.Identity(nil)
	assert.NoError(t, err)
	eID := p.EnrollmentID()
	ai := &AuditInfo{}
	err = ai.FromBytes(auditInfo)
	assert.NoError(t, err)

	assert.Equal(t, eID, ai.EnrollmentId)
	assert.Equal(t, "auditor.org1.example.com", eID)

	des := &Deserializer{}
	info, err := des.Info(id, nil)
	assert.NoError(t, err)
	assert.Equal(t, "MSP.x509: [f+hVlmGaPejN2G0XDcESSMX2ol29WPcPQ+Fp3lOARBQ=][apple][auditor.org1.example.com]", info)
}
