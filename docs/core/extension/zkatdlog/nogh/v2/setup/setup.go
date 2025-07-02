/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package setup

import (
	mathlib "github.com/IBM/mathlib"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

const (
	DLogIdentifier = v1.DLogIdentifier
	ProtocolV2     = 2
)

// PublicParams are identical to the v1's public parameters. What changes is the driver version.
type PublicParams = v1.PublicParams

func NewPublicParamsFromBytes(
	raw []byte,
	driverName driver.TokenDriverName,
	driverVersion driver.TokenDriverVersion,
) (*PublicParams, error) {
	return v1.NewPublicParamsFromBytes(raw, driverName, driverVersion)
}

func Setup(bitLength uint64, idemixIssuerPK []byte, idemixCurveID mathlib.CurveID) (*PublicParams, error) {
	return v1.NewWith(DLogIdentifier, ProtocolV2, bitLength, idemixIssuerPK, idemixCurveID)
}
