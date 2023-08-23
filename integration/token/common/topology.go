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

func SetDefaultParams(tokenSDKDriver string, tms *topology.TMS) {
	switch tokenSDKDriver {
	case "dlog":
		dlog.WithAries(tms)
		// max token value is 100^2 - 1 = 9999
		tms.SetTokenGenPublicParams("100", "2")
	case "fabtoken":
		tms.SetTokenGenPublicParams("9999")
	default:
		Expect(false).To(BeTrue(), "expected token driver in (dlog,fabtoken), got [%s]", tokenSDKDriver)
	}
}
