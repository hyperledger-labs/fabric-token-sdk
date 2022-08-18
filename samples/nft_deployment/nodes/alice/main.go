/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	fscnode "github.com/hyperledger-labs/fabric-smart-client/node"
	fabric "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
	viewregistry "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	nftsample "github.com/hyperledger-labs/fabric-token-sdk/samples/nft/views"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
)

func main() {
	n := fscnode.New()
	if err := n.InstallSDK(fabric.NewSDK(n)); err != nil {
		panic(err)
	}
	if err := n.InstallSDK(tokensdk.NewSDK(n)); err != nil {
		panic(err)
	}

	n.Execute(func() error {
		registry := viewregistry.GetRegistry(n)
		if err := registry.RegisterFactory("transfer", &nftsample.TransferHouseViewFactory{}); err != nil {
			return err
		}
		if err := registry.RegisterFactory("queryHouse", &nftsample.GetHouseViewFactory{}); err != nil {
			return err
		}
		if err := registry.RegisterResponder(&nftsample.AcceptIssuedHouseView{}, &nftsample.IssueHouseView{}); err != nil {
			return err
		}
		if err := registry.RegisterResponder(&nftsample.AcceptTransferHouseView{}, &nftsample.TransferHouseView{}); err != nil {
			return err
		}

		return nil
	})
}
