/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package badger

import "encoding/json"

// MarshalTransactionRecord marshals a TransactionRecord into a byte array
func MarshalTransactionRecord(txnRecord *TransactionRecord) ([]byte, error) {
	return json.Marshal(txnRecord)
}

// MarshalMovementRecord marshals a MovementRecord into a byte array
func MarshalMovementRecord(movementRecord *MovementRecord) ([]byte, error) {
	return json.Marshal(movementRecord)
}

// MarshalValidationRecord marshals a ValidationRecord into a byte array
func MarshalValidationRecord(mr *ValidationRecord) ([]byte, error) {
	return json.Marshal(mr)
}

// MarshalTokenRequest marshals a TokenRequest into a byte array
func MarshalTokenRequest(mr *TokenRequest) ([]byte, error) {
	return json.Marshal(mr)
}

// UnmarshalTokenRequest unmarshals a TokenRequest from a byte array
func UnmarshalTokenRequest(data []byte) (*TokenRequest, error) {
	var tokenRequest TokenRequest
	err := json.Unmarshal(data, &tokenRequest)
	if err != nil {
		return nil, err
	}
	return &tokenRequest, nil
}

// UnmarshalTransactionRecord unmarshals a TransactionRecord from a byte array
func UnmarshalTransactionRecord(data []byte) (*TransactionRecord, error) {
	var txnRecord TransactionRecord
	err := json.Unmarshal(data, &txnRecord)
	if err != nil {
		return nil, err
	}
	return &txnRecord, nil
}

// UnmarshalMovementRecord unmarshals a MovementRecord from a byte array
func UnmarshalMovementRecord(data []byte) (*MovementRecord, error) {
	var movementRecord MovementRecord
	err := json.Unmarshal(data, &movementRecord)
	if err != nil {
		return nil, err
	}
	return &movementRecord, nil
}

// UnmarshalValidationRecord unmarshals a ValidationRecord from a byte array
func UnmarshalValidationRecord(data []byte) (*ValidationRecord, error) {
	var record ValidationRecord
	err := json.Unmarshal(data, &record)
	if err != nil {
		return nil, err
	}
	return &record, nil
}
