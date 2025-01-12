/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
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

	var outs []*Output
	var outputsMetadata [][]byte
	pp := s.PublicParamsManager.PublicParameters()
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
			Owner:    owners[i],
			Type:     tokenType,
			Quantity: q.Hex(),
		})

		meta := &OutputMetadata{
			Issuer: issuerIdentity,
		}
		metaRaw, err := meta.Serialize()
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed serializing token information")
		}
		outputsMetadata = append(outputsMetadata, metaRaw)
	}

	action := &IssueAction{Issuer: issuerIdentity, Outputs: outs}
	auditInfo, err := s.Deserializer.GetOwnerAuditInfo(owners[0], s.WalletService)
	if err != nil {
		return nil, nil, err
	}

	meta := &driver.IssueMetadata{
		Issuer:              issuerIdentity,
		OutputsMetadata:     outputsMetadata,
		Receivers:           []driver.Identity{driver.Identity(owners[0])},
		ReceiversAuditInfos: auditInfo,
		ExtraSigners:        nil,
	}
	return action, meta, nil
}

// VerifyIssue checks if the outputs of an IssueAction match the passed tokenInfos
func (s *IssueService) VerifyIssue(tr driver.IssueAction, tokenInfos [][]byte) error {
	// TODO:
	return nil
}

// DeserializeIssueAction un-marshals the passed bytes into an IssueAction
// If unmarshalling fails, then DeserializeIssueAction returns an error
func (s *IssueService) DeserializeIssueAction(raw []byte) (driver.IssueAction, error) {
	issue := &IssueAction{}
	if err := issue.Deserialize(raw); err != nil {
		return nil, errors.Wrap(err, "failed deserializing issue action")
	}
	return issue, nil
}
