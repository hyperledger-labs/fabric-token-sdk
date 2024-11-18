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
	TokenIDKey = pledge.TokenIDKey
	NetworkKey = pledge.NetworkKey
	ProofKey   = pledge.ProofKey
)

type IssueMetadata struct {
	// OriginTokenID is the identifier of the pledged token in the origin network
	OriginTokenID *token.ID
	// OriginNetwork is the network where the pledge took place
	OriginNetwork string
}

func IssueActionMetadata(attributes map[string][]byte, opts *driver.IssueOptions) (map[string][]byte, error) {
	var metadata *IssueMetadata
	var proof []byte
	if len(opts.Attributes) != 0 {
		tokenID, ok1 := opts.Attributes[TokenIDKey]
		network, ok2 := opts.Attributes[NetworkKey]
		proofOpt, ok3 := opts.Attributes[ProofKey]
		if ok1 && ok2 {
			metadata = &IssueMetadata{
				OriginTokenID: tokenID.(*token.ID),
				OriginNetwork: network.(string),
			}
		}
		if ok3 {
			proof = proofOpt.([]byte)
		}
	}
	if metadata != nil {
		marshalled, err := json.Marshal(metadata)
		if err != nil {
			return nil, errors.Wrapf(err, "failed marshaling metadata; origin network [%s]; origin tokenID [%s]", metadata.OriginNetwork, metadata.OriginTokenID)
		}
		key := common.Hashable(marshalled).String()
		attributes[key] = marshalled
		attributes[key+"proof_of_claim"] = proof
	}

	return attributes, nil
}
