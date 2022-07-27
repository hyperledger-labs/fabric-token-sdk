/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// Issue returns an IssueAction as a function of the passed arguments
// Issue also returns a serialization TokenInformation associated with issued tokens
// and the identity of the issuer
func (s *Service) Issue(issuerIdentity view.Identity, typ string, values []uint64, owners [][]byte, opts *driver.IssueOptions) (driver.IssueAction, [][]byte, view.Identity, error) {
	for _, owner := range owners {
		// a recipient cannot be empty
		if len(owner) == 0 {
			return nil, nil, nil, errors.Errorf("all recipients should be defined")
		}
	}

	var outs []*TransferOutput
	var infos [][]byte
	for i, v := range values {
		outs = append(outs, &TransferOutput{
			Output: &token2.Token{
				Owner: &token2.Owner{
					Raw: owners[i],
				},
				Type:     typ,
				Quantity: token2.NewQuantityFromUInt64(v).Hex(),
			},
		})

		ti := &TokenInformation{
			Issuer: issuerIdentity,
		}
		tiRaw, err := ti.Serialize()
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "failed serializing token information")
		}
		infos = append(infos, tiRaw)
	}

	return &IssueAction{Issuer: issuerIdentity, Outputs: outs},
		infos,
		issuerIdentity,
		nil
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
