/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"crypto/sha256"
	"encoding/base64"
)

type Hashable []byte

func (id Hashable) Raw() []byte {
	if len(id) == 0 {
		return nil
	}
	hash := sha256.New()
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
