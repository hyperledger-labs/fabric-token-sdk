/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
)

type ResponderRegistry interface {
	RegisterResponder(responder view.View, initiatedBy interface{}) error
}

func InstallViews(viewRegistry ResponderRegistry) error {
	logger.Debugf("Installing custodian views...")
	if err := viewRegistry.RegisterResponder(&PublicParamsRequestResponderView{}, &PublicParamsRequestView{}); err != nil {
		return err
	}
	if err := viewRegistry.RegisterResponder(&RequestApprovalResponderView{}, &RequestApprovalView{}); err != nil {
		return err
	}
	if err := viewRegistry.RegisterResponder(&BroadcastResponderView{}, &BroadcastView{}); err != nil {
		return err
	}
	if err := viewRegistry.RegisterResponder(&LookupKeyRequestRespondView{}, &LookupKeyRequestView{}); err != nil {
		return err
	}
	if err := viewRegistry.RegisterResponder(&RequestTxStatusResponderView{}, &RequestTxStatusView{}); err != nil {
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
