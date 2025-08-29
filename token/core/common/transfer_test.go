/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/stretchr/testify/assert"
)

func TestSelectIssuerForRedeem(t *testing.T) {
	type testCase struct {
		name           string
		issuers        []driver.Identity
		attributes     map[interface{}]interface{}
		expectError    bool
		expectIdentity driver.Identity
	}

	issuerFSCIdentity := view.Identity("fsc-identity")
	issuerPPPublicKey := view.Identity("pp-public-key")

	testCases := []testCase{
		{
			name:    "opts with FSC issuer identity and public key",
			issuers: nil,
			attributes: map[interface{}]interface{}{
				ttx.IssuerFSCIdentityKey:        issuerFSCIdentity,
				ttx.IssuerPublicParamsPublicKey: issuerPPPublicKey,
			},
			expectError:    false,
			expectIdentity: nil,
		},
		{
			name:    "opts with FSC issuer identity but no public key",
			issuers: nil,
			attributes: map[interface{}]interface{}{
				ttx.IssuerFSCIdentityKey: issuerFSCIdentity,
			},
			expectError:    false,
			expectIdentity: nil,
		},
		{
			name:           "opts with no FSC issuer identity, issuers present",
			issuers:        []driver.Identity{issuerFSCIdentity, issuerPPPublicKey},
			attributes:     map[interface{}]interface{}{},
			expectError:    false,
			expectIdentity: issuerFSCIdentity,
		},
		{
			name:           "opts with no FSC issuer identity, no issuers",
			issuers:        []driver.Identity{},
			attributes:     map[interface{}]interface{}{},
			expectError:    false,
			expectIdentity: nil,
		},
		{
			name:    "IssuerFSCIdentityKey is not a view Identity",
			issuers: nil,
			attributes: map[interface{}]interface{}{
				ttx.IssuerFSCIdentityKey: 1234,
			},
			expectError:    false,
			expectIdentity: nil,
		},
		{
			name:    "IssuerPublicParamsPublicKey is not a view Identity",
			issuers: nil,
			attributes: map[interface{}]interface{}{
				ttx.IssuerFSCIdentityKey:        issuerFSCIdentity,
				ttx.IssuerPublicParamsPublicKey: 1234,
			},
			expectError:    false,
			expectIdentity: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := SelectIssuerForRedeem(tc.issuers, &driver.TransferOptions{Attributes: tc.attributes})
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, id)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectIdentity, id)
			}
		})
	}
}
