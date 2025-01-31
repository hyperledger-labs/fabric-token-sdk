/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/api"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/dlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/config"
	. "github.com/onsi/gomega"
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
	SDKs                []api.SDK
	WebEnabled          bool
	Monitoring          bool
	TokenSelector       string
	OnlyUnity           bool
	FSCBasedEndorsement bool
	ExtraTMSs           []TMSOpts
	FinalityType        config.ManagerType
}

func SetDefaultParams(tms *topology.TMS, opts TMSOpts) {
	switch opts.TokenSDKDriver {
	case "dlog":
		if opts.Aries {
			dlog.WithAries(tms)
		}
	case "fabtoken":
		// no nothig
	default:
		Expect(false).To(BeTrue(), "expected token driver in (dlog,fabtoken), got [%s]", opts.TokenSDKDriver)
	}
	if len(opts.PublicParamsGenArgs) != 0 {
		tms.SetTokenGenPublicParams(opts.PublicParamsGenArgs...)
	} else {
		// max token value is 2^16
		tms.SetTokenGenPublicParams("16")
	}
}
