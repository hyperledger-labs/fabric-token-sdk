/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"os"
	"path/filepath"
	"strconv"

	msp "github.com/IBM/idemix"
	math3 "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/dlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	fabtokenv1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/setup"
	dlognoghv1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509"
	"github.com/pkg/errors"
)

type FabTokenPublicParamsGenerator struct{}

func NewFabTokenPublicParamsGenerator() *FabTokenPublicParamsGenerator {
	return &FabTokenPublicParamsGenerator{}
}

func (f *FabTokenPublicParamsGenerator) Generate(tms *topology.TMS, wallets *topology.Wallets, args ...interface{}) ([]byte, error) {
	precision := fabtokenv1.DefaultPrecision
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
	pp, err := fabtokenv1.Setup(precision)
	if err != nil {
		return nil, err
	}

	keyStore := x509.NewKeyStore(kvs.NewTrackedMemory())
	if len(tms.Auditors) != 0 {
		if len(wallets.Auditors) == 0 {
			return nil, errors.Errorf("no auditor wallets provided")
		}
		for _, auditor := range wallets.Auditors {
			// Build an MSP Identity
			km, _, err := x509.NewKeyManager(auditor.Path, nil, auditor.Opts, keyStore)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to create x509 km")
			}
			id, _, err := km.Identity(context.Background(), nil)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to get identity")
			}
			if tms.Auditors[0] == auditor.ID {
				wrap, err := identity.WrapWithType(x509.IdentityType, id)
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
		issuersSet := collections.NewSet(tms.Issuers...)
		for _, issuer := range wallets.Issuers {
			// Build an MSP Identity
			km, _, err := x509.NewKeyManager(issuer.Path, nil, issuer.Opts, keyStore)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to create x509 km")
			}
			id, _, err := km.Identity(context.Background(), nil)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to get identity")
			}
			if issuersSet.Contains(issuer.ID) {
				wrap, err := identity.WrapWithType(x509.IdentityType, id)
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
	pp, err := dlognoghv1.Setup(bits, ipkBytes, curveID)
	if err != nil {
		return nil, err
	}

	keyStore := x509.NewKeyStore(kvs.NewTrackedMemory())
	if len(tms.Auditors) != 0 {
		if len(wallets.Auditors) == 0 {
			return nil, errors.Errorf("no auditor wallets provided")
		}
		for _, auditor := range wallets.Auditors {
			// Build an MSP Identity
			km, _, err := x509.NewKeyManager(auditor.Path, nil, auditor.Opts, keyStore)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to create x509 km")
			}
			id, _, err := km.Identity(context.Background(), nil)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to get identity")
			}
			if tms.Auditors[0] == auditor.ID {
				wrap, err := identity.WrapWithType(x509.IdentityType, id)
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
		issuersSet := collections.NewSet(tms.Issuers...)
		for _, issuer := range wallets.Issuers {
			// Build an MSP Identity
			km, _, err := x509.NewKeyManager(issuer.Path, nil, issuer.Opts, keyStore)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to create x509 km")
			}
			id, _, err := km.Identity(context.Background(), nil)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to get identity")
			}
			if issuersSet.Contains(issuer.ID) {
				wrap, err := identity.WrapWithType(x509.IdentityType, id)
				if err != nil {
					return nil, errors.WithMessagef(err, "failed to create x509 identity for issuer [%v]", issuer)
				}
				pp.AddIssuer(wrap)
			}
		}
	}

	// validate before serialization
	if err := pp.Validate(); err != nil {
		return nil, errors.Wrapf(err, "failed to validate public parameters")
	}

	// finalization
	ppRaw, err := pp.Serialize()
	if err != nil {
		return nil, err
	}
	tms.Wallets = wallets

	return ppRaw, nil
}
