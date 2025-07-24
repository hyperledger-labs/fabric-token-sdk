/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	nodepkg "github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/fabtokenv1"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/zkatdlognoghv1"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/config"
	"github.com/onsi/gomega"
)

type replicationOpts interface {
	For(name string) []node.Option
}

type TMSOpts struct {
	Alias               topology.TMSAlias
	TokenSDKDriver      string
	PublicParamsGenArgs []string
	Aries               bool
}

type Opts struct {
	CommType            fsc.P2PCommunicationType
	ReplicationOpts     replicationOpts
	Backend             string
	DefaultTMSOpts      TMSOpts
	AuditorAsIssuer     bool
	FSCLogSpec          string
	NoAuditor           bool
	HSM                 bool
	SDKs                []nodepkg.SDK
	WebEnabled          bool
	Monitoring          bool
	TokenSelector       string
	FSCBasedEndorsement bool
	ExtraTMSs           []TMSOpts
	FinalityType        config.ManagerType
}

func SetDefaultParams(tms *topology.TMS, opts TMSOpts) {
	switch opts.TokenSDKDriver {
	case zkatdlognoghv1.DriverIdentifier:
		if opts.Aries {
			zkatdlognoghv1.WithAries(tms)
		}
	case fabtokenv1.DriverIdentifier:
		// no nothig
	default:
		gomega.Expect(false).To(gomega.BeTrue(), "expected token driver in (dlog,fabtoken), got [%s]", opts.TokenSDKDriver)
	}
	if len(opts.PublicParamsGenArgs) != 0 {
		tms.SetTokenGenPublicParams(opts.PublicParamsGenArgs...)
	} else {
		// max token value is 2^16
		tms.SetTokenGenPublicParams("16")
	}
}
