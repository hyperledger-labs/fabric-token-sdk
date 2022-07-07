/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	fscnode "github.com/hyperledger-labs/fabric-smart-client/node"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"

	viewregistry "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	views "github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop/views"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
)

func main() {
	n := fscnode.New()
	n.InstallSDK(fabric.NewSDK(n))
	n.InstallSDK(sdk.NewSDK(n))
	
	n.Execute(func() error {
		registry := viewregistry.GetRegistry(n)
		if err := registry.RegisterFactory("issue", &views.IssueCashViewFactory{}); err != nil {
			return err
		}
		
		return nil
	})
}
