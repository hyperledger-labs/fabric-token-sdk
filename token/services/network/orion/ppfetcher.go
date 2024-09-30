/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"context"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
)

func NewCustodianPublicParamsFetcher(viewManager *view2.Manager) *custodianPublicParamsFetcher {
	return &custodianPublicParamsFetcher{viewManager: viewManager}
}

type custodianPublicParamsFetcher struct {
	viewManager *view2.Manager
}

func (f *custodianPublicParamsFetcher) Fetch(network driver2.Network, _ driver2.Channel, namespace driver2.Namespace) ([]byte, error) {
	pp, err := f.viewManager.InitiateView(NewPublicParamsRequestView(network, namespace), context.TODO())
	if err != nil {
		return nil, err
	}
	return pp.([]byte), nil
}
