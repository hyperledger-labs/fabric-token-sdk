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
	"github.com/stretchr/testify/require"
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
	dummyIssuer := view.Identity("dummy-issuer")

	testCases := []testCase{
		{
			name:    "opts with FSC issuer identity and public key",
			issuers: []driver.Identity{dummyIssuer},
			attributes: map[interface{}]interface{}{
				ttx.IssuerFSCIdentityKey:        issuerFSCIdentity,
				ttx.IssuerPublicParamsPublicKey: issuerPPPublicKey,
			},
			expectError:    false,
			expectIdentity: issuerPPPublicKey,
		},
		{
			name:    "opts with FSC issuer identity but no public key",
			issuers: []driver.Identity{dummyIssuer},
			attributes: map[interface{}]interface{}{
				ttx.IssuerFSCIdentityKey: issuerFSCIdentity,
			},
			expectError:    true,
			expectIdentity: nil,
		},
		{
			name:           "opts with no FSC issuer identity, issuers present",
			issuers:        []driver.Identity{dummyIssuer, issuerPPPublicKey},
			attributes:     map[interface{}]interface{}{},
			expectError:    false,
			expectIdentity: dummyIssuer,
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
			issuers: []driver.Identity{dummyIssuer},
			attributes: map[interface{}]interface{}{
				ttx.IssuerFSCIdentityKey: 1234,
			},
			expectError:    true,
			expectIdentity: nil,
		},
		{
			name:    "IssuerPublicParamsPublicKey is not a view Identity",
			issuers: []driver.Identity{dummyIssuer},
			attributes: map[interface{}]interface{}{
				ttx.IssuerFSCIdentityKey:        issuerFSCIdentity,
				ttx.IssuerPublicParamsPublicKey: 1234,
			},
			expectError:    true,
			expectIdentity: nil,
		},
		{
			name:    "IssuerFSCIdentityKey is none, issuers present",
			issuers: []driver.Identity{dummyIssuer},
			attributes: map[interface{}]interface{}{
				ttx.IssuerFSCIdentityKey: view.Identity(nil),
			},
			expectError:    false,
			expectIdentity: dummyIssuer,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := SelectIssuerForRedeem(tc.issuers, &driver.TransferOptions{Attributes: tc.attributes})
			if tc.expectError {
				require.Error(t, err)
				assert.Nil(t, id)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectIdentity, id)
			}
		})
	}
}
