/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package query

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
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

func (c *viewClient) Balance(typ string, opts ...token.ServiceOption) ([]Balance, error) {
	return c.WalletBalance("", typ, opts...)
}

func (c *viewClient) WalletBalance(wallet, typ string, opts ...token.ServiceOption) ([]Balance, error) {
	options, err := token.CompileServiceOptions(opts...)
	if err != nil {
		return nil, err
	}

	balance, err := c.vClient.CallView("zkat.balance.query", common.JSONMarshall(&BalanceQuery{
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

func (c *viewClient) AllMyBalances(opts ...token.ServiceOption) ([]Balance, error) {
	options, err := token.CompileServiceOptions(opts...)
	if err != nil {
		return nil, err
	}

	balances, err := c.vClient.CallView("zkat.all.balance.query",
		common.JSONMarshall(&AllBalanceQuery{
			TMSID: token.TMSID{
				Network:   options.Network,
				Channel:   options.Channel,
				Namespace: options.Namespace,
			},
		}),
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
