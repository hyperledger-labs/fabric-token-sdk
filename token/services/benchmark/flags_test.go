/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package benchmark

import (
	"runtime"
	"testing"
	"time"

	math "github.com/IBM/mathlib"
)

func TestBits_DefaultsAndParsing(t *testing.T) {
	obits := *bits
	defer func() { *bits = obits }()

	// when unset, defaults are returned
	*bits = ""
	vals, err := Bits(32, 64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vals) != 2 || vals[0] != 32 || vals[1] != 64 {
		t.Fatalf("unexpected defaults: %v", vals)
	}

	// set flag to single value
	*bits = "128"
	vals, err = Bits(32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vals) != 1 || vals[0] != 128 {
		t.Fatalf("unexpected parsed bits: %v", vals)
	}

	// set flag to comma-separated values
	*bits = "16, 64,256"
	vals, err = Bits()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vals) != 3 || vals[0] != 16 || vals[1] != 64 || vals[2] != 256 {
		t.Fatalf("unexpected parsed bits: %v", vals)
	}
}

func TestDuration_DefaultAndZero(t *testing.T) {
	od := *duration
	defer func() { *duration = od }()

	// default value
	*duration = 1 * time.Second
	d := Duration()
	if d != 1*time.Second {
		t.Fatalf("expected default 1s, got %v", d)
	}

	// set duration to 0 -> should get fallback 1s
	*duration = 0
	d = Duration()
	if d != 1*time.Second {
		t.Fatalf("expected fallback 1s, got %v", d)
	}

	// set duration to 2s
	*duration = 2 * time.Second
	d = Duration()
	if d != 2*time.Second {
		t.Fatalf("expected 2s, got %v", d)
	}
}

func TestCurves_DefaultsAndParsing(t *testing.T) {
	oc := *curves
	defer func() { *curves = oc }()

	// default when unset
	*curves = ""
	defaults := []math.CurveID{math.BN254}
	res := Curves(defaults...)
	if len(res) != 1 || res[0] != math.BN254 {
		t.Fatalf("unexpected default curves: %v", res)
	}

	// set numeric id
	*curves = "2"
	res = Curves()
	if len(res) != 1 || res[0] != math.CurveID(2) {
		t.Fatalf("unexpected numeric curve parsed: %v", res)
	}

	// set name
	*curves = "BN254, BLS12_381_BBS_GURVY_FAST_RNG"
	res = Curves()
	if len(res) != 2 {
		t.Fatalf("unexpected number of curves: %v", res)
	}
	if res[0] != math.BN254 {
		t.Fatalf("first curve expected BN254, got %v", res[0])
	}
}

func TestNumInputsOutputsAndIntegers(t *testing.T) {
	oni := *numInputs
	ono := *numOutputs
	defer func() { *numInputs = oni; *numOutputs = ono }()

	// defaults
	*numInputs = ""
	*numOutputs = ""
	ni, err := NumInputs(1, 2)
	if err != nil || len(ni) != 2 || ni[0] != 1 || ni[1] != 2 {
		t.Fatalf("unexpected default num inputs: %v err:%v", ni, err)
	}

	// set inputs
	*numInputs = "3,4"
	ni, err = NumInputs()
	if err != nil || len(ni) != 2 || ni[0] != 3 || ni[1] != 4 {
		t.Fatalf("unexpected parsed num inputs: %v err:%v", ni, err)
	}

	// outputs
	*numOutputs = ""
	nq, err := NumOutputs(5)
	if err != nil || len(nq) != 1 || nq[0] != 5 {
		t.Fatalf("unexpected default num outputs: %v err:%v", nq, err)
	}
	*numOutputs = "7,8,9"
	nq, err = NumOutputs()
	if err != nil || len(nq) != 3 || nq[0] != 7 || nq[2] != 9 {
		t.Fatalf("unexpected parsed num outputs: %v err:%v", nq, err)
	}
}

func TestWorkers_NumCPUTokenAndParsing(t *testing.T) {
	ow := *workers
	defer func() { *workers = ow }()

	// default
	*workers = ""
	w, err := Workers(1)
	if err != nil || len(w) != 1 || w[0] != 1 {
		t.Fatalf("unexpected default workers: %v err:%v", w, err)
	}

	// NumCPU token
	*workers = "NumCPU"
	w, err = Workers()
	if err != nil || len(w) != 1 || w[0] != runtime.NumCPU() {
		t.Fatalf("unexpected NumCPU replacement: %v err:%v", w, err)
	}

	// list of ints
	*workers = "1,2, 4"
	w, err = Workers()
	if err != nil || len(w) != 3 || w[2] != 4 {
		t.Fatalf("unexpected parsed workers: %v err:%v", w, err)
	}
}
