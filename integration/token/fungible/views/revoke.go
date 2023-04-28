/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"strconv"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type UpdateRevocationList struct {
	RH string
}

type UpdateRevocationListView struct {
	*UpdateRevocationList
}

type UpdateRevocationListViewFactory struct{}

func (u UpdateRevocationListView) Call(context view.Context) (interface{}, error) {
	kvsInstance := kvs.GetService(context)
	k := kvs.CreateCompositeKeyOrPanic("revocationList", []string{strconv.QuoteToASCII(u.RH)})
	assert.False(kvsInstance.Exists(k), "Identity already in revoked state")
	assert.NoError(kvsInstance.Put(k, u.RH), "failed to put revocation handle")
	return nil, nil
}

func (u *UpdateRevocationListViewFactory) NewView(in []byte) (view.View, error) {
	f := &UpdateRevocationListView{UpdateRevocationList: &UpdateRevocationList{}}
	err := json.Unmarshal(in, f.UpdateRevocationList)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
