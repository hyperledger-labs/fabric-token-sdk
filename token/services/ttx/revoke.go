/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/pkg/errors"
)

type AuditorRevocationView struct {
	revocationHandle string
}

func RevokeView(revocationHandle string) *AuditorRevocationView {
	return &AuditorRevocationView{revocationHandle: revocationHandle}
}

func (r *AuditorRevocationView) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(context)
	err := tms.WalletManager().UpdateRevocationList(r.revocationHandle)
	if err != nil {
		logger.Errorf("failed to get revocation handler list")
		return nil, errors.WithMessagef(err, "failed to get revocation handler list")
	}

	return nil, nil
}
