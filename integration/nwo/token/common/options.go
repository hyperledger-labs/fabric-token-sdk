/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"

// WithOnlyUnity notify the backend to use the unity driver for all databases
func WithOnlyUnity(tms *topology.TMS) {
	tms.BackendParams["OnlyUnity"] = true
}

// IsOnlyUnity return true if this TMS requires to use the unity driver for all databases
func IsOnlyUnity(tms *topology.TMS) bool {
	boxed, ok := tms.BackendParams["OnlyUnity"]
	if ok {
		return boxed.(bool)
	}
	return false
}
