/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package circuits

import (
	tedwards "github.com/consensys/gnark-crypto/ecc/twistededwards"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/algebra/native/twistededwards"
)

// IssueCircuit is a Groth16 circuit that proves a single token issuance is valid.
//
// For the POC this handles exactly one output token. The full driver will generate
// separate circuit compilations for each supported output count (1, 2, ...).
//
// The circuit enforces three properties:
//  1. Issuer key ownership: IssuerPrivKey * G == (IssuerPubKeyX, IssuerPubKeyY)
//  2. Commitment validity: Value * G + BlindingFactor * H == (CommitmentX, CommitmentY)
//  3. Range bound: 0 <= Value <= MaxValue
type IssueCircuit struct {
	// ── Public inputs (committed to in the proof, visible on-chain) ──────────

	// IssuerPubKeyX and IssuerPubKeyY are the Baby Jubjub coordinates of the
	// issuer's registered public key.
	IssuerPubKeyX frontend.Variable `gnark:",public"`
	IssuerPubKeyY frontend.Variable `gnark:",public"`

	// CommitmentX and CommitmentY are the coordinates of the issued token's
	// Pedersen commitment on Baby Jubjub. This is the on-chain token representation.
	CommitmentX frontend.Variable `gnark:",public"`
	CommitmentY frontend.Variable `gnark:",public"`

	// HX and HY are the coordinates of the second Pedersen generator H,
	// derived from the driver's public parameters.
	HX frontend.Variable `gnark:",public"`
	HY frontend.Variable `gnark:",public"`

	// MaxValue is the maximum allowed token value, set from the public parameters.
	MaxValue frontend.Variable `gnark:",public"`

	// ── Private witness (known only to the prover) ────────────────────────────

	// IssuerPrivKey is the issuer's secret scalar on Baby Jubjub.
	IssuerPrivKey frontend.Variable

	// Value is the plaintext token value being issued.
	Value frontend.Variable

	// BlindingFactor is the commitment randomness that hides the value.
	BlindingFactor frontend.Variable
}

// Define specifies the constraint system for the IssueCircuit.
// gnark calls this method when compiling the circuit into an R1CS.
func (c *IssueCircuit) Define(api frontend.API) error {
	curve, err := twistededwards.NewEdCurve(api, tedwards.BN254)
	if err != nil {
		return err
	}

	params := curve.Params()
	G := twistededwards.Point{X: params.Base[0], Y: params.Base[1]}

	// 1. Issuer key ownership: IssuerPrivKey * G must equal the registered public key.
	derivedPK := curve.ScalarMul(G, c.IssuerPrivKey)
	api.AssertIsEqual(derivedPK.X, c.IssuerPubKeyX)
	api.AssertIsEqual(derivedPK.Y, c.IssuerPubKeyY)

	// 2. Commitment validity: the token commitment must open to (Value, BlindingFactor).
	com := PedersenCommitment{
		CX:             c.CommitmentX,
		CY:             c.CommitmentY,
		Value:          c.Value,
		BlindingFactor: c.BlindingFactor,
		HX:             c.HX,
		HY:             c.HY,
	}
	if err := com.verify(api); err != nil {
		return err
	}

	// 3. Range bound: value must not exceed the maximum allowed by the public params.
	api.AssertIsLessOrEqual(c.Value, c.MaxValue)

	return nil
}
