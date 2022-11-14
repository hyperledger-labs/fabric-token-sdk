/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package query

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/pkg/errors"
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

type BalanceQuery struct {
	TMSID  token.TMSID
	Wallet string
	Type   string
}

type Balance struct {
	Type     string
	Quantity string
}

func (c *viewClient) WalletBalance(wallet, typ string, opts ...token.ServiceOption) ([]Balance, error) {
	options, err := token.CompileServiceOptions(opts...)
	if err != nil {
		return nil, err
	}

	balance, err := c.vClient.CallView("balance", common.JSONMarshall(&BalanceQuery{
		TMSID: token.TMSID{
			Network:   options.Network,
			Channel:   options.Channel,
			Namespace: options.Namespace,
		},
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
