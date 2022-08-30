/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

type Backend struct {
	// Ledger to access the ledger state
	Ledger driver.GetStateFnc
	// signed Message
	Message []byte
	// Cursor is used to iterate over the signatures
	Cursor int
	// signatures on Message
	Sigs [][]byte
}

func NewBackend(ledger driver.GetStateFnc, message []byte, sigs [][]byte) *Backend {
	return &Backend{Ledger: ledger, Message: message, Sigs: sigs}
}

// HasBeenSignedBy checks if a given Message has been signed by the signing identity matching
// the passed verifier
func (b *Backend) HasBeenSignedBy(id view.Identity, verifier driver.Verifier) ([]byte, error) {
	if b.Cursor >= len(b.Sigs) {
		return nil, errors.New("invalid state, insufficient number of signatures")
	}
	sigma := b.Sigs[b.Cursor]
	b.Cursor++

	return sigma, verifier.Verify(b.Message, sigma)
}

func (b *Backend) GetState(key string) ([]byte, error) {
	return b.Ledger(key)
}

func (b *Backend) Signatures() [][]byte {
	return b.Sigs
}
