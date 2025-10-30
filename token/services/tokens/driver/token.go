/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"bytes"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
)

type (
	Type = int32
)

// Token wraps the byte representation of a lower level token.
type Token []byte

// Equal return true if the identities are the same
func (id Token) Equal(id2 Token) bool {
	return bytes.Equal(id, id2)
}

// UniqueID returns a unique identifier of this token
func (id Token) UniqueID() string {
	if len(id) == 0 {
		return "<empty>"
	}
	return utils.Hashable(id).String()
}

// Hash returns the hash of this token
func (id Token) Hash() string {
	if len(id) == 0 {
		return "<empty>"
	}
	return utils.Hashable(id).RawString()
}

// String returns a string representation of this token
func (id Token) String() string {
	return id.UniqueID()
}

// Bytes returns the byte representation of this token
func (id Token) Bytes() []byte {
	return id
}

// IsNone returns true if this token is empty
func (id Token) IsNone() bool {
	return len(id) == 0
}

// Metadata wraps the byte representation of a lower level metadata.
type Metadata []byte

// Equal return true if the identities are the same
func (id Metadata) Equal(id2 Metadata) bool {
	return bytes.Equal(id, id2)
}

// UniqueID returns a unique identifier of this metadata
func (id Metadata) UniqueID() string {
	if len(id) == 0 {
		return "<empty>"
	}
	return utils.Hashable(id).String()
}

// Hash returns the hash of this metadata
func (id Metadata) Hash() string {
	if len(id) == 0 {
		return "<empty>"
	}
	return utils.Hashable(id).RawString()
}

// String returns a string representation of this metadata
func (id Metadata) String() string {
	return id.UniqueID()
}

// Bytes returns the byte representation of this metadata
func (id Metadata) Bytes() []byte {
	return id
}

// IsNone returns true if this metadata is empty
func (id Metadata) IsNone() bool {
	return len(id) == 0
}
