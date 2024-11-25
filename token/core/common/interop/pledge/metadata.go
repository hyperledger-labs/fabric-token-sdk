/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	TokenIDKey         = pledge.TokenIDKey
	NetworkKey         = pledge.NetworkKey
	ProofKey           = pledge.ProofKey
	ProofOfClaimSuffix = "proof_of_claim"
)

type IssueMetadata struct {
	// OriginTokenID is the identifier of the pledged token in the origin network
	OriginTokenID *token.ID
	// OriginNetwork is the network where the pledge took place
	OriginNetwork string
}

func IssueActionMetadata(attributes map[string][]byte, opts *driver.IssueOptions) (map[string][]byte, error) {
	if len(opts.Attributes) == 0 {
		return attributes, nil
	}

	tokenID, hasTokenID := opts.Attributes[TokenIDKey]
	network, hasNetwork := opts.Attributes[NetworkKey]
	if !hasTokenID && !hasNetwork {
		return attributes, nil
	}

	if hasTokenID && hasNetwork {
		marshalled, err := json.Marshal(&IssueMetadata{tokenID.(*token.ID), network.(string)})
		if err != nil {
			return nil, err
		}
		key := common.Hashable(marshalled).String()
		attributes[key] = marshalled

		// append proof, if needed
		var proof []byte
		proofOpt, hasProof := opts.Attributes[ProofKey]
		if hasProof {
			proof = proofOpt.([]byte)
		}
		attributes[key+ProofOfClaimSuffix] = proof
		return attributes, nil
	}

	return attributes, nil
}
