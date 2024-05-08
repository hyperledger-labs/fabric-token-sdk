/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

// Envelope models a network envelope
type Envelope interface {
	// Bytes marshals the envelope to bytes
	Bytes() ([]byte, error)

	// FromBytes unmarshals the envelope from bytes
	FromBytes([]byte) error

	// TxID returns the ID of this envelope
	TxID() string

	// String returns the string representation of this envelope
	String() string
}
