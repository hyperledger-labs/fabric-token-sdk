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
func (s *Service) Issue(issuerIdentity view.Identity, tokenType string, values []uint64, owners [][]byte, opts *driver.IssueOptions) (driver.IssueAction, *driver.IssueMetadata, error) {
	for _, owner := range owners {
		// a recipient cannot be empty
		if len(owner) == 0 {
			return nil, nil, errors.Errorf("all recipients should be defined")
		}
	}

	w, err := s.IssuerWalletByIdentity(issuerIdentity)
	if err != nil {
		return nil, nil, err
	}
	signer, err := w.GetSigner(issuerIdentity)
	if err != nil {
		return nil, nil, err
	}

	pp := s.PublicParams()
	if pp == nil {
		return nil, nil, errors.Errorf("public parameters not inizialized")
	}
	issuer := &nonanonym.Issuer{}
	issuer.New(tokenType, &common.WrappedSigningIdentity{
		Identity: issuerIdentity,
		Signer:   signer,
	}, pp)

	issue, outputMetadata, err := issuer.GenerateZKIssue(values, owners)
	if err != nil {
		return nil, nil, err
	}

	var outputMetadataRaw [][]byte
	for _, meta := range outputMetadata {
		raw, err := meta.Serialize()
		if err != nil {
			return nil, nil, errors.WithMessage(err, "failed serializing token info")
		}
		outputMetadataRaw = append(outputMetadataRaw, raw)
	}

	issuerSerializedIdentity, err := issuer.Signer.Serialize()
	if err != nil {
		return nil, nil, err
	}

	meta := &driver.IssueMetadata{
		Issuer:    issuerSerializedIdentity,
		TokenInfo: outputMetadataRaw,
	}
	return issue, meta, err
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
	if pp == nil {
		return errors.Errorf("public parameters not inizialized")
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
