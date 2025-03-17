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
	assert.NoError(t, err)
	assert.Nil(t, res)
}

func TestTokensService_CheckUpgradeProof(t *testing.T) {
	ts := NewService()
	res, err := ts.CheckUpgradeProof(nil, nil, nil)
	assert.NoError(t, err)
	assert.True(t, res)
}
