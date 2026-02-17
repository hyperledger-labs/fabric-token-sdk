/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"hash"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
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

type Hasher struct {
	h hash.Hash
}

func NewSHA256Hasher() *Hasher {
	return &Hasher{h: sha256.New()}
}

func (h *Hasher) AddInt32(i int32) error {
	return binary.Write(h.h, binary.LittleEndian, i)
}

func (h *Hasher) AddInt(i int) error {
	return binary.Write(h.h, binary.LittleEndian, int64(i))
}

func (h *Hasher) AddUInt64(i uint64) error {
	return binary.Write(h.h, binary.LittleEndian, i)
}

func (h *Hasher) AddBytes(b []byte) error {
	_, err := h.h.Write(b)

	return err
}

func (h *Hasher) AddString(s string) error {
	_, err := h.h.Write([]byte(s))

	return err
}

func (h *Hasher) AddBool(b bool) (int, error) {
	if b {
		return h.h.Write([]byte{1})
	}

	return h.h.Write([]byte{0})
}

func (h *Hasher) AddFloat64(f float64) error {
	return binary.Write(h.h, binary.LittleEndian, f)
}

func (h *Hasher) Digest() []byte {
	return h.h.Sum(nil)
}

func (h *Hasher) HexDigest() string {
	return hex.EncodeToString(h.h.Sum(nil))
}

func (h *Hasher) AddG1s(generators []*math.G1) error {
	for _, g := range generators {
		if err := h.AddBytes(g.Bytes()); err != nil {
			return errors.WithMessagef(err, "failed to add g1 element")
		}
	}

	return nil
}
