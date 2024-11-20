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
	"github.com/pkg/errors"
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
		return nil, nil
	}

	tokenID, hasTokenID := opts.Attributes[TokenIDKey]
	network, hasNetwork := opts.Attributes[NetworkKey]
	proofOpt, hasProof := opts.Attributes[ProofKey]
	if !hasTokenID && !hasNetwork && !hasProof {
		return nil, nil
	}

	if hasTokenID && hasNetwork && hasProof {
		marshalled, err := json.Marshal(&IssueMetadata{tokenID.(*token.ID), network.(string)})
		if err != nil {
			return nil, err
		}
		key := common.Hashable(marshalled).String()
		attributes[key] = marshalled
		attributes[key+ProofOfClaimSuffix] = proofOpt.([]byte)
		return attributes, nil
	}

	return nil, errors.Errorf("missing token ID or network or proof")
}
