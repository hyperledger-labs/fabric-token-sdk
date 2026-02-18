/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"go.opentelemetry.io/otel/trace"
)

var logger = logging.MustGetLogger()

// VaultLedgerTokenLoader loads tokens from the vault ledger.
type VaultLedgerTokenLoader[T any] struct {
	Logger       logging.Logger
	TokenVault   driver.TokenVault
	Deserializer driver.TokenDeserializer[T]

	// Variables used to control retry condition
	NumRetries int
	RetryDelay time.Duration
}

// NewLedgerTokenLoader returns a new VaultLedgerTokenLoader instance.
func NewLedgerTokenLoader[T any](logger logging.Logger, _ trace.TracerProvider, tokenVault driver.TokenVault, deserializer driver.TokenDeserializer[T]) *VaultLedgerTokenLoader[T] {
	return &VaultLedgerTokenLoader[T]{
		Logger:       logger,
		TokenVault:   tokenVault,
		Deserializer: deserializer,
		NumRetries:   6,
		RetryDelay:   1 * time.Second,
	}
}

// GetTokenOutputs takes an array of token identifiers (txID, index) and returns the corresponding token outputs.
func (s *VaultLedgerTokenLoader[T]) GetTokenOutputs(ctx context.Context, ids []*token.ID) (map[string]T, error) {
	var err error
	for i := range s.NumRetries {
		tokens := make(map[string]T, len(ids))
		counter := 0
		err = s.TokenVault.GetTokenOutputs(ctx, ids, func(id *token.ID, bytes []byte) error {
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
			s.Logger.DebugfContext(ctx, "retrieve [%d] token outputs for [%v]", len(tokens), ids)

			return tokens, nil
		}
		s.Logger.DebugfContext(ctx, "failed to retrieve tokens for [%v], any pending transaction? [%s]", ids, err)

		// check if there is any token id whose corresponding transaction is pending
		// if there is, then wait a bit and retry to load the outputs
		anyPending, anyError := s.isAnyPending(ctx, ids...)
		if anyError != nil {
			err = anyError

			break
		}
		if anyError == nil && !anyPending {
			s.Logger.DebugfContext(ctx, "failed to retrieve tokens: no transaction is pending")

			break
		}

		if lastRetry := s.NumRetries - 1; i < lastRetry {
			time.Sleep(s.RetryDelay)
		}
	}
	s.Logger.DebugfContext(ctx, "failed to retrieve tokens [%s]", err)

	return nil, errors.Wrapf(err, "failed to get token outputs")
}

func (s *VaultLedgerTokenLoader[T]) isAnyPending(ctx context.Context, ids ...*token.ID) (anyPending bool, anyError error) {
	for _, id := range ids {
		if pending, error := s.TokenVault.IsPending(ctx, id); pending || error != nil {
			return pending, error
		}
	}

	return false, nil
}

// LoadedToken represents a token and its metadata loaded from the vault.
type LoadedToken[T any, M any] struct {
	TokenFormat token.Format
	Token       T
	Metadata    M
}

// VaultLedgerTokenAndMetadataLoader loads tokens and their metadata from the vault ledger.
type VaultLedgerTokenAndMetadataLoader[T any, M any] struct {
	TokenVault   driver.TokenVault
	Deserializer driver.TokenAndMetadataDeserializer[T, M]
}

// NewVaultLedgerTokenAndMetadataLoader returns a new VaultLedgerTokenAndMetadataLoader instance.
func NewVaultLedgerTokenAndMetadataLoader[T any, M any](tokenVault driver.TokenVault, deserializer driver.TokenAndMetadataDeserializer[T, M]) *VaultLedgerTokenAndMetadataLoader[T, M] {
	return &VaultLedgerTokenAndMetadataLoader[T, M]{TokenVault: tokenVault, Deserializer: deserializer}
}

// LoadTokens takes an array of token identifiers (txID, index) and returns the keys in the vault
// matching the token identifiers, the corresponding zkatdlog tokens, the information of the
// tokens in clear text and the identities of their owners.
func (s *VaultLedgerTokenAndMetadataLoader[T, M]) LoadTokens(ctx context.Context, ids []*token.ID) ([]LoadedToken[T, M], error) {
	// return token outputs and the corresponding opening
	outputs, metadata, types, err := s.TokenVault.GetTokenOutputsAndMeta(ctx, ids)
	if err != nil {
		return nil, err
	}

	logger.DebugfContext(ctx, "Deserialize %d tokens", len(ids))
	result := make([]LoadedToken[T, M], len(ids))
	for i, id := range ids {
		if len(outputs[i]) == 0 {
			return nil, errors.Errorf("failed getting state for id [%v], nil comm value", id)
		}
		if len(metadata[i]) == 0 {
			return nil, errors.Errorf("failed getting state for id [%v], nil info value", id)
		}
		tok, err := s.Deserializer.DeserializeToken(outputs[i])
		if err != nil {
			return nil, errors.Wrapf(err, "failed deserializing token for id [%v][%s]", id, string(outputs[i]))
		}
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

// VaultTokenInfoLoader loads token metadata from the vault.
type VaultTokenInfoLoader[M any] struct {
	TokenVault   driver.QueryEngine
	Deserializer driver.MetadataDeserializer[M]
}

// NewVaultTokenInfoLoader returns a new VaultTokenInfoLoader instance.
func NewVaultTokenInfoLoader[M any](tokenVault driver.QueryEngine, deserializer driver.MetadataDeserializer[M]) *VaultTokenInfoLoader[M] {
	return &VaultTokenInfoLoader[M]{TokenVault: tokenVault, Deserializer: deserializer}
}

// GetTokenInfos takes an array of token identifiers (txID, index) and returns the corresponding token metadata.
func (s *VaultTokenInfoLoader[M]) GetTokenInfos(ctx context.Context, ids []*token.ID) ([]M, error) {
	infos, err := s.TokenVault.GetTokenMetadata(ctx, ids)
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

// VaultTokenLoader loads tokens from the vault.
type VaultTokenLoader struct {
	TokenVault driver.QueryEngine
}

// NewVaultTokenLoader returns a new VaultTokenLoader instance.
func NewVaultTokenLoader(tokenVault driver.QueryEngine) *VaultTokenLoader {
	return &VaultTokenLoader{TokenVault: tokenVault}
}

// GetTokens takes an array of token identifiers (txID, index) and returns the tokens from the vault.
func (s *VaultTokenLoader) GetTokens(ctx context.Context, ids []*token.ID) ([]*token.Token, error) {
	return s.TokenVault.GetTokens(ctx, ids...)
}

// VaultTokenCertificationLoader loads token certifications.
type VaultTokenCertificationLoader struct {
	TokenCertificationStorage driver.TokenCertificationStorage
}

// GetCertifications returns certifications for the passed token IDs.
func (s *VaultTokenCertificationLoader) GetCertifications(ctx context.Context, ids []*token.ID) ([][]byte, error) {
	return s.TokenCertificationStorage.GetCertifications(ctx, ids)
}

// IdentityTokenAndMetadataDeserializer is a deserializer that returns the input bytes as is.
type IdentityTokenAndMetadataDeserializer struct{}

// DeserializeToken returns the input bytes as is.
func (i IdentityTokenAndMetadataDeserializer) DeserializeToken(bytes []byte) ([]byte, error) {
	return bytes, nil
}

// DeserializeMetadata returns the input bytes as is.
func (i IdentityTokenAndMetadataDeserializer) DeserializeMetadata(bytes []byte) ([]byte, error) {
	return bytes, nil
}
