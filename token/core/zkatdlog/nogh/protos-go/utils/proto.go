/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	mathlib "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/protos-go/pp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/utils"
)

func ToProtoG1Slice(input []*mathlib.G1) ([]*pp.G1, error) {
	return utils.ToProtosSliceFunc(input, func(s *mathlib.G1) (*pp.G1, error) {
		return ToProtoG1(s)
	})
}

func ToProtoG1(s *mathlib.G1) (*pp.G1, error) {
	if s == nil {
		return &pp.G1{}, nil
	}
	return &pp.G1{Raw: s.Bytes()}, nil
}

func FromG1ProtoSlice(curve mathlib.CurveID, generators []*pp.G1) ([]*mathlib.G1, error) {
	res := make([]*mathlib.G1, len(generators))
	var err error
	for i, g := range generators {
		res[i], err = FromG1Proto(curve, g)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func FromG1Proto(curve mathlib.CurveID, p *pp.G1) (*mathlib.G1, error) {
	if p == nil {
		return nil, nil
	}
	return mathlib.Curves[curve].NewG1FromBytes(p.Raw)
}
