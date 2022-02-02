/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	fsc "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
)

func WithIssuerIdentity(label string) fsc.Option {
	return func(o *fsc.Options) error {
		to := topology.ToOptions(o)
		to.SetIssuers(append(to.Issuers(), label))

		if label != "_default_" {
			o.AddAlias(label)
		}

		fo := fabric.Options(o)
		fo.SetX509Identities(append(fo.X509Identities(), label))
		return nil
	}
}

func WithDefaultIssuerIdentity() fsc.Option {
	return WithIssuerIdentity("_default_")
}

func WithDefaultOwnerIdentity(driver string) fsc.Option {
	return WithOwnerIdentity(driver, "_default_")
}

func WithOwnerIdentity(driver string, label string) fsc.Option {
	return func(o *fsc.Options) error {
		to := topology.ToOptions(o)
		to.SetOwners(append(to.Owners(), label))

		if label != "_default_" {
			o.AddAlias(label)
		}

		fo := fabric.Options(o)
		switch driver {
		case "dlog":
			// skip
		case "fabtoken":
			fo.SetX509Identities(append(fo.X509Identities(), label))
		default:
			panic(fmt.Sprintf("unexpected driver [%s]", driver))
		}
		return nil
	}
}

func WithCertifierIdentity() fsc.Option {
	return func(o *fsc.Options) error {
		topology.ToOptions(o).SetCertifier(true)

		return nil
	}
}

func WithAuditorIdentity() fsc.Option {
	return func(o *fsc.Options) error {
		topology.ToOptions(o).SetAuditor(true)

		return nil
	}
}
