/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"context"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/chaincode"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/view"
)

type chaincodePublicParamsFetcher struct {
	viewManager ViewManager
}

func NewChaincodePublicParamsFetcher(viewManager *view.Manager) *chaincodePublicParamsFetcher {
	return &chaincodePublicParamsFetcher{viewManager: viewManager}
}

func (f *chaincodePublicParamsFetcher) Fetch(network driver2.Network, channel driver2.Channel, namespace driver2.Namespace) ([]byte, error) {
	ppBoxed, err := f.viewManager.InitiateView(
		chaincode.NewQueryView(
			namespace,
			QueryPublicParamsFunction,
		).WithNetwork(network).WithChannel(channel),
		context.Background(),
	)
	if err != nil {
		return nil, err
	}

	return ppBoxed.([]byte), nil
}
