/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package token

import (
	cryptofabtoken "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
)

type FabTokenPublicParamsGenerator struct {
}

func (f *FabTokenPublicParamsGenerator) Generate(p *Platform, tms *TMS) ([]byte, error) {
	pp, err := cryptofabtoken.Setup()
	if err != nil {
		return nil, err
	}
	ppRaw, err := pp.Serialize()
	if err != nil {
		return nil, err
	}
	return ppRaw, nil
}
