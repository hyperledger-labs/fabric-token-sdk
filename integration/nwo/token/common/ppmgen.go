/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	math3 "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	cryptodlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger/fabric/msp"
	"github.com/pkg/errors"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
)

type FabTokenPublicParamsGenerator struct{}

func NewFabTokenPublicParamsGenerator() *FabTokenPublicParamsGenerator {
	return &FabTokenPublicParamsGenerator{}
}

func (f *FabTokenPublicParamsGenerator) Generate(tms *topology.TMS, wallets *generators.Wallets, args ...interface{}) ([]byte, error) {
	pp, err := fabtoken.Setup()
	if err != nil {
		return nil, err
	}

	if len(tms.Auditors) != 0 {
		if len(wallets.Auditors) == 0 {
			return nil, errors.Errorf("no auditor wallets provided")
		}
		for _, auditor := range wallets.Auditors {
			// Build an MSP Identity
			types := strings.Split(auditor.Type, ":")
			provider, err := x509.NewProvider(auditor.Path, types[1], nil)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to create x509 provider")
			}
			id, _, err := provider.Identity(nil)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to get identity")
			}
			pp.AddAuditor(id)
		}
	}

	if len(tms.Issuers) != 0 {
		// TODO:
	}

	ppRaw, err := pp.Serialize()
	if err != nil {
		return nil, err
	}
	return ppRaw, nil
}

type DLogPublicParamsGenerator struct {
	CurveID math3.CurveID
}

func NewDLogPublicParamsGenerator(curveID math3.CurveID) *DLogPublicParamsGenerator {
	return &DLogPublicParamsGenerator{CurveID: curveID}
}

func (d *DLogPublicParamsGenerator) Generate(tms *topology.TMS, wallets *generators.Wallets, args ...interface{}) ([]byte, error) {
	if len(args) != 3 {
		return nil, errors.Errorf("invalid number of arguments, expected 3, got %d", len(args))
	}
	// first argument is the idemix root path
	idemixRootPath, ok := args[0].(string)
	if !ok {
		return nil, errors.Errorf("invalid argument type, expected string, got %T", args[0])
	}
	path := filepath.Join(idemixRootPath, msp.IdemixConfigDirMsp, msp.IdemixConfigFileIssuerPublicKey)
	ipkBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	baseArg, ok := args[1].(string)
	if !ok {
		return nil, errors.Errorf("invalid argument type, expected string, got %T", args[1])
	}
	base, err := strconv.ParseInt(baseArg, 10, 64)
	if err != nil {
		return nil, err
	}
	expArg, ok := args[2].(string)
	if !ok {
		return nil, errors.Errorf("invalid argument type, expected string, got %T", args[2])
	}
	exp, err := strconv.ParseInt(expArg, 10, 32)
	if err != nil {
		return nil, err
	}
	pp, err := cryptodlog.Setup(base, int(exp), ipkBytes, d.CurveID)
	if err != nil {
		return nil, err
	}

	if len(tms.Auditors) != 0 {
		if len(wallets.Auditors) == 0 {
			return nil, errors.Errorf("no auditor wallets provided")
		}
		for _, auditor := range wallets.Auditors {
			// Build an MSP Identity
			types := strings.Split(auditor.Type, ":")
			provider, err := x509.NewProvider(auditor.Path, types[1], nil)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to create x509 provider")
			}
			id, _, err := provider.Identity(nil)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to get identity")
			}
			pp.AddAuditor(id)
		}
	}

	if len(tms.Issuers) != 0 {
		// TODO
	}

	ppRaw, err := pp.Serialize()
	if err != nil {
		return nil, err
	}
	return ppRaw, nil
}
