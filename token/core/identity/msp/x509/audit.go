/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package x509

import "encoding/json"

type AuditInfo struct {
	EnrollmentId     string
	RevocationHandle string
}

func (a *AuditInfo) Bytes() ([]byte, error) {
	return json.Marshal(a)
}

func (a *AuditInfo) FromBytes(raw []byte) error {
	return json.Unmarshal(raw, a)
}

func (a *AuditInfo) EnrollmentID() string {
	return string(a.EnrollmentId)
}

//RevocationHandle
func (a *AuditInfo) GetRevocationHandle() string {
	return string(a.RevocationHandle)
}
