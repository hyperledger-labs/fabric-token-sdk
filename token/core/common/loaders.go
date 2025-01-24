/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type TokenVault interface {
	IsPending(id *token.ID) (bool, error)
	GetTokenOutputsAndMeta(ctx context.Context, ids []*token.ID) ([][]byte, [][]byte, []token.Format, error)
	GetTokenOutputs(ids []*token.ID, callback driver.QueryCallbackFunc) error
	UnspentTokensIteratorBy(ctx context.Context, id string, tokenType token.Type) (driver.UnspentTokensIterator, error)
	ListHistoryIssuedTokens() (*token.IssuedTokens, error)
	PublicParams() ([]byte, error)
	Balance(id string, tokenType token.Type) (uint64, error)
}

type LedgerToken interface {
	GetOwner() []byte
}

type TokenDeserializer[T any] interface {
	DeserializeToken([]byte) (T, error)
}

type MetadataDeserializer[M any] interface {
	DeserializeMetadata([]byte) (M, error)
}

type TokenAndMetadataDeserializer[T any, M any] interface {
	TokenDeserializer[T]
	MetadataDeserializer[M]
}

type VaultLedgerTokenLoader[T any] struct {
	Logger       logging.Logger
	TokenVault   TokenVault
	Deserializer TokenDeserializer[T]

	// Variables used to control retry condition
	NumRetries int
	RetryDelay time.Duration
}

func NewLedgerTokenLoader[T any](logger logging.Logger, _ trace.TracerProvider, tokenVault TokenVault, deserializer TokenDeserializer[T]) *VaultLedgerTokenLoader[T] {
	return &VaultLedgerTokenLoader[T]{
		Logger:       logger,
		TokenVault:   tokenVault,
		Deserializer: deserializer,
		NumRetries:   6,
		RetryDelay:   1 * time.Second,
	}
}

// GetTokenOutputs takes an array of token identifiers (txID, index) and returns the corresponding token outputs
func (s *VaultLedgerTokenLoader[T]) GetTokenOutputs(ctx context.Context, ids []*token.ID) (map[string]T, error) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent("start_get_token_outputs")
	defer span.AddEvent("end_get_token_outputs")

	var err error
	for i := 0; i < s.NumRetries; i++ {
		span.AddEvent("try_fetch")
		tokens := make(map[string]T, len(ids))
		counter := 0
		err = s.TokenVault.GetTokenOutputs(ids, func(id *token.ID, bytes []byte) error {
			if len(bytes) == 0 {
				return errors.Errorf("failed getting serialized token output for id [%v], nil value", id)
			}
			ti, err := s.Deserializer.DeserializeToken(bytes)
			if err != nil {
				return errors.Wrapf(err, "failed deserializing token for id [%v][%s]", id, string(bytes))
			}
			tokens[id.TxId] = ti
			counter++
			return nil
		})
		if err == nil {
			s.Logger.Debugf("retrieve [%d] token outputs for [%v]", len(tokens), ids)
			return tokens, nil
		}
		s.Logger.Debugf("failed to retrieve tokens for [%v], any pending transaction? [%s]", ids, err)

		// check if there is any token id whose corresponding transaction is pending
		// if there is, then wait a bit and retry to load the outputs
		span.AddEvent("check_any_pending")
		anyPending, anyError := s.isAnyPending(ids...)
		if anyError != nil {
			err = anyError
			break
		}
		if anyError == nil && !anyPending {
			s.Logger.Debugf("failed to retrieve tokens: no transaction is pending")
			break
		}

		if lastRetry := s.NumRetries - 1; i < lastRetry {
			time.Sleep(s.RetryDelay)
		}
	}
	s.Logger.Debugf("failed to retrieve tokens [%s]", err)

	return nil, errors.Wrapf(err, "failed to get token outputs")
}

func (s *VaultLedgerTokenLoader[T]) isAnyPending(ids ...*token.ID) (anyPending bool, anyError error) {
	for _, id := range ids {
		if pending, error := s.TokenVault.IsPending(id); pending || error != nil {
			return pending, error
		}
	}
	return false, nil
}

type LoadedToken[T any, M any] struct {
	TokenFormat token.Format
	Token       T
	Metadata    M
}

type VaultLedgerTokenAndMetadataLoader[T any, M any] struct {
	TokenVault   TokenVault
	Deserializer TokenAndMetadataDeserializer[T, M]
}

func NewVaultLedgerTokenAndMetadataLoader[T any, M any](tokenVault TokenVault, deserializer TokenAndMetadataDeserializer[T, M]) *VaultLedgerTokenAndMetadataLoader[T, M] {
	return &VaultLedgerTokenAndMetadataLoader[T, M]{TokenVault: tokenVault, Deserializer: deserializer}
}

// LoadTokens takes an array of token identifiers (txID, index) and returns the keys in the vault
// matching the token identifiers, the corresponding zkatdlog tokens, the information of the
// tokens in clear text and the identities of their owners
// LoadToken returns an error in case of failure
func (s *VaultLedgerTokenAndMetadataLoader[T, M]) LoadTokens(ctx context.Context, ids []*token.ID) ([]LoadedToken[T, M], error) {
	span := trace.SpanFromContext(ctx)
	// return token outputs and the corresponding opening
	outputs, metadata, types, err := s.TokenVault.GetTokenOutputsAndMeta(ctx, ids)
	if err != nil {
		return nil, err
	}

	span.AddEvent("iterate_tokens")
	result := make([]LoadedToken[T, M], len(ids))
	for i, id := range ids {
		if len(outputs[i]) == 0 {
			return nil, errors.Errorf("failed getting state for id [%v], nil comm value", id)
		}
		if len(metadata[i]) == 0 {
			return nil, errors.Errorf("failed getting state for id [%v], nil info value", id)
		}
		span.AddEvent("deserialize_token")
		tok, err := s.Deserializer.DeserializeToken(outputs[i])
		if err != nil {
			return nil, errors.Wrapf(err, "failed deserializing token for id [%v][%s]", id, string(outputs[i]))
		}
		span.AddEvent("deserialize_metadata")
		meta, err := s.Deserializer.DeserializeMetadata(metadata[i])
		if err != nil {
			return nil, errors.Wrapf(err, "failed deserializeing token info for id [%v]", id)
		}
		result[i] = LoadedToken[T, M]{
			TokenFormat: types[i],
			Token:       tok,
			Metadata:    meta,
		}
	}

	return result, nil
}

type VaultTokenInfoLoader[M any] struct {
	TokenVault   driver.QueryEngine
	Deserializer MetadataDeserializer[M]
}

func NewVaultTokenInfoLoader[M any](tokenVault driver.QueryEngine, deserializer MetadataDeserializer[M]) *VaultTokenInfoLoader[M] {
	return &VaultTokenInfoLoader[M]{TokenVault: tokenVault, Deserializer: deserializer}
}

func (s *VaultTokenInfoLoader[M]) GetTokenInfos(ids []*token.ID) ([]M, error) {
	infos, err := s.TokenVault.GetTokenMetadata(ids)
	if err != nil {
		return nil, err
	}

	inputInf := make([]M, len(ids))
	for i, bytes := range infos {
		ti, err := s.Deserializer.DeserializeMetadata(bytes)
		if err != nil {
			return nil, errors.Wrapf(err, "failed deserializeing token info for id [%v]", ids[i])
		}
		inputInf[i] = ti
	}
	return inputInf, nil
}

type VaultTokenLoader struct {
	TokenVault driver.QueryEngine
}

func NewVaultTokenLoader(tokenVault driver.QueryEngine) *VaultTokenLoader {
	return &VaultTokenLoader{TokenVault: tokenVault}
}

// GetTokens takes an array of token identifiers (txID, index) and returns the keys of the identified tokens
// in the vault and the content of the tokens
func (s *VaultTokenLoader) GetTokens(ids []*token.ID) ([]*token.Token, error) {
	return s.TokenVault.GetTokens(ids...)
}

type TokenCertificationStorage interface {
	GetCertifications(ids []*token.ID) ([][]byte, error)
}

type VaultTokenCertificationLoader struct {
	TokenCertificationStorage TokenCertificationStorage
}

func (s *VaultTokenCertificationLoader) GetCertifications(ids []*token.ID) ([][]byte, error) {
	return s.TokenCertificationStorage.GetCertifications(ids)
}

type IdentityTokenAndMetadataDeserializer struct{}

func (i IdentityTokenAndMetadataDeserializer) DeserializeToken(bytes []byte) ([]byte, error) {
	return bytes, nil
}

func (i IdentityTokenAndMetadataDeserializer) DeserializeMetadata(bytes []byte) ([]byte, error) {
	return bytes, nil
}
