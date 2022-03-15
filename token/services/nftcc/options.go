/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nftcc

import "github.com/hyperledger-labs/fabric-token-sdk/token"

// WithUniqueID sets the unique ID of the NFT
func WithUniqueID(uniqueID string) token.IssueOption {
	return token.WithIssueAttribute("github.com/hyperledger-labs/fabric-token-sdk/token/services/nftcc/UniqueID", uniqueID)
}
