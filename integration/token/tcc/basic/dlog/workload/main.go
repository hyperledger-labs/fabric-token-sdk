package main

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/integration"
	. "github.com/onsi/gomega"

	fscintegration "github.com/hyperledger-labs/fabric-token-sdk/integration"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/tcc/basic"
)

func main() {
	var err error
	network, err := integration.New(fscintegration.ZKATDLogBasics.StartPortForNode(), "./testdata", basic.Topology("dlog", true)...)
	Expect(err).NotTo(HaveOccurred())
	network.RegisterPlatformFactory(token.NewPlatformFactory())
	network.Generate()
	network.Start()
	defer network.Stop()
	basic.TestWorkload(network)
	time.Sleep(60 * time.Minute)
}
