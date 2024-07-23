/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type ResponderRegistry interface {
	RegisterResponder(responder view2.View, initiatedBy interface{}) error
}

func InstallViews(viewRegistry ResponderRegistry, dbManager *DBManager, statusCache TxStatusResponseCache) error {
	logger.Debugf("Installing custodian views...")
	if err := viewRegistry.RegisterResponder(&PublicParamsRequestResponderView{}, &PublicParamsRequestView{}); err != nil {
		return err
	}
	if err := viewRegistry.RegisterResponder(&RequestApprovalResponderView{dbManager: dbManager}, &RequestApprovalView{}); err != nil {
		return err
	}
	if err := viewRegistry.RegisterResponder(&BroadcastResponderView{dbManager: dbManager}, &BroadcastView{}); err != nil {
		return err
	}
	if err := viewRegistry.RegisterResponder(&LookupKeyRequestRespondView{}, &LookupKeyRequestView{}); err != nil {
		return err
	}
	if err := viewRegistry.RegisterResponder(&RequestTxStatusResponderView{dbManager: dbManager, statusCache: statusCache}, &RequestTxStatusView{}); err != nil {
		return err
	}
	if err := viewRegistry.RegisterResponder(&RequestSpentTokensResponderView{}, &RequestSpentTokensView{}); err != nil {
		return err
	}
	if err := viewRegistry.RegisterResponder(&RequestQueryTokensResponderView{}, &RequestQueryTokensView{}); err != nil {
		return err
	}

	return nil
}
