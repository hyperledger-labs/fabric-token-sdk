/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pp

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/pp"
	"github.com/stretchr/testify/assert"
)

func TestSerialization(t *testing.T) {
	// Marshal

	// check failures
	_, err := Marshal(nil)
	assert.Error(t, err)
	assert.EqualError(t, err, "nil public parameters")

	// success
	pp := &pp.PublicParameters{
		Identifier: "pineapple",
		Raw:        []byte{1, 2, 3},
	}
	res, err := Marshal(pp)
	assert.NoError(t, err)

	// Unmarshall

	// success
	pp2, err := Unmarshal(res)
	assert.NoError(t, err)
	assert.Equal(t, pp, pp2)

	// failure
	_, err = Unmarshal([]byte{})
	assert.Error(t, err)

	_, err = Unmarshal(nil)
	assert.Error(t, err)

	_, err = Unmarshal([]byte{1, 2, 3})
	assert.Error(t, err)
}
