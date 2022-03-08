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
	view.GetRegistry(sp).RegisterResponder(&RespondPublicParamsRequestView{}, &PublicParamsRequestView{})

	return nil
}
