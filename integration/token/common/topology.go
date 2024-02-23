/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/dlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	. "github.com/onsi/gomega"
)

func SetDefaultParams(tokenSDKDriver string, tms *topology.TMS, aries bool) {
	switch tokenSDKDriver {
	case "dlog":
		if aries {
			dlog.WithAries(tms)
		}
		// max token value is 2^16
		tms.SetTokenGenPublicParams("16")
	case "fabtoken":
		tms.SetTokenGenPublicParams("65535")
	default:
		Expect(false).To(BeTrue(), "expected token driver in (dlog,fabtoken), got [%s]", tokenSDKDriver)
	}
}
