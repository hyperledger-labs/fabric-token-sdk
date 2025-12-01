/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package benchmark

import (
	"flag"
	"runtime"
	"strconv"
	"strings"
	"time"

	math "github.com/IBM/mathlib"
	math2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/common/crypto/math"
)

var (
	bits       = flag.String("bits", "", "a comma-separated list of bit sizes (32, 64,...)")
	duration   = flag.Duration("duration", 1*time.Second, "test duration (1s, 1m, 1h,...)")
	curves     = flag.String("curves", "", "comma-separated list of curves. Supported curves are: BN254, BLS12_381_BBS_GURVY, BLS12_381_BBS_GURVY_FAST_RNG")
	numInputs  = flag.String("num_inputs", "", "a comma-separate list of number of inputs (1,2,3,...)")
	numOutputs = flag.String("num_outputs", "", "a comma-separate list of number of outputs (1,2,3,...)")
	workers    = flag.String("workers", "", "a comma-separate list of workers (1,2,3,...,NumCPU), where NumCPU is converted to the number of available CPUs")
)

func Bits(defaults ...uint64) ([]uint64, error) {
	str := *bits

	if len(str) == 0 {
		return defaults, nil
	}

	components := strings.Split(str, ",")
	values := make([]uint64, 0, len(components))
	for _, s := range components {
		v, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return nil, err
		}
		values = append(values, uint64(v))
	}
	return values, nil
}

func Duration() time.Duration {
	d := *duration
	if d == 0 {
		return 1 * time.Second
	}

	return d
}

func Curves(curveIDs ...math.CurveID) []math.CurveID {
	str := *curves
	if len(str) == 0 {
		return curveIDs
	}

	components := strings.Split(str, ",")
	values := make([]math.CurveID, 0, len(components))
	for _, s := range components {
		s = strings.TrimSpace(s)
		v, err := strconv.Atoi(s)
		if err == nil {
			values = append(values, math.CurveID(v))
			continue
		}

		values = append(values, math2.StringToCurveID(s))
	}
	return values
}

func NumInputs(defaults ...int) ([]int, error) {
	str := *numInputs
	if len(str) == 0 {
		return defaults, nil
	}
	components := strings.Split(str, ",")
	values := make([]int, 0, len(components))
	for _, s := range components {
		v, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, nil
}

func NumOutputs(defaults ...int) ([]int, error) {
	str := *numOutputs
	if len(str) == 0 {
		return defaults, nil
	}
	components := strings.Split(str, ",")
	values := make([]int, 0, len(components))
	for _, s := range components {
		v, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, nil
}

func Workers(defaults ...int) ([]int, error) {
	str := *workers
	if len(str) == 0 {
		return defaults, nil
	}
	components := strings.Split(str, ",")
	values := make([]int, 0, len(components))
	for _, s := range components {
		s = strings.TrimSpace(s)
		v, err := strconv.Atoi(s)
		if err != nil {
			if s == "NumCPU" {
				v = runtime.NumCPU()
			} else {
				return nil, err
			}
		}
		values = append(values, v)
	}
	return values, nil
}
