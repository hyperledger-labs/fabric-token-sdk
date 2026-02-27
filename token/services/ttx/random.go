/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"crypto/rand"
	"io"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

const (
	// NonceSize is the default NonceSize
	NonceSize = 24
)

// GetRandomBytes returns length random bytes, guaranteeing the buffer is fully filled
func GetRandomBytes(length int) ([]byte, error) {
	key := make([]byte, length)

	// Ensure the buffer is completely filled
	_, err := io.ReadFull(rand.Reader, key)
	if err != nil {
		return nil, errors.Wrap(err, "error getting random bytes")
	}

	return key, nil
}

// GetRandomNonce returns a random byte array of length NonceSize
func GetRandomNonce() ([]byte, error) {
	return GetRandomBytes(NonceSize)
}