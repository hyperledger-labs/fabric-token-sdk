/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type LedgerToken interface {
	GetOwner() []byte
}

type TokenDeserializer[T any] interface {
	DeserializeToken([]byte) (T, error)
}

type MetadataDeserializer[M any] interface {
	DeserializeMetadata([]byte) (M, error)
}

type TokenAndMetadataDeserializer[T LedgerToken, M any] interface {
	TokenDeserializer[T]
	MetadataDeserializer[M]
}

type VaultLedgerTokenLoader[T any] struct {
	TokenVault   TokenVault
	Deserializer TokenDeserializer[T]

	// Variables used to control retry condition
	NumRetries int
	RetryDelay time.Duration
}

func NewLedgerTokenLoader[T any](tokenVault TokenVault, deserializer TokenDeserializer[T], numRetries int, retryDelay time.Duration) *VaultLedgerTokenLoader[T] {
	return &VaultLedgerTokenLoader[T]{
		TokenVault:   tokenVault,
		Deserializer: deserializer,
		NumRetries:   numRetries,
		RetryDelay:   retryDelay,
	}
}

// GetTokenOutputs takes an array of token identifiers (txID, index) and returns the corresponding token outputs
func (s *VaultLedgerTokenLoader[T]) GetTokenOutputs(ids []*token.ID) ([]T, error) {
	var tokens []T

	var err error
	for i := 0; i < s.NumRetries; i++ {
		err = s.TokenVault.GetTokenOutputs(ids, func(id *token.ID, bytes []byte) error {
			if len(bytes) == 0 {
				return errors.Errorf("failed getting serialized token output for id [%v], nil value", id)
			}
			ti, err := s.Deserializer.DeserializeToken(bytes)
			if err != nil {
				return errors.Wrapf(err, "failed deserializing token for id [%v][%s]", id, string(bytes))
			}
			tokens = append(tokens, ti)
			return nil
		})
		if err == nil {
			return tokens, nil
		}

		// check if there is any token id whose corresponding transaction is pending
		// if there is, then wait a bit and retry to load the outputs
		retry := false
		for _, id := range ids {
			pending, err := s.TokenVault.IsPending(id)
			if err != nil {
				break
			}
			if pending {
				//logger.Warnf("failed getting serialized token output for id [%v] because the relative transaction is pending, retry at [%d]", id, i)
				if i >= s.NumRetries-1 {
					// too late, we tried already too many times
					return nil, errors.Wrapf(err, "failed to get token outputs, tx [%s] is still pending", id.TxId)
				}
				time.Sleep(s.RetryDelay)
				retry = true
				break
			}
		}

		if retry {
			tokens = nil
			continue
		}

		return nil, errors.Wrapf(err, "failed to get token outputs")
	}

	return nil, errors.Wrapf(err, "failed to get token outputs")
}

type VaultLedgerTokenAndMetadataLoader[T LedgerToken, M any] struct {
	TokenVault   TokenVault
	Deserializer TokenAndMetadataDeserializer[T, M]
}

func NewVaultLedgerTokenAndMetadataLoader[T LedgerToken, M any](tokenVault TokenVault, deserializer TokenAndMetadataDeserializer[T, M]) *VaultLedgerTokenAndMetadataLoader[T, M] {
	return &VaultLedgerTokenAndMetadataLoader[T, M]{TokenVault: tokenVault, Deserializer: deserializer}
}

// LoadTokens takes an array of token identifiers (txID, index) and returns the keys in the vault
// matching the token identifiers, the corresponding zkatdlog tokens, the information of the
// tokens in clear text and the identities of their owners
// LoadToken returns an error in case of failure
func (s *VaultLedgerTokenAndMetadataLoader[T, M]) LoadTokens(ids []*token.ID) ([]string, []T, []M, []view.Identity, error) {
	var tokens []T
	var inputIDs []string
	var inputInf []M
	var signerIds []view.Identity

	// return token outputs and the corresponding opening
	if err := s.TokenVault.GetTokenInfoAndOutputs(ids, func(id *token.ID, key string, comm, info []byte) error {
		if len(comm) == 0 {
			return errors.Errorf("failed getting state for id [%v], nil comm value", id)
		}
		if len(info) == 0 {
			return errors.Errorf("failed getting state for id [%v], nil info value", id)
		}

		//logger.Debugf("loaded transfer input [%s]", hash.Hashable(comm).String())
		tok, err := s.Deserializer.DeserializeToken(comm)
		if err != nil {
			return errors.Wrapf(err, "failed deserializing token for id [%v][%s]", id, string(comm))
		}
		ti, err := s.Deserializer.DeserializeMetadata(info)
		if err != nil {
			return errors.Wrapf(err, "failed deserializeing token info for id [%v]", id)
		}

		inputIDs = append(inputIDs, key)
		tokens = append(tokens, tok)
		inputInf = append(inputInf, ti)
		signerIds = append(signerIds, tok.GetOwner())

		return nil
	}); err != nil {
		return nil, nil, nil, nil, err
	}

	return inputIDs, tokens, inputInf, signerIds, nil
}

type VaultTokenInfoLoader[M any] struct {
	TokenVault   driver.QueryEngine
	Deserializer MetadataDeserializer[M]
}

func NewVaultTokenInfoLoader[M any](tokenVault driver.QueryEngine, deserializer MetadataDeserializer[M]) *VaultTokenInfoLoader[M] {
	return &VaultTokenInfoLoader[M]{TokenVault: tokenVault, Deserializer: deserializer}
}

func (s *VaultTokenInfoLoader[M]) GetTokenInfos(ids []*token.ID) ([]M, error) {
	var inputInf []M
	if err := s.TokenVault.GetTokenInfos(ids, func(id *token.ID, bytes []byte) error {
		ti, err := s.Deserializer.DeserializeMetadata(bytes)
		if err != nil {
			return errors.Wrapf(err, "failed deserializeing token info for id [%v]", id)
		}
		inputInf = append(inputInf, ti)
		return nil
	}); err != nil {
		return nil, err
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
func (s *VaultTokenLoader) GetTokens(ids []*token.ID) ([]string, []*token.Token, error) {
	return s.TokenVault.GetTokens(ids...)
}

type TokenCertificationStorage interface {
	GetCertifications(ids []*token.ID, callback func(*token.ID, []byte) error) error
}

type VaultTokenCertificationLoader struct {
	TokenCertificationStorage TokenCertificationStorage
}

func (s *VaultTokenCertificationLoader) GetCertifications(ids []*token.ID) ([][]byte, error) {
	var certifications [][]byte
	if err := s.TokenCertificationStorage.GetCertifications(ids, func(id *token.ID, bytes []byte) error {
		if len(bytes) == 0 {
			return errors.Errorf("failed getting certification for id [%v], nil value", id)
		}
		certifications = append(certifications, bytes)
		return nil
	}); err != nil {
		return nil, err
	}
	return certifications, nil
}
