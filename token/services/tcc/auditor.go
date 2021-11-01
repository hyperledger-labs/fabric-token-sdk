/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tcc

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
)

type RegisterAuditorView struct {
	TMSID token.TMSID
	Id    view.Identity
}

func NewRegisterAuditorView(tmsID token.TMSID, id view.Identity) *RegisterAuditorView {
	return &RegisterAuditorView{TMSID: tmsID, Id: id}
}

func (r *RegisterAuditorView) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(
		context,
		token.WithTMSID(r.TMSID),
	)

	var set bool
	key := fmt.Sprintf("token-sdk.%s.%s.%s.tcc.auditor.registered", tms.Network(), tms.Channel(), tms.Namespace())
	if kvs.GetService(context).Exists(key) {
		if err := kvs.GetService(context).Get(key, &set); err != nil {
			logger.Errorf("failed checking auditor has been registered to the chaincode [%s]", err)
			set = false
		}
	}

	if !set {
		err := network.GetInstance(
			context,
			tms.Network(),
			tms.Channel(),
		).RegisterAuditor(
			context,
			tms.Namespace(),
			r.Id,
		)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed auditor registration")
		}

		if err := kvs.GetService(context).Put(key, true); err != nil {
			logger.Errorf("failed recording auditor has been registered to the chaincode [%s]", err)
		}

		if err := tms.PublicParametersManager().ForceFetch(); err != nil {
			logger.Warnf("failed fetching parameters [%s]", err)
		}
	}
	return nil, nil
}
