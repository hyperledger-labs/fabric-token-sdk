/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"context"
	"time"

	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type IssueService struct {
	PublicParametersManager common2.PublicParametersManager[*crypto.PublicParams]
	WalletService           driver.WalletService
	Deserializer            driver.Deserializer
	Metrics                 *Metrics
}

func NewIssueService(
	publicParametersManager common2.PublicParametersManager[*crypto.PublicParams],
	walletService driver.WalletService,
	deserializer driver.Deserializer,
	metrics *Metrics,
) *IssueService {
	return &IssueService{
		PublicParametersManager: publicParametersManager,
		WalletService:           walletService,
		Deserializer:            deserializer,
		Metrics:                 metrics,
	}
}

// Issue returns an IssueAction as a function of the passed arguments
// Issue also returns a serialization TokenInformation associated with issued tokens
// and the identity of the issuer
func (s *IssueService) Issue(ctx context.Context, issuerIdentity driver.Identity, tokenType token.Type, values []uint64, owners [][]byte, opts *driver.IssueOptions) (driver.IssueAction, *driver.IssueMetadata, error) {
	for _, owner := range owners {
		// a recipient cannot be empty
		if len(owner) == 0 {
			return nil, nil, errors.Errorf("all recipients should be defined")
		}
	}

	w, err := s.WalletService.IssuerWallet(issuerIdentity)
	if err != nil {
		return nil, nil, err
	}
	signer, err := w.GetSigner(issuerIdentity)
	if err != nil {
		return nil, nil, err
	}

	pp := s.PublicParametersManager.PublicParams()
	issuer := &issue.Issuer{}
	issuer.New(tokenType, &common.WrappedSigningIdentity{
		Identity: issuerIdentity,
		Signer:   signer,
	}, pp)

	start := time.Now()
	action, zkOutputsMetadata, err := issuer.GenerateZKIssue(values, owners)
	duration := time.Since(start)
	if err != nil {
		return nil, nil, err
	}
	s.Metrics.zkIssueDuration.Observe(float64(duration.Milliseconds()))

	var outputsMetadata [][]byte
	for _, meta := range zkOutputsMetadata {
		raw, err := meta.Serialize()
		if err != nil {
			return nil, nil, errors.WithMessage(err, "failed serializing token info")
		}
		outputsMetadata = append(outputsMetadata, raw)
	}

	issuerSerializedIdentity, err := issuer.Signer.Serialize()
	if err != nil {
		return nil, nil, err
	}

	outputs, err := action.GetSerializedOutputs()
	if err != nil {
		return nil, nil, err
	}
	auditInfo, err := s.Deserializer.GetOwnerAuditInfo(owners[0], s.WalletService)
	if err != nil {
		return nil, nil, err
	}

	meta := &driver.IssueMetadata{
		Issuer:              issuerSerializedIdentity,
		Outputs:             outputs,
		OutputsMetadata:     outputsMetadata,
		Receivers:           []driver.Identity{driver.Identity(owners[0])},
		ReceiversAuditInfos: auditInfo,
		ExtraSigners:        nil,
	}

	return action, meta, err
}

// VerifyIssue checks if the outputs of an IssueAction match the passed metadata
func (s *IssueService) VerifyIssue(ia driver.IssueAction, outputsMetadata [][]byte) error {
	if ia == nil {
		return errors.New("failed to verify issue: nil issue action")
	}
	action, ok := ia.(*issue.IssueAction)
	if !ok {
		return errors.New("failed to verify issue: expected *zkatdlog.IssueAction")
	}
	pp := s.PublicParametersManager.PublicParameters()
	coms, err := action.GetCommitments()
	if err != nil {
		return errors.New("failed to verify issue")
	}
	// todo check tokenInfo
	return issue.NewVerifier(
		coms,
		pp.(*crypto.PublicParams)).Verify(action.GetProof())
}

// DeserializeIssueAction un-marshals raw bytes into a zkatdlog IssueAction
func (s *IssueService) DeserializeIssueAction(raw []byte) (driver.IssueAction, error) {
	issue := &issue.IssueAction{}
	err := issue.Deserialize(raw)
	if err != nil {
		return nil, errors.Wrap(err, "failed to deserialize issue action")
	}
	return issue, nil
}
