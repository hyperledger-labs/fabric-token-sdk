/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rp

import (
	"math/big"

	mathlib "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

// toBits returns the n-bit little-endian representation of v as field elements,
// where bits[0] is the LSB and bits[n-1] is the MSB. Returns an error if v >= 2^n.
func toBits(v *mathlib.Zr, n uint64, curve *mathlib.Curve) ([]*mathlib.Zr, error) {
	val := new(big.Int).SetBytes(v.Bytes())

	limit := new(big.Int).Lsh(big.NewInt(1), uint(n))
	if val.Cmp(limit) >= 0 {
		return nil, errors.Errorf("value %s does not fit in %d bits", val, n)
	}

	bits := make([]*mathlib.Zr, n)
	for i := uint64(0); i < n; i++ {
		bits[i] = curve.NewZrFromInt(int64(val.Bit(int(i))))
	}
	return bits, nil
}

// fieldDiffInt returns (a - b) mod GroupOrder as a field element.
// Handles negative differences by negating the absolute value.
func fieldDiffInt(a, b int, curve *mathlib.Curve) *mathlib.Zr {
	if a >= b {
		return curve.NewZrFromInt(int64(a - b))
	}
	diff := curve.NewZrFromInt(int64(b - a))
	diff.Neg()
	return diff
}

// getLagrangeMultipliers returns Lagrange coefficients [x_0, x_1, ..., x_n] such that
// for any degree-n polynomial p(X):
//
//	p(c) = x_0·p(0) + x_1·p(1) + ... + x_n·p(n)
//
// The evaluation points are the integers {0, 1, ..., n}. The i-th coefficient is
// the i-th Lagrange basis polynomial evaluated at c:
//
//	x_i = ∏_{j=0, j≠i}^{n} (c-j) / (i-j)

// Note that this is O(n^2) algorithm but field operations are significantly faster.
// We can revisit it if it ever becomes a bottle-neck to have FFT based implementation.
func getLagrangeMultipliers(n uint64, c *mathlib.Zr, Curve *mathlib.Curve) ([]*mathlib.Zr, error) {
	// For BN254 and BLS12 curves, we perform arithmetic over gnark crypto native type
	// instead of mathlib wrapper, which uses slower big.Int conversion.
	if r, ok, err := nativeLagrangeMultipliers(n, c, Curve); ok {
		return r, err
	}
	m := int(n) + 1 // number of evaluation points: 0, 1, ..., n

	// Precompute (c - j) for j = 0..n.
	cMinusJ := make([]*mathlib.Zr, m)
	for j := 0; j < m; j++ {
		neg := Curve.NewZrFromInt(int64(j))
		neg.Neg()
		cMinusJ[j] = Curve.ModAdd(c, neg, Curve.GroupOrder)
	}

	numers := make([]*mathlib.Zr, m)
	denoms := make([]*mathlib.Zr, m)
	for i := 0; i < m; i++ {
		numer := Curve.NewZrFromInt(1)
		denom := Curve.NewZrFromInt(1)
		for j := 0; j < m; j++ {
			if j == i {
				continue
			}
			numer = Curve.ModMul(numer, cMinusJ[j], Curve.GroupOrder)
			denom = Curve.ModMul(denom, fieldDiffInt(i, j, Curve), Curve.GroupOrder)
		}
		numers[i] = numer
		denoms[i] = denom
	}

	denomInvs := BatchInverse(denoms, Curve)
	result := make([]*mathlib.Zr, m)
	for i := 0; i < m; i++ {
		result[i] = Curve.ModMul(numers[i], denomInvs[i], Curve.GroupOrder)
	}
	return result, nil
}

// getLagrangeMultipliersPartial returns coefficients [x_0, x_{n+1}, ..., x_{2n}] such that
// for a degree-2n polynomial p(X) satisfying p(1) = p(2) = ... = p(n) = 0:
//
//	p(c) = x_0·p(0) + x_{n+1}·p(n+1) + ... + x_{2n}·p(2n)
//
// The multipliers are the standard Lagrange basis values over the full set of 2n+1
// evaluation points {0, 1, ..., 2n}; only those for {0, n+1, ..., 2n} are returned
// because the remaining evaluations at {1, ..., n} are assumed to be zero.
func getLagrangeMultipliersPartial(n uint64, c *mathlib.Zr, Curve *mathlib.Curve) ([]*mathlib.Zr, error) {
	// Use native gnark-crypto FrElement for faster field arithmetic.
	if r, ok, err := nativeLagrangeMultipliersPartial(n, c, Curve); ok {
		return r, err
	}
	total := 2*int(n) + 1 // all evaluation points: 0, 1, ..., 2n

	// Precompute (c - j) for j = 0..2n.
	cMinusJ := make([]*mathlib.Zr, total)
	for j := 0; j < total; j++ {
		neg := Curve.NewZrFromInt(int64(j))
		neg.Neg()
		cMinusJ[j] = Curve.ModAdd(c, neg, Curve.GroupOrder)
	}

	// Indices of interest: {0, n+1, n+2, ..., 2n} — n+1 values in total.
	relevant := make([]int, int(n)+1)
	relevant[0] = 0
	for k := 1; k <= int(n); k++ {
		relevant[k] = int(n) + k
	}

	numers := make([]*mathlib.Zr, len(relevant))
	denoms := make([]*mathlib.Zr, len(relevant))
	for k, i := range relevant {
		numer := Curve.NewZrFromInt(1)
		denom := Curve.NewZrFromInt(1)
		for j := 0; j < total; j++ {
			if j == i {
				continue
			}
			numer = Curve.ModMul(numer, cMinusJ[j], Curve.GroupOrder)
			denom = Curve.ModMul(denom, fieldDiffInt(i, j, Curve), Curve.GroupOrder)
		}
		numers[k] = numer
		denoms[k] = denom
	}

	denomInvs := BatchInverse(denoms, Curve)
	result := make([]*mathlib.Zr, len(relevant))
	for k := range relevant {
		result[k] = Curve.ModMul(numers[k], denomInvs[k], Curve.GroupOrder)
	}
	return result, nil
}

// interpolate takes the n+1 values of a degree-n polynomial at {0, 1, ..., n}
// and returns the 2n+1 values at {0, 1, ..., 2n} by Lagrange extension.
//
// The n+1 input values are copied directly; the n new values at {n+1, ..., 2n}
// are computed as p(x) = Σ_{i=0}^{n} L_i(x)·p(i).
//
// We achieve O(n^2) efficiency as follows:
// - The n+1 denominator factors d_i = ∏_{j≠i}(i-j) are computed upfront in O(n^2) time.
// - For each x \in [n+1,2n] numerator P(x) = ∏_j(x-j) takes O(n) time
// - For each x \in [n+1,2n] the 1/(x-j) for all j\in [0,n] takes O(n) time, where we use batch inversion for efficiency.
// - L_i(x) = P(x)/(d_i (x-i))
func interpolate(n uint64, valuesOverN []*mathlib.Zr, curve *mathlib.Curve) ([]*mathlib.Zr, error) {
	if r, ok, err := nativeInterpolate(n, valuesOverN, curve); ok {
		return r, err
	}
	m := int(n) + 1 // number of known points (indices 0..n)

	// Step 1: precompute denominator d_i = ∏_{j≠i}(i-j) for i=0..n. O(n²).
	denoms := make([]*mathlib.Zr, m)
	for i := 0; i < m; i++ {
		d := curve.NewZrFromInt(1)
		for j := 0; j < m; j++ {
			if j == i {
				continue
			}
			d = curve.ModMul(d, fieldDiffInt(i, j, curve), curve.GroupOrder)
		}
		denoms[i] = d
	}
	denomInvs := BatchInverse(denoms, curve)

	// Result: first n+1 entries are the inputs, followed by n new values.
	result := make([]*mathlib.Zr, 2*int(n)+1)
	copy(result, valuesOverN)

	// Step 2: for each x in {n+1, ..., 2n} evaluate p(x). O(n) per point.
	for x := int(n) + 1; x <= 2*int(n); x++ {
		// xMinusJ[j] = x - j, and P(x) = ∏_j xMinusJ[j].
		xMinusJ := make([]*mathlib.Zr, m)
		px := curve.NewZrFromInt(1)
		for j := 0; j < m; j++ {
			xMinusJ[j] = fieldDiffInt(x, j, curve)
			px = curve.ModMul(px, xMinusJ[j], curve.GroupOrder)
		}

		// Batch-invert xMinusJ so L_i(x) = px * xMinusJ[i]^{-1} * denomInvs[i].
		xMinusJInvs := BatchInverse(xMinusJ, curve)

		val := curve.NewZrFromInt(0)
		for i := 0; i < m; i++ {
			li := curve.ModMul(px, xMinusJInvs[i], curve.GroupOrder)
			li = curve.ModMul(li, denomInvs[i], curve.GroupOrder)
			val = curve.ModAdd(val, curve.ModMul(li, valuesOverN[i], curve.GroupOrder), curve.GroupOrder)
		}
		result[x] = val
	}
	return result, nil
}

type cspRangeProver struct {
	VGenerators  []*mathlib.G1  // two generators to commit to v
	AGenerators  []*mathlib.G1  // generators to commit to bits encoded as polynomial a(X)
	BGenerators  []*mathlib.G1  // generators to commit to b(X) by committing to b(n+1),...b(2n)
	VCommitment  *mathlib.G1    // commitment to value v
	NumberOfBits uint64         // number of bits n; the value must lie in [0, 2^n - 1]
	v            *mathlib.Zr    // value
	r            *mathlib.Zr    // randomness to mask v
	Curve        *mathlib.Curve // curve
}

type pokCommitment struct {
	A *mathlib.G1   // first message A = s[0].g[0] + s[1].g[1]
	Z []*mathlib.Zr // last message
}

type CspRangeProof struct {
	// commitment to a_0,a_1,...,a_n,b_0,b_{n+1},...,b_{2n}
	pComm *mathlib.G1
	// proof of knowledge of VCommitment over VGenerators
	pokV pokCommitment
	// prover evaluation a(c) for checking multiplication of polynomials
	u *mathlib.Zr
	// Commitment of vector to blind the CSP witness: a_0,a_1,..,a_n,b_0,b_{n+1},...,b_{2n},v,r
	sComm *mathlib.G1 // commitment to blinding vector s_0...s_{2n+1}
	sEval *mathlib.Zr // linear form on the blinding vector
	// CSP proof for: Com(wit) = pExt + \epsilon\cdot sComm, L(wit) = L(x) + \epsilon\cdot sEval
	cspProof CSPProof
}

func (rp *cspRangeProver) Prove() (*CspRangeProof, error) {
	tr := Transcript{Curve: rp.Curve}
	tr.InitHasher()
	n := rp.NumberOfBits
	rand, err := rp.Curve.Rand()
	if err != nil {
		return nil, errors.New("Unable to initialize rng")
	}

	// Absorb the public statement.
	tr.Absorb(rp.VGenerators[0].Bytes())
	tr.Absorb(rp.VGenerators[1].Bytes())

	// Schnorr proof of knowledge for VCommitment = v·G_v + r·G_r.
	// Prover samples blinding scalars, commits, then responds to the FS challenge.
	pokTv := rp.Curve.NewRandomZr(rand)
	pokTr := rp.Curve.NewRandomZr(rand)
	pokA := rp.Curve.MultiScalarMul(rp.VGenerators, []*mathlib.Zr{pokTv, pokTr})
	tr.Absorb(pokA.Bytes())
	pokE, err := tr.Squeeze()
	if err != nil {
		return nil, errors.New("unable to obtain PoK challenge")
	}
	// z_v = t_v + e·v,  z_r = t_r + e·r
	pokZv := rp.Curve.ModAdd(pokTv, rp.Curve.ModMul(pokE, rp.v, rp.Curve.GroupOrder), rp.Curve.GroupOrder)
	pokZr := rp.Curve.ModAdd(pokTr, rp.Curve.ModMul(pokE, rp.r, rp.Curve.GroupOrder), rp.Curve.GroupOrder)

	// Step 1: Compute witness p = aCoeffs || bCoeffs where
	//   aCoeffs = [a_0, a_1, ..., a_n]:  a_1..a_n are bits of v, a_0 is random
	//   bCoeffs = [b_0, b_{n+1}, ..., b_{2n}]:  b_i = a(i)*(a(i)-1)
	a0 := rp.Curve.NewRandomZr(rand)
	b0 := rp.Curve.ModSub(a0, rp.Curve.NewZrFromUint64(1), rp.Curve.GroupOrder)
	bitsOfV, err := toBits(rp.v, n, rp.Curve)
	if err != nil {
		return nil, errors.Errorf("Number does not fit in %d bits", n)
	}

	aCoeffs := make([]*mathlib.Zr, n+1)
	bCoeffs := make([]*mathlib.Zr, n+1)
	aCoeffs[0] = a0
	copy(aCoeffs[1:], bitsOfV)

	// Extend a(X) to all of {0,...,2n} and compute b(i) = a(i)*(a(i)-1).
	aCoeffsExt, err := interpolate(n, aCoeffs, rp.Curve)
	if err != nil {
		return nil, errors.New("Error while extending a polynomial")
	}
	bCoeffs[0] = rp.Curve.ModMul(a0, b0, rp.Curve.GroupOrder)
	for i := uint64(1); i <= n; i++ {
		ai := aCoeffsExt[n+i]
		aiMinus1 := rp.Curve.ModSub(ai, rp.Curve.NewZrFromUint64(1), rp.Curve.GroupOrder)
		bCoeffs[i] = rp.Curve.ModMul(ai, aiMinus1, rp.Curve.GroupOrder)
	}

	p := make([]*mathlib.Zr, 2*n+2)
	g := make([]*mathlib.G1, 2*n+2)
	copy(p, aCoeffs)
	copy(p[n+1:], bCoeffs)
	copy(g, rp.AGenerators)
	copy(g[n+1:], rp.BGenerators)

	// First prover message: pComm = MSM(g, p). Absorb and squeeze eta, c.
	pComm := rp.Curve.MultiScalarMul(g, p)
	tr.Absorb(pComm.Bytes())
	eta, err := tr.Squeeze()
	if err != nil {
		return nil, errors.New("Unable to obtain challenge eta")
	}
	c, err := tr.Squeeze()
	if err != nil {
		return nil, errors.New("Unable to obtain challenge c")
	}

	// Compute u = a(c) via Lagrange interpolation, absorb it, then squeeze gamma.
	mu, err := getLagrangeMultipliers(n, c, rp.Curve)
	if err != nil {
		return nil, errors.New("Unable to obtain lagrange multipliers")
	}
	nu, err := getLagrangeMultipliersPartial(n, c, rp.Curve)
	if err != nil {
		return nil, errors.New("Unable to obtain partial lagrange multipliers")
	}
	u := InnerProduct(aCoeffs, mu, rp.Curve)

	tr.Absorb(u.Bytes())
	gamma, err := tr.Squeeze()
	if err != nil {
		return nil, errors.New("Unable to obtain challenge gamma")
	}

	// Extended commitment: pCommExt = pComm + eta * VCommitment.
	pCommExt := pComm.Copy()
	pCommExt.Add(rp.VCommitment.Copy().Mul(eta))

	// Extended witness pExt = aCoeffs || bCoeffs || v || r
	// over generators  gExt = AGenerators || BGenerators || VGenerators.
	pExt := make([]*mathlib.Zr, 2*n+4)
	gExt := make([]*mathlib.G1, 2*n+4)
	copy(pExt, p)
	pExt[2*n+2] = rp.v.Copy()
	pExt[2*n+3] = rp.r.Copy()
	copy(gExt, g)
	gExt[2*n+2] = rp.VGenerators[0].Copy().Mul(eta)
	gExt[2*n+3] = rp.VGenerators[1].Copy().Mul(eta)

	// Build aggregated linear form lf = L1 + gamma*L2 + gamma^2*L3 over pExt.
	//
	// pExt layout  [0..n]=aCoeffs  [n+1..2n+1]=bCoeffs  [2n+2]=v  [2n+3]=r
	//
	// L1: eta*2^{i-1} at [1..n], -eta at [2n+2]          → checks eta*(Σ a_i·2^{i-1} - v) = 0
	// L2: mu[i]       at [0..n]                            → checks a(c) = u
	// L3: nu[k]       at [n+1..2n+1] (k=0..n)             → checks b(c) = u*(u-1)
	gammaSquare := rp.Curve.ModMul(gamma, gamma, rp.Curve.GroupOrder)
	lf := make([]*mathlib.Zr, len(pExt))
	for i := range lf {
		lf[i] = rp.Curve.NewZrFromInt(0)
	}
	// L1 contributions.
	pow2 := rp.Curve.NewZrFromInt(1)
	for i := uint64(1); i <= n; i++ {
		lf[i] = rp.Curve.ModMul(eta, pow2, rp.Curve.GroupOrder)
		pow2 = rp.Curve.ModMul(pow2, rp.Curve.NewZrFromInt(2), rp.Curve.GroupOrder)
	}
	negEta := eta.Copy()
	negEta.Neg()
	lf[2*n+2] = negEta
	// L2 contributions: add gamma*mu[i] at positions 0..n.
	for i := uint64(0); i <= n; i++ {
		lf[i] = rp.Curve.ModAdd(lf[i], rp.Curve.ModMul(gamma, mu[i], rp.Curve.GroupOrder), rp.Curve.GroupOrder)
	}
	// L3 contributions: gamma^2*nu[k] at positions n+1..2n+1.
	for k := uint64(0); k <= n; k++ {
		lf[n+1+k] = rp.Curve.ModMul(gammaSquare, nu[k], rp.Curve.GroupOrder)
	}

	// Claimed value: lVal = gamma*u + gamma^2*u*(u-1)  (L1(pExt)=0 for honest prover).
	uMinus1 := rp.Curve.ModSub(u, rp.Curve.NewZrFromUint64(1), rp.Curve.GroupOrder)
	lVal := rp.Curve.ModAdd(
		rp.Curve.ModMul(gamma, u, rp.Curve.GroupOrder),
		rp.Curve.ModMul(gammaSquare, rp.Curve.ModMul(u, uMinus1, rp.Curve.GroupOrder), rp.Curve.GroupOrder),
		rp.Curve.GroupOrder,
	)

	// ZK blinding: random sBlind, commit it, evaluate L on it.
	sBlind := make([]*mathlib.Zr, len(pExt))
	for i := range sBlind {
		sBlind[i] = rp.Curve.NewRandomZr(rand)
	}
	sComm := rp.Curve.MultiScalarMul(gExt, sBlind)
	sVal := InnerProduct(lf, sBlind, rp.Curve)
	tr.Absorb(sComm.Bytes())
	tr.Absorb(sVal.Bytes())

	rho, err := tr.Squeeze()
	if err != nil {
		return nil, errors.New("Unable to obtain challenge rho")
	}

	// Blinded witness: wit = pExt + rho*sBlind  so that
	//   MSM(gExt, wit) = pCommExt + rho*sComm
	//   L(wit)         = lVal + rho*sVal
	wit := make([]*mathlib.Zr, len(pExt))
	for i := range pExt {
		wit[i] = rp.Curve.ModAdd(
			pExt[i],
			rp.Curve.ModMul(rho, sBlind[i], rp.Curve.GroupOrder),
			rp.Curve.GroupOrder,
		)
	}
	witComm := pCommExt.Copy()
	witComm.Add(sComm.Mul(rho))
	witVal := rp.Curve.ModAdd(lVal, rp.Curve.ModMul(rho, sVal, rp.Curve.GroupOrder), rp.Curve.GroupOrder)

	// Pad witness / generators / linear form to the next power of 2 for CSP.
	witSize := uint64(len(wit))
	cspRounds := uint64(0)
	paddedSize := uint64(1)
	for paddedSize < witSize {
		paddedSize <<= 1
		cspRounds++
	}
	for uint64(len(wit)) < paddedSize {
		wit = append(wit, rp.Curve.NewZrFromInt(0))
		gExt = append(gExt, rp.Curve.GenG1)
		lf = append(lf, rp.Curve.NewZrFromInt(0))
	}

	// Non-ZK CSP proof for the blinded statement.
	cspP := &cspProver{
		Commitment:     witComm,
		Generators:     gExt,
		LinearForm:     lf,
		Value:          witVal,
		NumberOfRounds: cspRounds,
		Curve:          rp.Curve,
		witness:        wit,
	}
	cspProof, err := cspP.Prove()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate CSP proof")
	}

	return &CspRangeProof{
		pComm:    pComm,
		pokV:     pokCommitment{A: pokA, Z: []*mathlib.Zr{pokZv, pokZr}},
		u:        u,
		sComm:    sComm,
		sEval:    sVal,
		cspProof: *cspProof,
	}, nil
}

type cspRangeVerifier struct {
	VGenerators  []*mathlib.G1 // generators for the value commitment
	AGenerators  []*mathlib.G1 // generators for the bits polynomial a(X)
	BGenerators  []*mathlib.G1 // generators for b(X) at points {0, n+1, ..., 2n}
	VCommitment  *mathlib.G1   // commitment to value v
	NumberOfBits uint64        // number of bits n; the value must lie in [0, 2^n - 1]
	Curve        *mathlib.Curve
}

// Verify checks that proof is a valid CSP range proof against the public statement.
// It mirrors the prover transcript exactly, reconstructs all challenges, rebuilds
// the aggregated linear form, and delegates the final check to cspVerifier.
func (rv *cspRangeVerifier) Verify(proof *CspRangeProof) error {
	tr := Transcript{Curve: rv.Curve}
	tr.InitHasher()
	n := rv.NumberOfBits

	// Replay transcript: absorb the same public statement as the prover.
	tr.Absorb(rv.VGenerators[0].Bytes())
	tr.Absorb(rv.VGenerators[1].Bytes())

	// Verify Schnorr PoK for VCommitment = v·G_v + r·G_r.
	// Check: z_v·G_v + z_r·G_r == pokA + e·V
	tr.Absorb(proof.pokV.A.Bytes())
	pokE, err := tr.Squeeze()
	if err != nil {
		return errors.New("unable to recompute PoK challenge")
	}
	pokLHS := rv.Curve.MultiScalarMul(rv.VGenerators, proof.pokV.Z)
	pokRHS := proof.pokV.A.Copy()
	pokRHS.Add(rv.VCommitment.Copy().Mul(pokE))
	if !pokLHS.Equals(pokRHS) {
		return errors.New("proof of knowledge for value commitment failed")
	}

	tr.Absorb(proof.pComm.Bytes())

	eta, err := tr.Squeeze()
	if err != nil {
		return errors.New("unable to recompute challenge eta")
	}
	c, err := tr.Squeeze()
	if err != nil {
		return errors.New("unable to recompute challenge c")
	}

	//start := time.Now()
	mu, err := getLagrangeMultipliers(n, c, rv.Curve)
	if err != nil {
		return errors.New("unable to obtain lagrange multipliers")
	}
	nu, err := getLagrangeMultipliersPartial(n, c, rv.Curve)
	if err != nil {
		return errors.New("unable to obtain partial lagrange multipliers")
	}
	//fmt.Printf("Tine taken for Lagrange multipliers: %d", time.Since(start).Milliseconds())

	tr.Absorb(proof.u.Bytes())
	gamma, err := tr.Squeeze()
	if err != nil {
		return errors.New("unable to recompute challenge gamma")
	}

	// pCommExt = pComm + eta * VCommitment
	pCommExt := proof.pComm.Copy()
	pCommExt.Add(rv.VCommitment.Copy().Mul(eta))

	// Rebuild gExt = AGenerators || BGenerators || VGenerators (size 2n+4).
	gExt := make([]*mathlib.G1, 2*n+4)
	copy(gExt, rv.AGenerators)
	copy(gExt[n+1:], rv.BGenerators)
	gExt[2*n+2] = rv.VGenerators[0].Copy().Mul(eta)
	gExt[2*n+3] = rv.VGenerators[1].Copy().Mul(eta)

	// Rebuild lf = L1 + gamma*L2 + gamma^2*L3 — identical to the prover.
	gammaSquare := rv.Curve.ModMul(gamma, gamma, rv.Curve.GroupOrder)
	lf := make([]*mathlib.Zr, 2*n+4)
	for i := range lf {
		lf[i] = rv.Curve.NewZrFromInt(0)
	}
	pow2 := rv.Curve.NewZrFromInt(1)
	for i := uint64(1); i <= n; i++ {
		lf[i] = rv.Curve.ModMul(eta, pow2, rv.Curve.GroupOrder)
		pow2 = rv.Curve.ModMul(pow2, rv.Curve.NewZrFromInt(2), rv.Curve.GroupOrder)
	}
	negEta := eta.Copy()
	negEta.Neg()
	lf[2*n+2] = negEta
	for i := uint64(0); i <= n; i++ {
		lf[i] = rv.Curve.ModAdd(lf[i], rv.Curve.ModMul(gamma, mu[i], rv.Curve.GroupOrder), rv.Curve.GroupOrder)
	}
	for k := uint64(0); k <= n; k++ {
		lf[n+1+k] = rv.Curve.ModMul(gammaSquare, nu[k], rv.Curve.GroupOrder)
	}

	// lVal = gamma*u + gamma^2*u*(u-1)
	uMinus1 := rv.Curve.ModSub(proof.u, rv.Curve.NewZrFromUint64(1), rv.Curve.GroupOrder)
	lVal := rv.Curve.ModAdd(
		rv.Curve.ModMul(gamma, proof.u, rv.Curve.GroupOrder),
		rv.Curve.ModMul(gammaSquare, rv.Curve.ModMul(proof.u, uMinus1, rv.Curve.GroupOrder), rv.Curve.GroupOrder),
		rv.Curve.GroupOrder,
	)

	// Absorb sComm and sEval, squeeze rho.
	tr.Absorb(proof.sComm.Bytes())
	tr.Absorb(proof.sEval.Bytes())
	rho, err := tr.Squeeze()
	if err != nil {
		return errors.New("unable to recompute challenge rho")
	}

	// witComm = pCommExt + rho*sComm
	witComm := pCommExt.Copy()
	witComm.Add(proof.sComm.Copy().Mul(rho))

	// witVal = lVal + rho*sEval
	witVal := rv.Curve.ModAdd(lVal, rv.Curve.ModMul(rho, proof.sEval, rv.Curve.GroupOrder), rv.Curve.GroupOrder)

	// Pad gExt and lf to the next power of 2 (same logic as prover).
	extSize := uint64(len(gExt))
	cspRounds := uint64(0)
	paddedSize := uint64(1)
	for paddedSize < extSize {
		paddedSize <<= 1
		cspRounds++
	}
	for uint64(len(gExt)) < paddedSize {
		gExt = append(gExt, rv.Curve.GenG1)
		lf = append(lf, rv.Curve.NewZrFromInt(0))
	}

	cspV := &cspVerifier{
		Commitment:     witComm,
		Generators:     gExt,
		LinearForm:     lf,
		Value:          witVal,
		NumberOfRounds: cspRounds,
		Curve:          rv.Curve,
	}
	return cspV.Verify(&proof.cspProof)
}
