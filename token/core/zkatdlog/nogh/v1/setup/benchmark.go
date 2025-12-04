/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package setup

import (
	"fmt"
	"os"
	"path/filepath"

	math "github.com/IBM/mathlib"
	math2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/common/crypto/math"
)

type Configurations struct {
	configurations map[string]*PublicParams
}

func NewConfigurations(idemixTestdataPath string, bits []uint64, curveIDs []math.CurveID) (*Configurations, error) {
	configurations := map[string]*PublicParams{}
	for _, curveID := range curveIDs {
		var ipk []byte
		var err error
		switch curveID {
		case math.BN254:
			ipk, err = os.ReadFile(filepath.Join(idemixTestdataPath, "bn254", "idemix", "msp", "IssuerPublicKey"))
			if err != nil {
				return nil, err
			}
		case math.BLS12_381_BBS_GURVY:
			fallthrough
		case math2.BLS12_381_BBS_GURVY_FAST_RNG:
			ipk, err = os.ReadFile(filepath.Join(idemixTestdataPath, "bls12_381_bbs", "idemix", "msp", "IssuerPublicKey"))
			if err != nil {
				return nil, err
			}
		}

		for _, bit := range bits {
			pp, err := Setup(bit, ipk, curveID)
			if err != nil {
				return nil, err
			}
			configurations[key(bit, curveID)] = pp
		}
	}
	return &Configurations{
		configurations: configurations,
	}, nil
}

func (c *Configurations) Get(bits uint64, curveID math.CurveID) (*PublicParams, error) {
	pp, ok := c.configurations[key(bits, curveID)]
	if !ok {
		return nil, fmt.Errorf("configuration not found")
	}
	return pp, nil
}

func key(bits uint64, curveID math.CurveID) string {
	return fmt.Sprintf("%d-%d", bits, curveID)
}
