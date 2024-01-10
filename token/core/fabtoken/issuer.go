/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// Issue returns an IssueAction as a function of the passed arguments
// Issue also returns a serialization OutputMetadata associated with issued tokens
// and the identity of the issuer
func (s *Service) Issue(issuerIdentity view.Identity, tokenType string, values []uint64, owners [][]byte, opts *driver.IssueOptions) (driver.IssueAction, *driver.IssueMetadata, error) {
	for _, owner := range owners {
		// a recipient cannot be empty
		if len(owner) == 0 {
			return nil, nil, errors.Errorf("all recipients should be defined")
		}
	}

	var outs []*Output
	var metas [][]byte
	pp := s.PublicParamsManager().PublicParameters()
	if pp == nil {
		return nil, nil, errors.Errorf("public paramenters not set")
	}
	precision := pp.Precision()
	for i, v := range values {
		q, err := token2.UInt64ToQuantity(v, precision)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to convert [%d] to quantity of precision [%d]", v, precision)
		}
		outs = append(outs, &Output{
			Output: &token2.Token{
				Owner: &token2.Owner{
					Raw: owners[i],
				},
				Type:     tokenType,
				Quantity: q.Hex(),
			},
		})

		meta := &OutputMetadata{
			Issuer: issuerIdentity,
		}
		metaRaw, err := meta.Serialize()
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed serializing token information")
		}
		metas = append(metas, metaRaw)
	}

	md, err := getIssueActionMetadata(opts)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed getting issue action metadata")
	}

	issueAction := &IssueAction{
		Issuer:   issuerIdentity,
		Outputs:  outs,
		Metadata: md,
	}
	issueMetadata := &driver.IssueMetadata{
		Issuer:    issuerIdentity,
		TokenInfo: metas,
	}
	return issueAction, issueMetadata, nil
}

func getIssueActionMetadata(opts *driver.IssueOptions) (map[string][]byte, error) {
	var metadata *IssueMetadata
	var proof []byte
	if len(opts.Attributes) != 0 {
		tokenID, ok1 := opts.Attributes["github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge/tokenID"]
		network, ok2 := opts.Attributes["github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge/network"]
		proofOpt, ok3 := opts.Attributes["github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge/proof"]
		if ok1 && ok2 {
			metadata = &IssueMetadata{
				OriginTokenID: tokenID.(*token2.ID),
				OriginNetwork: network.(string),
			}
		}
		if ok3 {
			proof = proofOpt.([]byte)
		}
	}
	if metadata != nil {
		marshalled, err := json.Marshal(metadata)
		key := hash.Hashable(marshalled).String()
		if err != nil {
			return nil, errors.Wrapf(err, "failed marshaling metadata; origin network [%s]; origin tokenID [%s]", metadata.OriginNetwork, metadata.OriginTokenID)
		}
		return map[string][]byte{key: marshalled, key + "proof_of_claim": proof}, nil
	}
	return nil, nil
}

// VerifyIssue checks if the outputs of an IssueAction match the passed tokenInfos
func (s *Service) VerifyIssue(tr driver.IssueAction, tokenInfos [][]byte) error {
	// TODO:
	return nil
}

// DeserializeIssueAction un-marshals the passed bytes into an IssueAction
// If unmarshalling fails, then DeserializeIssueAction returns an error
func (s *Service) DeserializeIssueAction(raw []byte) (driver.IssueAction, error) {
	issue := &IssueAction{}
	if err := issue.Deserialize(raw); err != nil {
		return nil, errors.Wrap(err, "failed deserializing issue action")
	}
	return issue, nil
}
