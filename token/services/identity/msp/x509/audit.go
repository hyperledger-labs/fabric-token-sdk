/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import "encoding/json"

type AuditInfo struct {
	EID string
	RH  []byte
}

func (a *AuditInfo) Bytes() ([]byte, error) {
	return json.Marshal(a)
}

func (a *AuditInfo) FromBytes(raw []byte) error {
	return json.Unmarshal(raw, a)
}

func (a *AuditInfo) EnrollmentID() string {
	return a.EID
}

func (a *AuditInfo) RevocationHandle() string {
	return string(a.RH)
}
