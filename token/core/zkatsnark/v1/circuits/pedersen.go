/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package circuits contains gnark circuit definitions for the zkatsnark token driver.
// It implements Pedersen commitments on Baby Jubjub (the twisted Edwards curve
// embedded in BN254) proved using Groth16 SNARKs.
package circuits

import (
	tedwards "github.com/consensys/gnark-crypto/ecc/twistededwards"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/algebra/native/twistededwards"
)

// PedersenCommitment is a reusable circuit gadget that proves knowledge of
// a valid opening (Value, BlindingFactor) of a Pedersen commitment on Baby Jubjub.
//
// The commitment equation is:
//
//	C = Value * G + BlindingFactor * H
//
// where G is the Baby Jubjub base point (embedded as a constant from the curve
// parameters) and H is a second independent generator passed as a public input.
type PedersenCommitment struct {
	// CX and CY are the public commitment coordinates.
	CX frontend.Variable
	CY frontend.Variable

	// Value is the secret token value being committed to.
	Value frontend.Variable

	// BlindingFactor is the secret randomness that hides the value.
	BlindingFactor frontend.Variable

	// HX and HY are the coordinates of the second Pedersen generator H.
	// These are public inputs set from the driver's public parameters.
	HX frontend.Variable
	HY frontend.Variable
}

// verify enforces the Pedersen commitment equation inside the circuit.
// It must be called from within a parent circuit's Define method.
func (c *PedersenCommitment) verify(api frontend.API) error {
	curve, err := twistededwards.NewEdCurve(api, tedwards.BN254)
	if err != nil {
		return err
	}

	// G is the Baby Jubjub base point, a circuit constant derived from the curve params.
	params := curve.Params()
	G := twistededwards.Point{X: params.Base[0], Y: params.Base[1]}

	// H is the second generator, a public input from the driver's public parameters.
	H := twistededwards.Point{X: c.HX, Y: c.HY}

	// Compute Value * G + BlindingFactor * H and assert it equals the commitment.
	vG := curve.ScalarMul(G, c.Value)
	bfH := curve.ScalarMul(H, c.BlindingFactor)
	computed := curve.Add(vG, bfH)

	api.AssertIsEqual(computed.X, c.CX)
	api.AssertIsEqual(computed.Y, c.CY)

	return nil
}
