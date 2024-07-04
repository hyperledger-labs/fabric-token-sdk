/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"context"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestValidator_UnmarshalActions(t *testing.T) {
	validator := &Validator{
		backend: &mock.Validator{},
	}

	raw := []byte("some_raw_data")

	expectedActions := []interface{}{"action1", "action2"}
	mockValidator := validator.backend.(*mock.Validator)
	mockValidator.UnmarshalActionsReturns(expectedActions, nil)

	actions, err := validator.UnmarshalActions(raw)

	assert.NoError(t, err)
	assert.Equal(t, expectedActions, actions)
}

func TestValidator_UnmarshallAndVerify(t *testing.T) {
	validator := &Validator{
		backend: &mock.Validator{},
	}
	mockLedger := &mock.ValidatorLedger{}
	raw := []byte("some_raw_data")
	anchor := "some_anchor"

	expectedActions := []interface{}{"action1", "action2"}
	mockValidator := validator.backend.(*mock.Validator)
	mockValidator.VerifyTokenRequestFromRawReturns(expectedActions, nil, nil)

	actions, err := validator.UnmarshallAndVerify(context.TODO(), mockLedger, anchor, raw)

	assert.NoError(t, err)
	assert.Equal(t, expectedActions, actions)
}

func TestValidator_UnmarshallAndVerifyWithMetadata(t *testing.T) {
	validator := &Validator{
		backend: &mock.Validator{},
	}
	mockLedger := &mock.ValidatorLedger{}
	raw := []byte("some_raw_data")
	anchor := "some_anchor"

	expectedActions := []interface{}{"action1", "action2"}
	expectedMetadata := map[string][]byte{"key1": []byte("value1"), "key2": []byte("value2")}
	mockValidator := validator.backend.(*mock.Validator)
	mockValidator.VerifyTokenRequestFromRawReturns(expectedActions, expectedMetadata, nil)
	actions, metadata, err := validator.UnmarshallAndVerifyWithMetadata(context.TODO(), mockLedger, anchor, raw)

	assert.NoError(t, err)
	assert.Equal(t, expectedActions, actions)
	assert.Equal(t, expectedMetadata, metadata)
}

func TestValidator_UnmarshalActions_Error(t *testing.T) {
	validator := &Validator{
		backend: &mock.Validator{},
	}

	raw := []byte("some_raw_data")

	mockValidator := validator.backend.(*mock.Validator)
	mockValidator.UnmarshalActionsReturns(nil, errors.New("mocked error"))

	actions, err := validator.UnmarshalActions(raw)

	assert.Error(t, err)
	assert.Nil(t, actions)
}

func TestValidator_UnmarshallAndVerify_Error(t *testing.T) {
	validator := &Validator{
		backend: &mock.Validator{},
	}
	mockLedger := &mock.ValidatorLedger{}
	raw := []byte("some_raw_data")
	anchor := "some_anchor"

	mockValidator := validator.backend.(*mock.Validator)
	mockValidator.VerifyTokenRequestFromRawReturns(nil, nil, errors.New("mocked error"))
	actions, err := validator.UnmarshallAndVerify(context.TODO(), mockLedger, anchor, raw)

	assert.Error(t, err)
	assert.Nil(t, actions)
}

func TestValidator_UnmarshallAndVerifyWithMetadata_Error(t *testing.T) {
	validator := &Validator{
		backend: &mock.Validator{},
	}
	mockLedger := &mock.ValidatorLedger{}
	raw := []byte("some_raw_data")
	anchor := "some_anchor"

	mockValidator := validator.backend.(*mock.Validator)
	mockValidator.VerifyTokenRequestFromRawReturns(nil, nil, errors.New("mocked error"))
	actions, metadata, err := validator.UnmarshallAndVerifyWithMetadata(context.TODO(), mockLedger, anchor, raw)
	assert.Error(t, err)
	assert.Nil(t, actions)
	assert.Nil(t, metadata)
}
