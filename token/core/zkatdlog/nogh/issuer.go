/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue/nonanonym"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
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

	signer, err := s.IssuerWalletByIdentity(issuerIdentity).GetSigner(issuerIdentity)
	if err != nil {
		return nil, nil, nil, err
	}

	pp := s.PublicParams()
	issuer := &nonanonym.Issuer{}
	issuer.New(typ, &common.WrappedSigningIdentity{
		Identity: issuerIdentity,
		Signer:   signer,
	}, pp)

	issue, outputMetadata, err := issuer.GenerateZKIssue(values, owners)
	if err != nil {
		return nil, nil, nil, err
	}

	var outputMetadataRaw [][]byte
	for _, meta := range outputMetadata {
		raw, err := meta.Serialize()
		if err != nil {
			return nil, nil, nil, errors.WithMessage(err, "failed serializing token info")
		}
		outputMetadataRaw = append(outputMetadataRaw, raw)
	}

	fid, err := issuer.Signer.Serialize()
	if err != nil {
		return nil, nil, nil, err
	}

	return issue, outputMetadataRaw, fid, err
}

// VerifyIssue checks if the outputs of an IssueAction match the passed metadata
func (s *Service) VerifyIssue(ia driver.IssueAction, outputsMetadata [][]byte) error {
	if ia == nil {
		return errors.New("failed to verify issue: nil issue action")
	}
	action, ok := ia.(*issue.IssueAction)
	if !ok {
		return errors.New("failed to verify issue: expected *zkatdlog.IssueAction")
	}
	pp := s.PublicParams()
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
