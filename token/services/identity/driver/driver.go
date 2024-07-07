/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import tdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"

type Deserializer interface {
	DeserializeVerifier(raw []byte) (tdriver.Verifier, error)
	DeserializeSigner(raw []byte) (tdriver.Signer, error)
	Info(raw []byte, auditInfo []byte) (string, error)
}

type DeserializerManager interface {
	AddDeserializer(deserializer Deserializer)
	DeserializeSigner(raw []byte) (tdriver.Signer, error)
}

type AuditInfo interface {
	EnrollmentID() string
	RevocationHandle() string
}

type AuditInfoDeserializer interface {
	DeserializeAuditInfo([]byte) (AuditInfo, error)
}
