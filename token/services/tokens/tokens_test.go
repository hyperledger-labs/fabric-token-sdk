/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/test-go/testify/assert"
)

type qsMock struct{}

func (qs qsMock) IsMine(id *token2.ID) (bool, error) {
	return true, nil
}

type authMock struct{}

func (a authMock) Issued(issuer driver.Identity, tok *token2.Token) bool {
	return false
}
func (a authMock) IsMine(tok *token2.Token) (string, []string, bool) {
	return "", []string{string(tok.Owner)}, true
}
func (a authMock) AmIAnAuditor() bool {
	return false
}
func (a authMock) OwnerType(raw []byte) (string, []byte, error) {
	return "idemix", raw, nil
}

type mdMock struct{}

func (md mdMock) SpentTokenID() []*token2.ID {
	// only called if graphHiding is true
	return []*token2.ID{}
}

func TestParse(t *testing.T) {
	tokens := &Tokens{
		TMSProvider: nil,
		Storage:     &DBStorage{},
	}
	md := mdMock{}

	// simple transfer
	input1 := &token.Input{
		Id: &token2.ID{
			TxId:  "in",
			Index: 0,
		},
		ActionIndex:  0,
		EnrollmentID: "alice",
		Type:         "TOK",
		Quantity:     token2.NewQuantityFromUInt64(10),
	}
	output1 := &token.Output{
		ActionIndex:  0,
		Index:        0,
		EnrollmentID: "bob",
		Type:         "TOK",
		LedgerOutput: []byte("bob,TOK,0x0"),
		Quantity:     token2.NewQuantityFromUInt64(10),
	}
	is := token.NewInputStream(qsMock{}, []*token.Input{input1}, 64)
	os := token.NewOutputStream([]*token.Output{output1}, 64)

	spend, store := tokens.parse(&authMock{}, "tx1", nil, md, is, os, false, 64, false)

	assert.Len(t, spend, 1)
	assert.Equal(t, "in", spend[0].TxId)
	assert.Equal(t, uint64(0), spend[0].Index)

	assert.Len(t, store, 1)
	assert.Equal(t, "tx1", store[0].txID)
	assert.Equal(t, output1.Index, store[0].index)
	assert.Equal(t, output1.LedgerOutput, store[0].tokenOnLedger)
	assert.Equal(t, true, store[0].flags.Mine)
	assert.Equal(t, false, store[0].flags.Auditor)
	assert.Equal(t, false, store[0].flags.Issuer)
	assert.Equal(t, uint64(64), store[0].precision)
	assert.Equal(t, output1.Type, store[0].tok.Type)

	// no ledger output -> spend
	output1.LedgerOutput = []byte{}
	os = token.NewOutputStream([]*token.Output{output1}, 64)
	spend, store = tokens.parse(&authMock{}, "tx1", nil, md, is, os, false, 64, false)
	assert.Len(t, spend, 2)
	assert.Len(t, store, 0)

	// transfer with several inputs and outputs
	input1 = &token.Input{
		Id: &token2.ID{
			TxId:  "in1",
			Index: 1,
		},
		ActionIndex:  0,
		EnrollmentID: "alice",
		Type:         "TOK",
		Quantity:     token2.NewQuantityFromUInt64(50),
	}
	input2 := &token.Input{
		Id: &token2.ID{
			TxId:  "in2",
			Index: 2,
		},
		ActionIndex:  0,
		EnrollmentID: "alice",
		Type:         "TOK",
		Quantity:     token2.NewQuantityFromUInt64(50),
	}
	output1 = &token.Output{
		ActionIndex:  0,
		Index:        0,
		EnrollmentID: "bob",
		Type:         "TOK",
		LedgerOutput: []byte("bob,TOK,0x0"),
		Quantity:     token2.NewQuantityFromUInt64(10),
	}
	output2 := &token.Output{
		ActionIndex:  0,
		Index:        1,
		EnrollmentID: "alice",
		Type:         "TOK",
		LedgerOutput: []byte("bob,TOK,0x0"),
		Quantity:     token2.NewQuantityFromUInt64(90),
	}
	is = token.NewInputStream(qsMock{}, []*token.Input{input1, input2}, 64)
	os = token.NewOutputStream([]*token.Output{output1, output2}, 64)

	spend, store = tokens.parse(&authMock{}, "tx2", nil, md, is, os, false, 64, false)
	assert.Len(t, spend, 2)
	assert.Equal(t, "in1", spend[0].TxId)
	assert.Equal(t, uint64(1), spend[0].Index)
	assert.Equal(t, "in2", spend[1].TxId)
	assert.Equal(t, uint64(2), spend[1].Index)

	assert.Len(t, store, 2)
	assert.Equal(t, output1.LedgerOutput, store[0].tokenOnLedger)
	assert.Equal(t, "tx2", store[0].txID)
	assert.Equal(t, output1.Index, store[0].index)
	assert.Equal(t, output1.Type, store[0].tok.Type)

	assert.Equal(t, output2.LedgerOutput, store[1].tokenOnLedger)
	assert.Equal(t, "tx2", store[1].txID)
	assert.Equal(t, output2.Index, store[1].index)
	assert.Equal(t, output2.Type, store[1].tok.Type)
}
