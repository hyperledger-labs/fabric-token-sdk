/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	fsc "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
)

func SetCustodian(tms *topology.TMS, custodian *fsc.Node) *topology.TMS {
	tms.BackendParams["orion.custodian"] = custodian
	return tms
}
