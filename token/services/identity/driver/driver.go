/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"context"

	tdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// IdentityType identifies the type of identity
type IdentityType = tdriver.IdentityType

type TypedSignerDeserializer interface {
	DeserializeSigner(ctx context.Context, typ IdentityType, raw []byte) (tdriver.Signer, error)
}

type Deserializer interface {
	DeserializeVerifier(ctx context.Context, raw []byte) (tdriver.Verifier, error)
	DeserializeSigner(ctx context.Context, raw []byte) (tdriver.Signer, error)
	Info(ctx context.Context, raw []byte, auditInfo []byte) (string, error)
}

type SignerDeserializerManager interface {
	AddTypedSignerDeserializer(typ IdentityType, d TypedSignerDeserializer)
	DeserializeSigner(ctx context.Context, raw []byte) (tdriver.Signer, error)
}

type AuditInfo interface {
	EnrollmentID() string
	RevocationHandle() string
}

//go:generate counterfeiter -o mock/aides.go -fake-name AuditInfoDeserializer . AuditInfoDeserializer
type AuditInfoDeserializer interface {
	DeserializeAuditInfo(ctx context.Context, raw []byte) (AuditInfo, error)
}
