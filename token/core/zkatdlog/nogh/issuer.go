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

func (s *service) Issue(issuerIdentity view.Identity, typ string, values []uint64, owners [][]byte, opts *driver.IssueOptions) (driver.IssueAction, [][]byte, view.Identity, error) {
	for _, owner := range owners {
		if len(owner) == 0 {
			return nil, nil, nil, errors.Errorf("all recipients should be defined")
		}
	}

	signer, err := s.IssuerWalletByIdentity(issuerIdentity).GetSigner(issuerIdentity)
	if err != nil {
		return nil, nil, nil, err
	}

	issuer := &nonanonym.Issuer{}
	issuer.New(typ, &common.WrappedSigningIdentity{
		Identity: issuerIdentity,
		Signer:   signer,
	}, s.PublicParams())

	issue, infos, err := issuer.GenerateZKIssue(values, owners)
	if err != nil {
		return nil, nil, nil, err
	}

	//if err := s.registerIssuerSigner(issuer.Signer); err != nil {
	//	return nil, nil, nil, errors.WithMessage(err, "failed registering zkat issuer")
	//}

	infoRaws := [][]byte{}
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

func (s *service) VerifyIssue(ia driver.IssueAction, tokenInfos [][]byte) error {
	action := ia.(*issue.IssueAction)

	return issue.NewVerifier(
		action.GetCommitments(),
		action.IsAnonymous(),
		s.PublicParams()).Verify(action.GetProof())
}

func (s *service) DeserializeIssueAction(raw []byte) (driver.IssueAction, error) {
	issue := &issue.IssueAction{}
	err := issue.Deserialize(raw)
	if err != nil {
		return nil, err
	}
	return issue, nil
}
