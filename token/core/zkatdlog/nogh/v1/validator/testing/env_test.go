/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package testing

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSaveTransferToFile(t *testing.T) {
	e := &Env{
		TRWithTransferTxID: "tx123",
		TRWithTransferRaw:  []byte{1, 2, 3, 4, 5},
	}
	path := filepath.Join(t.TempDir(), "transfer.json")
	err := e.SaveTransferToFile(path)
	assert.NoError(t, err)

	b, err := os.ReadFile(path)
	assert.NoError(t, err)

	var payload struct {
		TxID   string `json:"txid"`
		ReqRaw string `json:"req_raw"`
	}
	err = json.Unmarshal(b, &payload)
	assert.NoError(t, err)
	assert.Equal(t, e.TRWithTransferTxID, payload.TxID)

	decoded, err := base64.StdEncoding.DecodeString(payload.ReqRaw)
	assert.NoError(t, err)
	assert.Equal(t, e.TRWithTransferRaw, decoded)
}
