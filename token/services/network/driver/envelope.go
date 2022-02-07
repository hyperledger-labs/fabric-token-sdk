/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

// Envelope models a network envelope
type Envelope interface {
	// Results returns the results
	Results() []byte

	// Bytes marshals the envelope to bytes
	Bytes() ([]byte, error)

	FromBytes([]byte) error

	// TxID returns the ID of this envelope
	TxID() string

	// Nonce returns the nonce, if any
	Nonce() []byte

	// Creator returns the creator of this envelope
	Creator() []byte
}
