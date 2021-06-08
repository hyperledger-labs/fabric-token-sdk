/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package query

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
)

func InstallQueryViewFactories(sp view.ServiceProvider) {
	view.GetRegistry(sp).RegisterFactory("zkat.balance.query", &BalanceViewFactory{})
	view.GetRegistry(sp).RegisterFactory("zkat.all.balance.query", &AllMyBalanceViewFactory{})
}
