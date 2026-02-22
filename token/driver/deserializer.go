/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import "context"

//go:generate counterfeiter -o mock/matcher_deserializer.go -fake-name MatcherDeserializer . MatcherDeserializer

// MatcherDeserializer is the interface for identity matchers' deserializer.
type MatcherDeserializer interface {
	GetAuditInfoMatcher(ctx context.Context, owner Identity, auditInfo []byte) (Matcher, error)
}

// VerifierDeserializer is the interface for verifiers' deserializer.
// A verifier checks the validity of a signature against the identity associated with the verifier
//
//go:generate counterfeiter -o mock/verifier_deserializer.go -fake-name VerifierDeserializer . VerifierDeserializer
type VerifierDeserializer interface {
	DeserializeVerifier(ctx context.Context, id Identity) (Verifier, error)
}

// AuditMatcherProvider provides audit related deserialization functionalities
//
//go:generate counterfeiter -o mock/audit_matcher_provider.go -fake-name AuditMatcherProvider . AuditMatcherProvider
type AuditMatcherProvider interface {
	MatcherDeserializer
	MatchIdentity(ctx context.Context, id Identity, ai []byte) error
	GetAuditInfo(ctx context.Context, id Identity, p AuditInfoProvider) ([]byte, error)
}

// RecipientExtractor extracts the recipients from an identity
//
//go:generate counterfeiter -o mock/recipient_extractor.go -fake-name RecipientExtractor . RecipientExtractor
type RecipientExtractor interface {
	Recipients(id Identity) ([]Identity, error)
}

//go:generate counterfeiter -o mock/token_deserializer.go -fake-name TokenDeserializer . TokenDeserializer
type TokenDeserializer[T any] interface {
	DeserializeToken([]byte) (T, error)
}

//go:generate counterfeiter -o mock/metadata_deserializer.go -fake-name MetadataDeserializer . MetadataDeserializer
type MetadataDeserializer[M any] interface {
	DeserializeMetadata([]byte) (M, error)
}

//go:generate counterfeiter -o mock/token_and_metadata_deserializer.go -fake-name TokenAndMetadataDeserializer . TokenAndMetadataDeserializer
type TokenAndMetadataDeserializer[T any, M any] interface {
	TokenDeserializer[T]
	MetadataDeserializer[M]
}
