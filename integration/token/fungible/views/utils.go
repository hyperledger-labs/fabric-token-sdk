/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"strings"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

// AssertTokens checks that the tokens are or are not in the tokendb
func AssertTokens(sp token.ServiceProvider, tx *ttx.Transaction, outputs *token.OutputStream, id view.Identity) {
	db, err := tokendb.GetByTMSId(sp, tx.TokenService().ID())
	assert.NoError(err, "failed to get token db for [%s]", tx.TokenService().ID())
	for _, output := range outputs.Outputs() {
		tokenID := output.ID(tx.ID())
		if output.Owner.Equal(id) || tx.TokenService().SigService().IsMe(output.Owner) {
			// check it exists
			_, toks, err := db.GetTokens(tokenID)
			assert.NoError(err, "failed to retrieve token [%s]", tokenID)
			assert.Equal(1, len(toks), "expected one token")
			assert.Equal(output.Quantity.Hex(), toks[0].Quantity, "token quantity mismatch")
			assert.Equal(output.Type, toks[0].Type, "token type mismatch")
		} else {
			// check it does not exist
			_, _, err := db.GetTokens(tokenID)
			assert.Error(err, "token [%s] should not exist", tokenID)
			assert.True(strings.Contains(err.Error(), "token not found"))
		}
	}
}

type KVSEntry struct {
	Key   string
	Value string
}

type SetKVSEntryView struct {
	*KVSEntry
}

func (s *SetKVSEntryView) Call(context view.Context) (interface{}, error) {
	assert.NoError(GetKVS(context).Put(s.Key, s.Value), "failed to put in KVS [%s:%s]", s.Key, s.Value)
	return nil, nil
}

type SetKVSEntryViewFactory struct{}

func (p *SetKVSEntryViewFactory) NewView(in []byte) (view.View, error) {
	f := &SetKVSEntryView{KVSEntry: &KVSEntry{}}
	err := json.Unmarshal(in, f.KVSEntry)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

// ServiceOpts creates a new array of token.ServiceOption containing token.WithTMSID and any additional token.ServiceOption passed to this function
func ServiceOpts(tmsId *token.TMSID, opts ...token.ServiceOption) []token.ServiceOption {
	var serviceOpts []token.ServiceOption
	if tmsId != nil {
		serviceOpts = append(serviceOpts, token.WithTMSID(*tmsId))
	}
	return append(serviceOpts, opts...)
}

func TxOpts(tmsId *token.TMSID) []ttx.TxOption {
	var txOpts []ttx.TxOption
	if tmsId != nil {
		txOpts = append(txOpts, ttx.WithTMSID(*tmsId))
	}
	return txOpts
}

func GetKVS(sp view2.ServiceProvider) *kvs.KVS {
	kvss, err := sp.GetService(&kvs.KVS{})
	if err != nil {
		panic(err)
	}
	return kvss.(*kvs.KVS)
}
