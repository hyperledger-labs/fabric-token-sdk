/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nftcc

import (
	"encoding/base64"
	"encoding/json"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/thedevsaddam/gojsonq"
	"testing"
)

type House struct {
	LinearID  string
	Address   string
	Valuation uint64
}

func TestJsonFilter(t *testing.T) {
	h := &House{
		LinearID:  "hello world",
		Address:   "5th Avenue",
		Valuation: 100,
	}
	raw, err := json.Marshal(h)
	assert.NoError(t, err, "json marshal failed")
	tok := &token2.UnspentToken{
		Type: base64.StdEncoding.EncodeToString(raw),
	}

	// hit
	f := &jsonFilter{
		q:     gojsonq.New(),
		key:   "LinearID",
		value: "hello world",
	}
	assert.True(t, f.ContainsToken(tok))

	// no hit
	f = &jsonFilter{
		q:     gojsonq.New(),
		key:   "LinearID",
		value: "pineapple",
	}
	assert.False(t, f.ContainsToken(tok))
}
