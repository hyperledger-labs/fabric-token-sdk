/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package nogh

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue/nonanonym"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
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

	signer, err := s.IssuerWalletByIdentity(issuerIdentity).GetSigner(issuerIdentity)
	if err != nil {
		return nil, nil, nil, err
	}

	pp, err := s.PublicParams()
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "failed to get public parameters")
	}
	issuer := &nonanonym.Issuer{}
	issuer.New(typ, &common.WrappedSigningIdentity{
		Identity: issuerIdentity,
		Signer:   signer,
	}, pp)

	issue, infos, err := issuer.GenerateZKIssue(values, owners)
	if err != nil {
		return nil, nil, nil, err
	}

	var infoRaws [][]byte
	for _, information := range infos {
		raw, err := information.Serialize()
		if err != nil {
			return nil, nil, nil, errors.WithMessage(err, "failed serializing token info")
		}
		infoRaws = append(infoRaws, raw)
	}

	fid, err := issuer.Signer.Serialize()
	if err != nil {
		return nil, nil, nil, err
	}

	return issue, infoRaws, fid, err
}

// VerifyIssue checks if the outputs of an IssueAction match the passed tokenInfos
func (s *Service) VerifyIssue(ia driver.IssueAction, tokenInfos [][]byte) error {
	action := ia.(*issue.IssueAction)
	if action == nil {
		return errors.New("failed to verify issue: nil issue action")
	}
	pp, err := s.PublicParams()
	if err != nil {
		return errors.Wrap(err, "failed to verify issue: can't get public parameters")
	}
	coms, err := action.GetCommitments()
	if err != nil {
		return errors.New("failed to verify issue")
	}
	// todo check tokenInfo
	return issue.NewVerifier(
		coms,
		action.IsAnonymous(),
		pp).Verify(action.GetProof())
}

// DeserializeIssueAction un-marshals raw bytes into a zkatdlog IssueAction
func (s *Service) DeserializeIssueAction(raw []byte) (driver.IssueAction, error) {
	issue := &issue.IssueAction{}
	err := issue.Deserialize(raw)
	if err != nil {
		return nil, errors.Wrap(err, "failed to deserialize issue action")
	}
	return issue, nil
}
