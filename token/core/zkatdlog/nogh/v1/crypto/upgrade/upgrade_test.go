/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package upgrade

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokensService_NewUpgradeChallenge(t *testing.T) {
	ts := NewService()
	challenge, err := ts.NewUpgradeChallenge()
	assert.NoError(t, err)
	assert.Len(t, challenge, ChallengeSize)
}

func TestTokensService_GenUpgradeProof(t *testing.T) {
	ts := NewService()
	res, err := ts.GenUpgradeProof(nil, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, res)
	assert.EqualError(t, err, "not supported")
}

func TestTokensService_CheckUpgradeProof(t *testing.T) {
	ts := NewService()
	res, err := ts.CheckUpgradeProof(nil, nil, nil)
	assert.Error(t, err)
	assert.False(t, res)
	assert.EqualError(t, err, "not supported")
}
