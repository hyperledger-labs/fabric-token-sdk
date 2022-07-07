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
	exchange "github.com/hyperledger-labs/fabric-token-sdk/integration/token/interop/views/exchange"
	sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
)

func main() {
	n := fscnode.New()
	n.InstallSDK(fabric.NewSDK(n))
	n.InstallSDK(sdk.NewSDK(n))
	
	n.Execute(func() error {
		registry := viewregistry.GetRegistry(n)
		if err := registry.RegisterFactory("exchange.lock", &exchange.LockViewFactory{}); err != nil {
			return err
		}
		if err := registry.RegisterFactory("exchange.reclaimAll", &exchange.ReclaimAllViewFactory{}); err != nil {
			return err
		}
		if err := registry.RegisterFactory("exchange.claim", &exchange.ClaimViewFactory{}); err != nil {
			return err
		}
		registry.RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{})
		registry.RegisterResponder(&exchange.LockAcceptView{}, &exchange.LockView{})
		
		return nil
	})
}
