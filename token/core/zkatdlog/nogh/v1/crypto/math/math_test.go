/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package math

import (
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/stretchr/testify/require"
)

func TestCheckElement(t *testing.T) {
	var g1 *math.G1
	err := CheckElement(g1, math.BN254)
	require.Error(t, err)
	require.Contains(t, err.Error(), "elememt is nil")

	g1 = &math.G1{}
	err = CheckElement(g1, math.BN254)
	require.Error(t, err)
	// mathlib G1{} might panic on CurveID() or IsInfinity() if not initialized
	// The CheckElement has a recover block

	curve := math.Curves[math.BN254]
	g1 = curve.GenG1
	err = CheckElement(g1, math.BN254)
	require.NoError(t, err)

	err = CheckElement(g1, math.BLS12_381_BBS)
	require.Error(t, err)
	require.Contains(t, err.Error(), "element curve must equal curve ID")

	inf := curve.NewG1()
	err = CheckElement(inf, math.BN254)
	require.Error(t, err)
	require.Contains(t, err.Error(), "element is infinity")
}

func TestCheckBaseElement(t *testing.T) {
	var zr *math.Zr
	err := CheckBaseElement(zr, math.BN254)
	require.Error(t, err)
	require.Contains(t, err.Error(), "elememt is nil")

	curve := math.Curves[math.BN254]
	zr = curve.NewZrFromUint64(1)
	err = CheckBaseElement(zr, math.BN254)
	require.NoError(t, err)

	err = CheckBaseElement(zr, math.BLS12_381_BBS)
	require.Error(t, err)
	require.Contains(t, err.Error(), "element curve must equal curve ID")
}

func TestCheckElements(t *testing.T) {
	curve := math.Curves[math.BN254]
	g1 := curve.GenG1
	g2 := curve.GenG1

	err := CheckElements([]*math.G1{g1, g2}, math.BN254, 2)
	require.NoError(t, err)

	err = CheckElements([]*math.G1{g1, g2}, math.BN254, 3)
	require.Error(t, err)
	require.Contains(t, err.Error(), "length of elements does not match length of curveID")

	err = CheckElements([]*math.G1{g1, nil}, math.BN254, 2)
	require.Error(t, err)
	require.Contains(t, err.Error(), "elememt is nil")
}

func TestCheckZrElements(t *testing.T) {
	curve := math.Curves[math.BN254]
	zr1 := curve.NewZrFromUint64(1)
	zr2 := curve.NewZrFromUint64(2)

	err := CheckZrElements([]*math.Zr{zr1, zr2}, math.BN254, 2)
	require.NoError(t, err)

	err = CheckZrElements([]*math.Zr{zr1, zr2}, math.BN254, 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "length of elements does not match length of curveID")

	err = CheckZrElements([]*math.Zr{zr1, nil}, math.BN254, 2)
	require.Error(t, err)
	require.Contains(t, err.Error(), "elememt is nil")
}

func TestIsNilInterface(t *testing.T) {
	require.True(t, isNilInterface(nil))
	var g1 *math.G1
	require.True(t, isNilInterface(g1))
	require.False(t, isNilInterface(math.Curves[math.BN254].GenG1))
	require.False(t, isNilInterface(10))
}
