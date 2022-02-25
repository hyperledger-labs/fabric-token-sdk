//go:build tools
// +build tools

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tools

import (
	_ "github.com/IBM/idemix/bccsp"
	_ "github.com/IBM/idemix/bccsp/keystore"
	_ "github.com/IBM/idemix/bccsp/schemes"
	_ "github.com/IBM/idemix/bccsp/schemes/dlog/crypto/translator/amcl"
	_ "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/tracing"
	_ "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/weaver/relay/fabric"
	_ "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/comm"
	_ "github.com/hyperledger/fabric/cmd/idemixgen"
	_ "github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	_ "github.com/hyperledger/fabric/common/metrics/prometheus"
	_ "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb/statecouchdb"
	_ "github.com/hyperledger/fabric/core/ledger/pvtdatastorage"
	_ "github.com/hyperledger/fabric/core/operations"
	_ "github.com/libp2p/go-libp2p-core/network"
	_ "github.com/maxbrunsfeld/counterfeiter/v6"
)
