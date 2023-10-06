/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
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
		return nil
	}
}

func WithIssuerIdentityWithHSM(label string) fsc.Option {
	return func(o *fsc.Options) error {
		to := topology.ToOptions(o)
		to.SetIssuers(append(to.Issuers(), label))
		to.UseHSMForIssuer(label)

		if label != "_default_" {
			o.AddAlias(label)
		}
		return nil
	}
}

func WithDefaultIssuerIdentity() fsc.Option {
	return WithIssuerIdentity("_default_")
}

func WithDefaultIssuerIdentityWithHSM() fsc.Option {
	return WithIssuerIdentityWithHSM("_default_")
}

func WithDefaultOwnerIdentity() fsc.Option {
	return WithOwnerIdentity("_default_")
}

func WithOwnerIdentity(label string) fsc.Option {
	return func(o *fsc.Options) error {
		to := topology.ToOptions(o)
		to.SetOwners(append(to.Owners(), label))

		if label != "_default_" {
			o.AddAlias(label)
		}
		return nil
	}
}

// WithRemoteOwnerIdentity adds a new owner identity and mark it as remote.
func WithRemoteOwnerIdentity(label string) fsc.Option {
	return func(o *fsc.Options) error {
		to := topology.ToOptions(o)
		to.SetOwners(append(to.Owners(), label))
		to.SetRemoteOwner(label)

		if label != "_default_" {
			o.AddAlias(label)
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

func WithAuditorIdentityWithHSM() fsc.Option {
	return func(o *fsc.Options) error {
		topology.ToOptions(o).SetAuditor(true)
		topology.ToOptions(o).UseHSMForAuditor()

		return nil
	}
}
