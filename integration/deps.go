//go:build tools
// +build tools

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package integration

// these imports are necessary to create the go.sum entries for transitive dependencies
// of generated code.
import (
	_ "github.com/hyperledger-labs/fabric-smart-client"
	_ "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/comm"
	_ "github.com/libp2p/go-libp2p"
)
