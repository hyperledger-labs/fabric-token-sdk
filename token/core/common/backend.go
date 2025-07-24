/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/utils/logging"
	"github.com/pkg/errors"
)

type Backend struct {
	Logger logging.Logger
	// Ledger to access the ledger state
	Ledger driver.GetStateFnc
	// signed Message
	Message []byte
	// Cursor is used to iterate over the signatures
	Cursor int
	// signatures on Message
	Sigs [][]byte
}

func NewBackend(logger logging.Logger, ledger driver.GetStateFnc, message []byte, sigs [][]byte) *Backend {
	return &Backend{Logger: logger, Ledger: ledger, Message: message, Sigs: sigs}
}

// HasBeenSignedBy checks if a given Message has been signed by the signing identity matching
// the passed verifier
func (b *Backend) HasBeenSignedBy(id driver.Identity, verifier driver.Verifier) ([]byte, error) {
	if b.Cursor >= len(b.Sigs) {
		return nil, errors.New("invalid state, insufficient number of signatures")
	}
	sigma := b.Sigs[b.Cursor]
	b.Cursor++

	b.Logger.Infof("verify signature [%s][%s][%s]", id, logging.Base64(sigma), utils.Hashable(b.Message))

	return sigma, verifier.Verify(b.Message, sigma)
}

func (b *Backend) GetState(id token.ID) ([]byte, error) {
	return b.Ledger(id)
}

func (b *Backend) Signatures() [][]byte {
	return b.Sigs
}
