/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtokenv1

import (
	"context"
	"strconv"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	fabtokenv1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509"
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
				return nil, errors.WithMessagef(err, "failed to create x509 km")
			}
			id, _, err := km.Identity(context.Background(), nil)
			if err != nil {
				return nil, errors.WithMessagef(err, "failed to get identity")
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
				return nil, errors.WithMessagef(err, "failed to create x509 km")
			}
			id, _, err := km.Identity(context.Background(), nil)
			if err != nil {
				return nil, errors.WithMessagef(err, "failed to get identity")
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
