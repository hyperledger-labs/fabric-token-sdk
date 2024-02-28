/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
)

func InstallViews(sp view.ServiceProvider) error {
	logger.Debugf("Installing custodian views...")
	if err := view.GetRegistry(sp).RegisterResponder(&PublicParamsRequestResponderView{}, &PublicParamsRequestView{}); err != nil {
		return err
	}
	if err := view.GetRegistry(sp).RegisterResponder(&RequestApprovalResponderView{}, &RequestApprovalView{}); err != nil {
		return err
	}
	if err := view.GetRegistry(sp).RegisterResponder(&BroadcastResponderView{}, &BroadcastView{}); err != nil {
		return err
	}
	if err := view.GetRegistry(sp).RegisterResponder(&LookupKeyRequestRespondView{}, &LookupKeyRequestView{}); err != nil {
		return err
	}
	if err := view.GetRegistry(sp).RegisterResponder(&RequestTxStatusResponderView{}, &RequestTxStatusView{}); err != nil {
		return err
	}
	if err := view.GetRegistry(sp).RegisterResponder(&RequestSpentTokensResponderView{}, &RequestSpentTokensView{}); err != nil {
		return err
	}
	if err := view.GetRegistry(sp).RegisterResponder(&RequestQueryTokensResponderView{}, &RequestQueryTokensView{}); err != nil {
		return err
	}

	return nil
}
