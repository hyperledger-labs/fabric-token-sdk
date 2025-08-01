/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"context"

	tdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type Deserializer interface {
	DeserializeVerifier(ctx context.Context, raw []byte) (tdriver.Verifier, error)
	DeserializeSigner(ctx context.Context, raw []byte) (tdriver.Signer, error)
	Info(ctx context.Context, raw []byte, auditInfo []byte) (string, error)
}

type DeserializerManager interface {
	AddDeserializer(ctx context.Context, deserializer Deserializer)
	DeserializeSigner(ctx context.Context, raw []byte) (tdriver.Signer, error)
}

type AuditInfo interface {
	EnrollmentID() string
	RevocationHandle() string
}

type AuditInfoDeserializer interface {
	DeserializeAuditInfo(context.Context, []byte) (AuditInfo, error)
}
