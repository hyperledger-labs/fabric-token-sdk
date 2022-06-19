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
