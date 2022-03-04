/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"

func SetOrgs(tms *topology.TMS, orgs ...string) *topology.TMS {
	tms.BackendParams["fabric.orgs"] = orgs
	return tms
}
