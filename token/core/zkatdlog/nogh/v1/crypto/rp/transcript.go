/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rp

import (
	"crypto/sha256"

	mathlib "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/asn1"
)

type Transcript struct {
	fsState []byte
	Curve   *mathlib.Curve
}

// Initialize the hasher state
func (tr *Transcript) InitHasher() {
	key := sha256.Sum256([]byte("hello"))
	tr.fsState = key[:]
}

// Absorb the message from transcript
func (tr *Transcript) Absorb(hashBytes []byte) {
	bytesToHash := append(tr.fsState, hashBytes...)
	newHash := sha256.Sum256(bytesToHash)
	tr.fsState = newHash[:]
}

// Squeeze out a challenge
func (tr *Transcript) Squeeze() (*mathlib.Zr, error) {
	raw, err := asn1.MarshalStd(tr.fsState)
	if err != nil {
		return nil, err
	}
	x := tr.Curve.HashToZr(raw)
	tr.Absorb(x.Bytes())
	return x, nil
}
