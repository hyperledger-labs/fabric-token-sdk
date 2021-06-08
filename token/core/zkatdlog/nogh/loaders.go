/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package nogh

import (
	"github.com/pkg/errors"

	api2 "github.com/hyperledger-labs/fabric-token-sdk/token/api"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TokenVault interface {
	PublicParams() ([]byte, error)
	GetTokenInfos(ids []*token3.Id, callback api2.QueryCallbackFunc) error
	GetTokenCommitments(ids []*token3.Id, callback api2.QueryCallbackFunc) error
}

type VaultTokenCommitmentLoader struct {
	TokenVault TokenVault
}

func (s *VaultTokenCommitmentLoader) GetTokenCommitments(ids []*token3.Id) ([]*token.Token, error) {
	var tokens []*token.Token
	if err := s.TokenVault.GetTokenCommitments(ids, func(id *token3.Id, bytes []byte) error {
		if len(bytes) == 0 {
			return errors.Errorf("failed getting state for id [%v], nil value", id)
		}
		ti := &token.Token{}
		if err := ti.Deserialize(bytes); err != nil {
			return errors.Wrapf(err, "failed deserializeing token for id [%v][%s]", id, string(bytes))
		}
		tokens = append(tokens, ti)
		return nil
	}); err != nil {
		return nil, err
	}
	return tokens, nil
}
