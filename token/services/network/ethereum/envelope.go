/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ethereum

import (
	"encoding/json"
)

// Envelope is a lightweight placeholder for a future Ethereum transaction envelope.
//
// The scaffold keeps serialization simple so tests and later driver slices have a stable type to
// build on without committing yet to a concrete backend or transaction format.
type Envelope struct {
	Tx string `json:"tx,omitempty"`
}

// Bytes marshals the envelope to bytes.
func (e *Envelope) Bytes() ([]byte, error) {
	return json.Marshal(e)
}

// FromBytes unmarshals the envelope from bytes.
func (e *Envelope) FromBytes(raw []byte) error {
	return json.Unmarshal(raw, e)
}

// TxID returns the ID carried by the scaffold envelope.
func (e *Envelope) TxID() string {
	return e.Tx
}

// String returns a JSON representation of the envelope.
func (e *Envelope) String() string {
	raw, err := e.Bytes()
	if err != nil {
		return "{}"
	}

	return string(raw)
}
