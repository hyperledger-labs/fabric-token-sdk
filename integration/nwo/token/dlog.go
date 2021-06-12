/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package token

import (
	"io/ioutil"
	"path/filepath"
	"strconv"

	"github.com/hyperledger/fabric/msp"

	cryptodlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
)

type DLogPublicParamsGenerator struct {
}

func (d *DLogPublicParamsGenerator) Generate(p *Platform, tms *TMS) ([]byte, error) {
	path := filepath.Join(p.FabricNetwork.DefaultIdemixOrgMSPDir(), msp.IdemixConfigDirMsp, msp.IdemixConfigFileIssuerPublicKey)
	ipkBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	base, err := strconv.ParseInt(tms.TokenChaincode.PublicParamsGenArgs[0], 10, 64)
	if err != nil {
		return nil, err
	}
	exp, err := strconv.ParseInt(tms.TokenChaincode.PublicParamsGenArgs[1], 10, 32)
	if err != nil {
		return nil, err
	}
	pp, err := cryptodlog.Setup(
		base,
		int(exp),
		ipkBytes,
	)
	if err != nil {
		return nil, err
	}

	ppRaw, err := pp.Serialize()
	if err != nil {
		return nil, err
	}
	return ppRaw, nil
}
