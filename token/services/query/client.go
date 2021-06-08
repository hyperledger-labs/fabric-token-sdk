/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package query

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
)

type Client interface {
	CallView(fid string, in []byte) (interface{}, error)
}

type viewClient struct {
	vClient Client
}

func NewClient(vClient Client) *viewClient {
	return &viewClient{vClient: vClient}
}

func (c *viewClient) Balance(typ string) ([]Balance, error) {
	return c.WalletBalance("", typ)
}

func (c *viewClient) WalletBalance(wallet, typ string) ([]Balance, error) {
	balance, err := c.vClient.CallView("zkat.balance.query", common.JSONMarshall(&BalanceQuery{
		Wallet: wallet,
		Type:   typ,
	}))
	if err != nil {
		return nil, err
	}
	bal := Balance{}
	err = json.Unmarshal(balance.([]byte), &bal)
	if err != nil {
		return nil, errors.Errorf("could not retrieve balance")
	}
	return []Balance{bal}, nil
}

func (c *viewClient) AllMyBalances() ([]Balance, error) {
	balances, err := c.vClient.CallView("zkat.all.balance.query",
		common.JSONMarshall(&AllBalanceQuery{}),
	)
	if err != nil {
		return nil, err
	}
	bal := AllMyBalances{}
	err = json.Unmarshal(balances.([]byte), &bal)
	if err != nil {
		return nil, errors.Errorf("could not retrieve balance")
	}
	return bal.Balances, err
}
