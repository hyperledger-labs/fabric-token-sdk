/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

// SelectIssuerForRedeem return the issuer's public key to use for a redeem.
// If opts specify an FSC issuer identity, then we expect to find the opts also the public key to add in the transfer action.
// Otherwise, the first public key in the public params is used.
func SelectIssuerForRedeem(issuers []driver.Identity, opts *driver.TransferOptions) (driver.Identity, error) {
	issuerNetworkIdentity, err := ttx.GetFSCIssuerIdentityFromOpts(opts.Attributes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get issuer network identity")
	}
	if !issuerNetworkIdentity.IsNone() {
		issuerSigningKey, err := ttx.GetIssuerPublicParamsPublicKeyFromOpts(opts.Attributes)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get issuer public params public key")
		}
		if issuerSigningKey.IsNone() {
			return nil, errors.New("issuer public params public key not found in opts")
		}
		return issuerSigningKey, nil
	}

	if len(issuers) < 1 {
		return nil, errors.New("no issuer found")
	}
	return issuers[0], nil
}
