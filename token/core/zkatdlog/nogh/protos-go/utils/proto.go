/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	mathlib "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/protos-go/math"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/utils"
)

func ToProtoG1Slice(input []*mathlib.G1) ([]*math.G1, error) {
	return utils.ToProtosSliceFunc(input, func(s *mathlib.G1) (*math.G1, error) {
		return ToProtoG1(s)
	})
}

func ToProtoG1(s *mathlib.G1) (*math.G1, error) {
	if s == nil {
		return &math.G1{}, nil
	}
	return &math.G1{Raw: s.Bytes()}, nil
}

func FromG1ProtoSlice(curve mathlib.CurveID, generators []*math.G1) ([]*mathlib.G1, error) {
	return utils.FromProtosSliceFunc(generators, func(s *math.G1) (*mathlib.G1, error) {
		return FromG1Proto(curve, s)
	})
}

func FromG1Proto(curve mathlib.CurveID, p *math.G1) (*mathlib.G1, error) {
	if p == nil {
		return nil, nil
	}
	return mathlib.Curves[curve].NewG1FromBytes(p.Raw)
}
