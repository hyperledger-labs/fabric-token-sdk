//go:build tools
// +build tools

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tools

import (
	_ "github.com/hyperledger-labs/fabric-smart-client"
	_ "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/comm"
	_ "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing/optl"
)
