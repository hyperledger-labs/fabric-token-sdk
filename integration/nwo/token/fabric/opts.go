/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"

// WithFabricCA notify the backend to activate fabric-ca for the issuance of identities
func WithFabricCA(tms *topology.TMS) {
	tms.BackendParams["fabricca"] = true
}

// IsFabricCA return true if this TMS requires to enable Fabric-CA
func IsFabricCA(tms *topology.TMS) bool {
	boxed, ok := tms.BackendParams["fabricca"]
	if ok {
		return boxed.(bool)
	}
	return false
}
