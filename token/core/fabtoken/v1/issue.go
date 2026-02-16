/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/meta"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type IssueService struct {
	PublicParamsManager driver.PublicParamsManager
	WalletService       driver.WalletService
	Deserializer        driver.Deserializer
}

func NewIssueService(publicParamsManager driver.PublicParamsManager, walletService driver.WalletService, deserializer driver.Deserializer) *IssueService {
	return &IssueService{PublicParamsManager: publicParamsManager, WalletService: walletService, Deserializer: deserializer}
}

// Issue returns an IssueAction as a function of the passed arguments
// Issue also returns a serialization OutputMetadata associated with issued tokens
// and the identity of the issuer
func (s *IssueService) Issue(ctx context.Context, issuerIdentity driver.Identity, tokenType token2.Type, values []uint64, owners [][]byte, opts *driver.IssueOptions) (driver.IssueAction, *driver.IssueMetadata, error) {
	for _, owner := range owners {
		// a recipient cannot be empty
		if len(owner) == 0 {
			return nil, nil, errors.Errorf("all recipients should be defined")
		}
	}

	if issuerIdentity.IsNone() && len(tokenType) == 0 && values == nil {
		return nil, nil, errors.Errorf("issuer identity, token type and values should be defined")
	}
	if opts != nil {
		if opts.TokensUpgradeRequest != nil {
			return nil, nil, errors.Errorf("redeem during issue is not supported")
		}
	}

	precision := s.PublicParamsManager.PublicParameters().Precision()
	var outs []*v1.Output
	var outputsMetadata []*driver.IssueOutputMetadata
	for i, v := range values {
		q, err := token2.UInt64ToQuantity(v, precision)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to convert [%d] to quantity of precision [%d]", v, precision)
		}
		outs = append(outs, &v1.Output{
			Owner:    owners[i],
			Type:     tokenType,
			Quantity: q.Hex(),
		})

		outputMetadata := &v1.OutputMetadata{
			Issuer: issuerIdentity,
		}
		outputMetadataRaw, err := outputMetadata.Serialize()
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed serializing token information")
		}
		auditInfo, err := s.Deserializer.GetAuditInfo(ctx, owners[i], s.WalletService)
		if err != nil {
			return nil, nil, err
		}
		outputsMetadata = append(outputsMetadata, &driver.IssueOutputMetadata{
			OutputMetadata: outputMetadataRaw,
			Receivers: []*driver.AuditableIdentity{
				{
					Identity:  owners[i],
					AuditInfo: auditInfo,
				},
			},
		})
	}
	issuerAuditInfo, err := s.Deserializer.GetAuditInfo(ctx, issuerIdentity, s.WalletService)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get audit info for issuer identity")
	}

	action := &v1.IssueAction{
		Issuer:  issuerIdentity,
		Outputs: outs,
	}
	// add issuer action's metadata
	if opts != nil {
		action.Metadata = meta.IssueActionMetadata(opts.Attributes)
	}

	meta := &driver.IssueMetadata{
		Issuer: driver.AuditableIdentity{
			Identity:  issuerIdentity,
			AuditInfo: issuerAuditInfo,
		},
		Inputs:       nil,
		Outputs:      outputsMetadata,
		ExtraSigners: nil,
	}

	return action, meta, nil
}

// VerifyIssue checks if the outputs of an IssueAction match the passed tokenInfos
func (s *IssueService) VerifyIssue(ctx context.Context, ia driver.IssueAction, metadata []*driver.IssueOutputMetadata) error {
	// TODO:
	return nil
}

// DeserializeIssueAction un-marshals the passed bytes into an IssueAction
// If unmarshalling fails, then DeserializeIssueAction returns an error
func (s *IssueService) DeserializeIssueAction(raw []byte) (driver.IssueAction, error) {
	issue := &v1.IssueAction{}
	if err := issue.Deserialize(raw); err != nil {
		return nil, errors.Wrap(err, "failed deserializing issue action")
	}

	return issue, nil
}
