/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"os"
	"path/filepath"
	"strconv"

	msp "github.com/IBM/idemix"
	math3 "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/dlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	msp2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/msp"
	cryptodlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/pkg/errors"
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
	if len(args) == 2 {
		// First is empty
		// Second is the max token value
		maxTokenValueStr, ok := args[1].(string)
		if !ok {
			return nil, errors.Errorf("expected string as first argument")
		}
		maxTokenValue, err := strconv.ParseUint(maxTokenValueStr, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse max token value [%s] to uint64", maxTokenValueStr)
		}
		pp.MaxToken = maxTokenValue
	}

	if len(tms.Auditors) != 0 {
		if len(wallets.Auditors) == 0 {
			return nil, errors.Errorf("no auditor wallets provided")
		}
		for _, auditor := range wallets.Auditors {
			// Build an MSP Identity
			provider, err := x509.NewProviderWithBCCSPConfig(auditor.Path, "", msp2.AuditorMSPID, nil, auditor.Opts)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to create x509 provider")
			}
			id, _, err := provider.Identity(nil)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to get identity")
			}
			if tms.Auditors[0] == auditor.ID {
				pp.AddAuditor(id)
			}
		}
	}

	//lint:ignore SA9003 To-Do pending
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
	DefaultCurveID math3.CurveID
}

func NewDLogPublicParamsGenerator(defaultCurveID math3.CurveID) *DLogPublicParamsGenerator {
	return &DLogPublicParamsGenerator{DefaultCurveID: defaultCurveID}
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
	ipkBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	curveID := d.DefaultCurveID
	if dlog.IsAries(tms) {
		curveID = math3.BLS12_381_BBS
	}

	baseArg, ok := args[1].(string)
	if !ok {
		return nil, errors.Errorf("invalid argument type, expected string, got %T", args[1])
	}
	base, err := strconv.ParseUint(baseArg, 10, 32)
	if err != nil {
		return nil, err
	}
	expArg, ok := args[2].(string)
	if !ok {
		return nil, errors.Errorf("invalid argument type, expected string, got %T", args[2])
	}
	exp, err := strconv.ParseUint(expArg, 10, 32)
	if err != nil {
		return nil, err
	}
	pp, err := cryptodlog.Setup(uint(base), uint(exp), ipkBytes, curveID)
	if err != nil {
		return nil, err
	}
	if err := pp.Validate(); err != nil {
		return nil, errors.Wrapf(err, "failed to validate public parameters")
	}

	if len(tms.Auditors) != 0 {
		if len(wallets.Auditors) == 0 {
			return nil, errors.Errorf("no auditor wallets provided")
		}
		for _, auditor := range wallets.Auditors {
			// Build an MSP Identity
			provider, err := x509.NewProviderWithBCCSPConfig(auditor.Path, "", msp2.AuditorMSPID, nil, auditor.Opts)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to create x509 provider")
			}
			id, _, err := provider.Identity(nil)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to get identity")
			}
			if tms.Auditors[0] == auditor.ID {
				pp.AddAuditor(id)
			}
		}
	}

	//lint:ignore SA9003 To-Do pending
	if len(tms.Issuers) != 0 {
		// TODO
	}

	ppRaw, err := pp.Serialize()
	if err != nil {
		return nil, err
	}
	return ppRaw, nil
}
