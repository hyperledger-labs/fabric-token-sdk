/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"crypto/sha256"
	"encoding/base64"
	"hash"
)

//go:generate counterfeiter -o hash_mock.go -fake-name HashMock hash.Hash

type Hashable []byte

// hashFunc is a variable that can be overridden in tests
var hashFunc = func() hash.Hash {
	return sha256.New()
}

func (id Hashable) Raw() []byte {
	if len(id) == 0 {
		return nil
	}
	hash := hashFunc()
	n, err := hash.Write(id)
	if n != len(id) {
		panic("hash failure")
	}
	if err != nil {
		panic(err)
	}

	return hash.Sum(nil)
}

func (id Hashable) String() string { return base64.StdEncoding.EncodeToString(id.Raw()) }

func (id Hashable) RawString() string { return string(id.Raw()) }
