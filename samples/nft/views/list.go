/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"fmt"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/nftcc"
)

// GetHouse contains the input to query a house by id
type GetHouse struct {
	HouseID string
}

type GetHouseView struct {
	*GetHouse
}

func (p *GetHouseView) Call(context view.Context) (interface{}, error) {
	house := &House{}
	if err := nftcc.MyWallet(context).QueryByKey(house, "LinearID", p.HouseID); err != nil {
		if err == nftcc.ErrNoResults {
			return fmt.Sprintf("no house found with id [%s]", p.HouseID), nil
		}
		return nil, err
	}
	return house, nil
}

type GetHouseViewFactory struct{}

func (i *GetHouseViewFactory) NewView(in []byte) (view.View, error) {
	f := &GetHouseView{GetHouse: &GetHouse{}}
	err := json.Unmarshal(in, f.GetHouse)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
