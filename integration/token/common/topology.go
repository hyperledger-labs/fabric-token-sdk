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
	. "github.com/onsi/gomega"
)

type replicationOpts interface {
	For(name string) []node.Option
}

type TMSOpts struct {
	Alias               string
	TokenSDKDriver      string
	PublicParamsGenArgs []string
	Aries               bool
}

type Opts struct {
	CommType            fsc.P2PCommunicationType
	ReplicationOpts     replicationOpts
	Backend             string
	TokenSDKDriver      string
	AuditorAsIssuer     bool
	Aries               bool
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
}

func SetDefaultParams(tokenSDKDriver string, tms *topology.TMS, aries bool) {
	switch tokenSDKDriver {
	case "dlog":
		if aries {
			dlog.WithAries(tms)
		}
		// max token value is 2^16
		tms.SetTokenGenPublicParams("16")
	case "fabtoken":
		// max token value is 2^16
		tms.SetTokenGenPublicParams("16")
	default:
		Expect(false).To(BeTrue(), "expected token driver in (dlog,fabtoken), got [%s]", tokenSDKDriver)
	}
}
