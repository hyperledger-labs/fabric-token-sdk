//go:build deps

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package deps

import (
	_ "github.com/dgraph-io/badger/v3"
	_ "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/comm/host/rest"
	_ "github.com/libp2p/go-libp2p-kad-dht"
	_ "github.com/ugorji/go"
)
