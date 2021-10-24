/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	fabric3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	approver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/approver"
)

type approver struct {
	sp view2.ServiceProvider
}

func NewApprover(sp view2.ServiceProvider) *approver {
	return &approver{sp: sp}
}

func (a approver) Validate(tx *Transaction) error {
	logger.Debugf("start appriving process for tx [%s]", tx.ID())

	rws, err := tx.RWSet()
	if err != nil {
		return errors.WithMessage(err, "failed getting rws")
	}

	ts := tx.TokenService()
	app := approver2.NewTokenRWSetApprover(
		ts.Validator(),
		fabric3.NewVault(fabric.GetChannel(tx.tx.ServiceProvider, tx.Network(), tx.Channel())),
		tx.ID(),
		fabric3.NewRWSWrapper(rws),
		ts.Namespace(),
	)

	return app.Validate(func(id view.Identity, verifier approver2.Verifier) error {
		return tx.HasBeenEndorsedBy(id)
	}, tx.TokenRequest)
}
