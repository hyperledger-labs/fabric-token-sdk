/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package prover

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"hash"
	"math/big"
	"sort"

	bls12381tw "github.com/consensys/gnark-crypto/ecc/bls12-381/twistededwards"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"

	"github.com/LFDT-Panurus/panurus/x/token/core/zkatsnark/crypto/edwards"
)

// domainSeparator prevents a binding signature produced by this scheme from
// being valid in, or confused with, any other protocol context.
const domainSeparator = "zkatsnark-action-binding-v1"

// ProofResult holds everything one input or output token contributes to an
// action's binding signature. All fields are raw bytes to avoid locking
// the type to a single curve.
type ProofResult struct {
	Commitment       []byte // 32 bytes
	ValueCommitmentX []byte // 32 bytes
	ValueCommitmentY []byte // 32 bytes
	RCV              []byte // 32 bytes
}

// sortedByCommitment returns results sorted lexicographically by
// Commitment bytes. This makes ComputeActionHash canonical regardless of
// which order the Orchestrator's goroutines happen to complete in — two
// runs proving the same logical action, in any goroutine completion order,
// must produce the same hash.
func sortedByCommitment(results []ProofResult) []ProofResult {
	sorted := make([]ProofResult, len(results))
	copy(sorted, results)
	sort.Slice(sorted, func(i, j int) bool {
		return bytes.Compare(sorted[i].Commitment, sorted[j].Commitment) < 0
	})

	return sorted
}

// writeLengthPrefixed writes an 8-byte big-endian length followed by the
// bytes. Used for actionType and tokenType, which are variable-length
// strings — without a length prefix, concatenating two variable-length
// strings back-to-back is ambiguous (e.g. "a"+"bc" and "ab"+"c" would hash
// identically under naive concatenation). Fixed-length fields (commitments,
// coordinates) don't need this.
func writeLengthPrefixed(h hash.Hash, b []byte) {
	var lenBuf [8]byte
	binary.BigEndian.PutUint64(lenBuf[:], uint64(len(b)))
	_, _ = h.Write(lenBuf[:])
	_, _ = h.Write(b)
}

// ComputeActionHash computes the canonical, deterministic message the
// binding signature is produced over. Must be computed byte-identically by
// the prover (when signing) and eventually the validator (when verifying) —
// any divergence in sorting, encoding, or field ordering causes every
// legitimate binding signature to fail verification with no useful
// diagnostic.
func ComputeActionHash(actionType string, typeCommitment []byte, inputs, outputs []ProofResult) []byte {
	h := sha256.New()
	_, _ = h.Write([]byte(domainSeparator))
	writeLengthPrefixed(h, []byte(actionType))

	// TypeCommitment is a fixed-length field element (32 bytes), so no
	// length prefix is needed — there is no concatenation ambiguity.
	_, _ = h.Write(typeCommitment)

	for _, r := range sortedByCommitment(inputs) {
		_, _ = h.Write(r.Commitment)
		_, _ = h.Write(r.ValueCommitmentX)
		_, _ = h.Write(r.ValueCommitmentY)
	}

	for _, r := range sortedByCommitment(outputs) {
		_, _ = h.Write(r.Commitment)
		_, _ = h.Write(r.ValueCommitmentX)
		_, _ = h.Write(r.ValueCommitmentY)
	}

	return h.Sum(nil)
}

// ComputeBVK computes the binding verification key:
//
//	bvk = Σcv_in − Σcv_out + valueBalance·V
//
// valueBalance corrects for non-conserved value flows, following the
// ZCash Sapling binding signature convention:
//
//   - Transfer (conserved values): valueBalance = 0
//   - Issuance (no inputs):        valueBalance = Σv_out
//
// When values are conserved the V-terms in Σcv_in and Σcv_out cancel,
// leaving bvk = bsk·R. For issuance the V-terms do NOT cancel;
// adding valueBalance·V restores the equality bvk = bsk·R so the
// Schnorr verification succeeds.
//
// ComputeBVK works over BLS12-381/Jubjub; use ComputeBVKFor for another curve.
func ComputeBVK(inputs, outputs []ProofResult, valueBalance uint64) (bls12381tw.PointAffine, error) {
	return ComputeBVKFor(edwards.Jubjub(), inputs, outputs, valueBalance)
}

// ComputeBVKFor is ComputeBVK over the given twisted Edwards curve.
func ComputeBVKFor[F any, PF edwards.Field[F], P any](
	c *edwards.Curve[F, PF, P], inputs, outputs []ProofResult, valueBalance uint64,
) (P, error) {
	var zero P
	points := make([]P, 0, len(inputs)+len(outputs)+1)

	for _, r := range inputs {
		pt, err := c.PointFromBytes(r.ValueCommitmentX, r.ValueCommitmentY)
		if err != nil {
			return zero, errors.Wrapf(err, "prover: invalid input value commitment")
		}
		points = append(points, pt)
	}

	for _, r := range outputs {
		pt, err := c.PointFromBytes(r.ValueCommitmentX, r.ValueCommitmentY)
		if err != nil {
			return zero, errors.Wrapf(err, "prover: invalid output value commitment")
		}
		points = append(points, c.NegatePoint(pt))
	}

	if valueBalance != 0 {
		points = append(points, c.ScalarMul(c.V, new(big.Int).SetUint64(valueBalance)))
	}

	return c.ValueCommitSum(points), nil
}

// ComputeBindingSignature computes the per-action binding signature proving
// value conservation: Σvalue_in == Σvalue_out, over BLS12-381/Jubjub.
// See ComputeBindingSignatureFor.
func ComputeBindingSignature(actionType string, typeCommitment []byte, inputs, outputs []ProofResult) (edwards.JubjubSignature, error) {
	return ComputeBindingSignatureFor(edwards.Jubjub(), actionType, typeCommitment, inputs, outputs)
}

// ComputeBindingSignatureFor computes the per-action binding signature over
// the given twisted Edwards curve.
//
// bsk = Σrcv_in − Σrcv_out is the private signing key, computable only
// because the caller collected every RCV used across this action's proofs.
// Signing uses the curve's R generator as the base point, matching the ZCash
// Sapling convention: bvk = Σcv_in − Σcv_out reduces to exactly bsk·R when
// values are conserved, and to something else entirely (unreachable by the
// prover, since nobody knows log_R(V)) when they are not.
//
// For issuance (empty inputs), bsk becomes −Σrcv_out. There is no
// conservation requirement for issuance, the signature instead attests
// that the published output value commitments are correctly formed.
func ComputeBindingSignatureFor[F any, PF edwards.Field[F], P any](
	c *edwards.Curve[F, PF, P], actionType string, typeCommitment []byte, inputs, outputs []ProofResult,
) (edwards.Signature[F, P], error) {
	var none edwards.Signature[F, P]

	// bsk must be computed modulo the embedded curve's subgroup order, NOT the
	// host scalar field modulus: field arithmetic reduces mod the latter, but
	// the Schnorr scheme operates in the former.
	bskBig := new(big.Int)

	for _, r := range inputs {
		rcv, err := c.ScalarFromBytes(r.RCV)
		if err != nil {
			return none, errors.Wrapf(err, "prover: invalid input RCV bytes")
		}
		bskBig.Add(bskBig, PF(&rcv).BigInt(new(big.Int)))
	}

	for _, r := range outputs {
		rcv, err := c.ScalarFromBytes(r.RCV)
		if err != nil {
			return none, errors.Wrapf(err, "prover: invalid output RCV bytes")
		}
		bskBig.Sub(bskBig, PF(&rcv).BigInt(new(big.Int)))
	}

	bskBig.Mod(bskBig, c.Order())

	var bsk F
	PF(&bsk).SetBigInt(bskBig)

	actionHash := ComputeActionHash(actionType, typeCommitment, inputs, outputs)

	sig, err := c.Sign(bsk, actionHash, c.R)
	if err != nil {
		return none, errors.Wrapf(err, "prover: binding signature generation failed")
	}

	return sig, nil
}
