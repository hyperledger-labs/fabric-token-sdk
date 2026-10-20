/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"crypto/rand"
	"encoding/binary"
	"io"

	"github.com/consensys/gnark-crypto/ecc"
	bls12381fr "github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	bn254fr "github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"

	"github.com/LFDT-Panurus/panurus/x/token/core/zkatsnark/crypto/mimc"
	mimcbn254 "github.com/LFDT-Panurus/panurus/x/token/core/zkatsnark/crypto/mimc/bn254"
)

// CurveField represents the scalar field of any supported SNARK curve.
type CurveField interface {
	bls12381fr.Element | bn254fr.Element
}

// DefaultRandomnessLength is 64 bytes (512 bits), providing at least 256 bits
// of entropy and negligible modular reduction bias (< 2^-256) across all
// supported 254/255-bit elliptic curves.
const DefaultRandomnessLength = 64

// Note is the complete private description of a token, everything needed
// to spend it. It is never published on-chain in cleartext.
//
// Note stores only what must persist for the token's entire lifetime:
// Randomness is the exact value that must be resupplied to open the token's
// commitment whenever it is spent, however far in the future that is.
//
// It deliberately does not store any value-commitment randomness (RCV),
// that is generated fresh every time the token is used in a proof, by the
// WitnessBuilder, and has no relationship to anything stored here.
//
// Randomness is stored as raw bytes to avoid locking Note to a single
// curve's fr.Element type. It is decoded into the appropriate curve's field
// element at usage sites.
type Note struct {
	Value      uint64
	TokenType  string
	Randomness []byte
	Issuer     []byte
}

// FieldElementFromBytes converts raw randomness or element bytes into field element F.
// For 32-byte inputs, it attempts canonical conversion first. For longer byte sequences
// (e.g. 48 or 64 bytes wide randomness) or non-canonical 32-byte sequences, it performs
// modular reduction via SetBytes, ensuring uniform sampling across any curve.
func FieldElementFromBytes[F CurveField](b []byte) (F, error) {
	var f F
	if len(b) == 0 {
		return f, errors.New("note: empty bytes for field element")
	}
	switch el := any(&f).(type) {
	case *bls12381fr.Element:
		if len(b) == 32 {
			if err := el.SetBytesCanonical(b); err == nil {
				return f, nil
			}
		}
		el.SetBytes(b)

		return f, nil
	case *bn254fr.Element:
		if len(b) == 32 {
			if err := el.SetBytesCanonical(b); err == nil {
				return f, nil
			}
		}
		el.SetBytes(b)

		return f, nil
	default:
		return f, errors.Errorf("note: unsupported curve field")
	}
}

// EncodeTokenType maps a token type string to a field element of curve field F.
func EncodeTokenType[F CurveField](tokenType string) F {
	var t F
	switch ptr := any(&t).(type) {
	case *bls12381fr.Element:
		ptr.SetBytes([]byte(tokenType))
	case *bn254fr.Element:
		ptr.SetBytes([]byte(tokenType))
	}

	return t
}

// EncodeTokenTypeBytes maps a token type string to a canonical field element
// encoding as raw bytes for curve field F.
func EncodeTokenTypeBytes[F CurveField](tokenType string) []byte {
	t := EncodeTokenType[F](tokenType)
	switch v := any(t).(type) {
	case bls12381fr.Element:
		b := v.Bytes()

		return b[:]
	case bn254fr.Element:
		b := v.Bytes()

		return b[:]
	default:
		return nil
	}
}

// EncodeTokenTypeBLS12381 maps a token type to a BLS12-381 fr.Element.
func EncodeTokenTypeBLS12381(tokenType string) bls12381fr.Element {
	return EncodeTokenType[bls12381fr.Element](tokenType)
}

// EncodeTokenTypeBN254 maps a token type to a BN254 fr.Element.
func EncodeTokenTypeBN254(tokenType string) bn254fr.Element {
	return EncodeTokenType[bn254fr.Element](tokenType)
}

// SampleRandomness generates cryptographically secure uniform random bytes.
// If byteLen is omitted or non-positive, it defaults to DefaultRandomnessLength (64 bytes).
func SampleRandomness(byteLen ...int) ([]byte, error) {
	n := DefaultRandomnessLength
	if len(byteLen) > 0 && byteLen[0] > 0 {
		n = byteLen[0]
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, errors.Wrapf(err, "note: randomness sampling failed")
	}

	return b, nil
}

// SampleFieldRandomness samples a uniform random element in curve field F.
func SampleFieldRandomness[F CurveField]() (F, []byte, error) {
	var r F
	switch el := any(&r).(type) {
	case *bls12381fr.Element:
		if _, err := el.SetRandom(); err != nil {
			return r, nil, errors.Wrapf(err, "note: randomness generation failed")
		}
		b := el.Bytes()

		return r, b[:], nil
	case *bn254fr.Element:
		if _, err := el.SetRandom(); err != nil {
			return r, nil, errors.Wrapf(err, "note: randomness generation failed")
		}
		b := el.Bytes()

		return r, b[:], nil
	default:
		return r, nil, errors.Errorf("note: unsupported curve field")
	}
}

// NewRandomNote constructs a Note for a newly created token with fresh
// cryptographically random commitment randomness.
// If curve is specified, randomness is sampled in that curve's scalar field;
// otherwise DefaultRandomnessLength (64 bytes) of uniform randomness is used,
// guaranteeing unbiased modular reduction across all supported curves.
func NewRandomNote(value uint64, tokenType string, curve ...ecc.ID) (*Note, error) {
	if len(curve) > 0 {
		switch curve[0] {
		case ecc.BLS12_381:
			return NewRandomNoteForField[bls12381fr.Element](value, tokenType)
		case ecc.BN254:
			return NewRandomNoteForField[bn254fr.Element](value, tokenType)
		default:
			return nil, errors.Errorf("note: unsupported curve %s", curve[0])
		}
	}
	rBytes, err := SampleRandomness(DefaultRandomnessLength)
	if err != nil {
		return nil, err
	}

	return &Note{Value: value, TokenType: tokenType, Randomness: rBytes, Issuer: nil}, nil
}

// NewRandomNoteForField constructs a Note specifically for curve field F,
// using field-level random sampling.
func NewRandomNoteForField[F CurveField](value uint64, tokenType string) (*Note, error) {
	_, rBytes, err := SampleFieldRandomness[F]()
	if err != nil {
		return nil, err
	}

	return &Note{Value: value, TokenType: tokenType, Randomness: rBytes, Issuer: nil}, nil
}

// ComputeTypeCommitment computes a hiding commitment to the token type:
// MiMC(EncodeTokenType(tokenType), typeRandomness). All inputs and outputs
// in a single action share the same typeRandomness, so they produce the
// same TypeCommitment, letting the validator confirm type homogeneity
// without learning the plaintext type.
func ComputeTypeCommitment(tokenType string, typeRandomness []byte, curve ecc.ID) ([]byte, error) {
	switch curve {
	case ecc.BLS12_381:
		t := EncodeTokenType[bls12381fr.Element](tokenType)
		tr, err := FieldElementFromBytes[bls12381fr.Element](typeRandomness)
		if err != nil {
			return nil, errors.Wrapf(err, "note: invalid type randomness bytes")
		}
		tc, err := mimc.Hash(t, tr)
		if err != nil {
			return nil, errors.Wrapf(err, "note: type commitment computation failed")
		}
		b := tc.Bytes()

		return b[:], nil

	case ecc.BN254:
		t := EncodeTokenType[bn254fr.Element](tokenType)
		tr, err := FieldElementFromBytes[bn254fr.Element](typeRandomness)
		if err != nil {
			return nil, errors.Wrapf(err, "note: invalid type randomness bytes")
		}
		tc, err := mimcbn254.Hash(t, tr)
		if err != nil {
			return nil, errors.Wrapf(err, "note: type commitment computation failed")
		}
		b := tc.Bytes()

		return b[:], nil

	default:
		return nil, errors.Errorf("note: unsupported curve %s for type commitment", curve)
	}
}

// ComputeTypeCommitmentForField computes a type commitment for curve field F.
func ComputeTypeCommitmentForField[F CurveField](tokenType string, typeRandomness []byte) ([]byte, error) {
	var zero F
	switch any(zero).(type) {
	case bls12381fr.Element:
		return ComputeTypeCommitment(tokenType, typeRandomness, ecc.BLS12_381)
	case bn254fr.Element:
		return ComputeTypeCommitment(tokenType, typeRandomness, ecc.BN254)
	default:
		return nil, errors.Errorf("note: unsupported curve field")
	}
}

// Commitment computes cm = MiMC(Value, TokenType, Randomness) and returns
// the result as raw bytes.
//
// This must match Constraint Group 1 in circuit.SpendCircuit.Define and
// circuit.OutputCircuit.Define exactly, same three inputs, same order.
//
// The curve parameter selects the MiMC variant (BLS12-381 or BN254).
func (n *Note) Commitment(curve ecc.ID) ([]byte, error) {
	switch curve {
	case ecc.BLS12_381:
		var v bls12381fr.Element
		v.SetUint64(n.Value)
		t := EncodeTokenType[bls12381fr.Element](n.TokenType)
		r, err := FieldElementFromBytes[bls12381fr.Element](n.Randomness)
		if err != nil {
			return nil, errors.Wrapf(err, "note: invalid randomness bytes")
		}
		cm, err := mimc.Hash(v, t, r)
		if err != nil {
			return nil, errors.Wrapf(err, "note: commitment computation failed")
		}
		b := cm.Bytes()

		return b[:], nil

	case ecc.BN254:
		var v bn254fr.Element
		v.SetUint64(n.Value)
		t := EncodeTokenType[bn254fr.Element](n.TokenType)
		r, err := FieldElementFromBytes[bn254fr.Element](n.Randomness)
		if err != nil {
			return nil, errors.Wrapf(err, "note: invalid randomness bytes")
		}
		cm, err := mimcbn254.Hash(v, t, r)
		if err != nil {
			return nil, errors.Wrapf(err, "note: commitment computation failed")
		}
		b := cm.Bytes()

		return b[:], nil

	default:
		return nil, errors.Errorf("note: unsupported curve %s", curve)
	}
}

// Serialize encodes the Note as CLEARTEXT bytes:
// 8-byte big-endian Value,
// 8-byte big-endian TokenType length, then TokenType bytes,
// 8-byte big-endian Randomness length, then Randomness bytes,
// 8-byte big-endian Issuer length, then Issuer bytes.
func (n *Note) Serialize() ([]byte, error) {
	out := make([]byte, 0, 8+8+len(n.TokenType)+8+len(n.Randomness)+8+len(n.Issuer))
	var valueBuf [8]byte
	binary.BigEndian.PutUint64(valueBuf[:], n.Value)
	out = append(out, valueBuf[:]...)

	var typeLenBuf [8]byte
	binary.BigEndian.PutUint64(typeLenBuf[:], uint64(len(n.TokenType)))
	out = append(out, typeLenBuf[:]...)
	out = append(out, []byte(n.TokenType)...)

	var randLenBuf [8]byte
	binary.BigEndian.PutUint64(randLenBuf[:], uint64(len(n.Randomness)))
	out = append(out, randLenBuf[:]...)
	out = append(out, n.Randomness...)

	var issuerLenBuf [8]byte
	binary.BigEndian.PutUint64(issuerLenBuf[:], uint64(len(n.Issuer)))
	out = append(out, issuerLenBuf[:]...)
	out = append(out, n.Issuer...)

	return out, nil
}

// Deserialize reconstructs a Note from bytes produced by Serialize.
// It supports both new length-prefixed randomness notes and legacy 32-byte fixed randomness notes.
func Deserialize(raw []byte) (*Note, error) {
	if len(raw) < 16 {
		return nil, errors.Errorf("note: serialized note too short: %d bytes", len(raw))
	}

	value := binary.BigEndian.Uint64(raw[0:8])
	typeLen := binary.BigEndian.Uint64(raw[8:16])

	if uint64(len(raw)) < 16+typeLen {
		return nil, errors.Errorf("note: length mismatch, expected at least %d got %d", 16+typeLen, len(raw))
	}

	tokenType := string(raw[16 : 16+typeLen])
	rem := raw[16+typeLen:]

	var randomness []byte
	var issuer []byte

	isLengthPrefixed := false
	if len(rem) >= 8 {
		rLen := binary.BigEndian.Uint64(rem[0:8])
		if rLen > 0 && rLen <= 1024 && uint64(len(rem)) >= 8+rLen {
			remAfterRand := rem[8+rLen:]
			if len(remAfterRand) == 0 {
				isLengthPrefixed = true
			} else if len(remAfterRand) >= 8 {
				issLen := binary.BigEndian.Uint64(remAfterRand[0:8])
				if uint64(len(remAfterRand)) == 8+issLen {
					isLengthPrefixed = true
				}
			}
		}
	}

	if isLengthPrefixed {
		rLen := binary.BigEndian.Uint64(rem[0:8])
		randomness = make([]byte, rLen)
		copy(randomness, rem[8:8+rLen])
		remAfterRand := rem[8+rLen:]
		if len(remAfterRand) >= 8 {
			issLen := binary.BigEndian.Uint64(remAfterRand[0:8])
			issuer = make([]byte, issLen)
			copy(issuer, remAfterRand[8:8+issLen])
		}
	} else {
		if len(rem) < 32 {
			return nil, errors.Errorf("note: length mismatch, expected at least 32 randomness bytes, got %d", len(rem))
		}
		randomness = make([]byte, 32)
		copy(randomness, rem[:32])
		remAfterRand := rem[32:]
		if len(remAfterRand) >= 8 {
			issLen := binary.BigEndian.Uint64(remAfterRand[0:8])
			if uint64(len(remAfterRand)) != 8+issLen {
				return nil, errors.Errorf("note: length mismatch, expected %d issuer bytes got %d", 8+issLen, len(remAfterRand))
			}
			issuer = make([]byte, issLen)
			copy(issuer, remAfterRand[8:8+issLen])
		} else if len(remAfterRand) != 0 {
			return nil, errors.Errorf("note: invalid trailing bytes: %d", len(remAfterRand))
		}
	}

	return &Note{Value: value, TokenType: tokenType, Randomness: randomness, Issuer: issuer}, nil
}
