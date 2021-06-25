/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"io/ioutil"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo"

	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/dvp"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/tcc/basic"
)

func main() {
	gomega.RegisterFailHandler(ginkgo.Fail)
	topologies := map[string][]nwo.Topology{}

	topologies["tcc_basic_fabtoken.yaml"] = basic.Topology("fabtoken")
	topologies["tcc_basic_dlog.yaml"] = basic.Topology("dlog")
	topologies["dvp_fabtoken.yaml"] = dvp.Topology("fabtoken")
	topologies["dvp_dlog.yaml"] = dvp.Topology("dlog")

	for name, topologies := range topologies {
		t := nwo.Topologies{Topologies: topologies}
		raw, err := t.Export()
		if err != nil {
			panic(err)
		}
		if err := ioutil.WriteFile(name, raw, 0770); err != nil {
			panic(err)
		}
	}
}
