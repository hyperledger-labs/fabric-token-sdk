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
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/dlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	cryptodlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	msp2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509"
	"github.com/pkg/errors"
)

type FabTokenPublicParamsGenerator struct{}

func NewFabTokenPublicParamsGenerator() *FabTokenPublicParamsGenerator {
	return &FabTokenPublicParamsGenerator{}
}

func (f *FabTokenPublicParamsGenerator) Generate(tms *topology.TMS, wallets *topology.Wallets, args ...interface{}) ([]byte, error) {
	precision := fabtoken.DefaultPrecision
	if len(args) == 2 {
		// First is empty

		// Second is the `precision`.
		precisionStr, ok := args[1].(string)
		if !ok {
			return nil, errors.Errorf("expected string as first argument")
		}
		var err error
		precision, err = strconv.ParseUint(precisionStr, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse max token value [%s] to uint64", precisionStr)
		}
	}
	pp, err := fabtoken.Setup(precision)
	if err != nil {
		return nil, err
	}

	if len(tms.Auditors) != 0 {
		if len(wallets.Auditors) == 0 {
			return nil, errors.Errorf("no auditor wallets provided")
		}
		for _, auditor := range wallets.Auditors {
			// Build an MSP Identity
			km, _, err := x509.NewKeyManager(auditor.Path, "", msp2.AuditorMSPID, nil, auditor.Opts)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to create x509 km")
			}
			id, _, err := km.Identity(nil)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to get identity")
			}
			if tms.Auditors[0] == auditor.ID {
				wrap, err := identity.WrapWithType(msp2.X509Identity, id)
				if err != nil {
					return nil, errors.WithMessagef(err, "failed to create x509 identity for auditor [%v]", auditor)
				}
				pp.AddAuditor(wrap)
			}
		}
	}

	if len(tms.Issuers) != 0 {
		if len(wallets.Issuers) == 0 {
			return nil, errors.Errorf("no issuer wallets provided")
		}
		for _, issuer := range wallets.Issuers {
			// Build an MSP Identity
			km, _, err := x509.NewKeyManager(issuer.Path, "", msp2.AuditorMSPID, nil, issuer.Opts)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to create x509 km")
			}
			id, _, err := km.Identity(nil)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to get identity")
			}
			if tms.Issuers[0] == issuer.ID {
				wrap, err := identity.WrapWithType(msp2.X509Identity, id)
				if err != nil {
					return nil, errors.WithMessagef(err, "failed to create x509 identity for auditor [%v]", issuer)
				}
				pp.AddIssuer(wrap)
			}
		}
	}

	ppRaw, err := pp.Serialize()
	if err != nil {
		return nil, err
	}

	tms.Wallets = wallets

	return ppRaw, nil
}

type DLogPublicParamsGenerator struct {
	DefaultCurveID math3.CurveID
}

func NewDLogPublicParamsGenerator(defaultCurveID math3.CurveID) *DLogPublicParamsGenerator {
	return &DLogPublicParamsGenerator{DefaultCurveID: defaultCurveID}
}

func (d *DLogPublicParamsGenerator) Generate(tms *topology.TMS, wallets *topology.Wallets, args ...interface{}) ([]byte, error) {
	if len(args) != 2 {
		return nil, errors.Errorf("invalid number of arguments, expected 2, got %d", len(args))
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

	bits := uint64(64)
	if len(args) > 1 {
		baseArg, ok := args[1].(string)
		if !ok {
			return nil, errors.Errorf("invalid argument type, expected string, got %T", args[1])
		}
		bits, err = strconv.ParseUint(baseArg, 10, 32)
		if err != nil {
			return nil, err
		}
	}
	pp, err := cryptodlog.Setup(bits, ipkBytes, curveID)
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
			km, _, err := x509.NewKeyManager(auditor.Path, "", msp2.AuditorMSPID, nil, auditor.Opts)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to create x509 km")
			}
			id, _, err := km.Identity(nil)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to get identity")
			}
			if tms.Auditors[0] == auditor.ID {
				wrap, err := identity.WrapWithType(msp2.X509Identity, id)
				if err != nil {
					return nil, errors.WithMessagef(err, "failed to create x509 identity for auditor [%v]", auditor)
				}
				pp.AddAuditor(wrap)
			}
		}
	}

	if len(tms.Issuers) != 0 {
		if len(wallets.Issuers) == 0 {
			return nil, errors.Errorf("no issuer wallets provided")
		}
		for _, issuer := range wallets.Issuers {
			// Build an MSP Identity
			km, _, err := x509.NewKeyManager(issuer.Path, "", msp2.AuditorMSPID, nil, issuer.Opts)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to create x509 km")
			}
			id, _, err := km.Identity(nil)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to get identity")
			}
			if tms.Issuers[0] == issuer.ID {
				wrap, err := identity.WrapWithType(msp2.X509Identity, id)
				if err != nil {
					return nil, errors.WithMessagef(err, "failed to create x509 identity for issuer [%v]", issuer)
				}
				pp.AddIssuer(wrap)
			}
		}
	}

	ppRaw, err := pp.Serialize()
	if err != nil {
		return nil, err
	}

	tms.Wallets = wallets

	return ppRaw, nil
}
