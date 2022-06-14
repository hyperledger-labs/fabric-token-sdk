/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

type GetEnrollmentID struct {
	Wallet string `json:"wallet"`
}

// GetEnrollmentIDView is a view that returns the enrollment ID of a wallet.
type GetEnrollmentIDView struct {
	*GetEnrollmentID
}

func (r *GetEnrollmentIDView) Call(context view.Context) (interface{}, error) {
	w := ttx.GetWallet(context, r.Wallet)
	assert.NotNil(w, "wallet not found [%s]", r.Wallet)

	return w.EnrollmentID(), nil
}

type GetEnrollmentIDViewFactory struct{}

func (p *GetEnrollmentIDViewFactory) NewView(in []byte) (view.View, error) {
	f := &GetEnrollmentIDView{GetEnrollmentID: &GetEnrollmentID{}}
	err := json.Unmarshal(in, f.GetEnrollmentID)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
