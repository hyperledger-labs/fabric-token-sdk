/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"context"
	"time"

	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/upgrade"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type IssueService struct {
	Logger                  logging.Logger
	PublicParametersManager common2.PublicParametersManager[*setup.PublicParams]
	WalletService           driver.WalletService
	Deserializer            driver.Deserializer
	Metrics                 *Metrics
	TokensService           *token2.TokensService
	TokensUpgradeService    *upgrade.Service
}

func NewIssueService(
	logger logging.Logger,
	publicParametersManager common2.PublicParametersManager[*setup.PublicParams],
	walletService driver.WalletService,
	deserializer driver.Deserializer,
	metrics *Metrics,
	tokensService *token2.TokensService,
	tokensUpgradeService *upgrade.Service,
) *IssueService {
	return &IssueService{
		Logger:                  logger,
		PublicParametersManager: publicParametersManager,
		WalletService:           walletService,
		Deserializer:            deserializer,
		Metrics:                 metrics,
		TokensService:           tokensService,
		TokensUpgradeService:    tokensUpgradeService,
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
	if opts == nil {
		opts = &driver.IssueOptions{}
	}

	pp := s.PublicParametersManager.PublicParams()
	if issuerIdentity.IsNone() && len(tokenType) == 0 && values == nil {
		// this is a special case where the issue contains also redemption
		// we need to extract token types and values from the passed tokens
		tokensToUpgrade, err := s.TokensUpgradeService.ProcessTokensUpgradeRequest(opts.TokensUpgradeRequest)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to extract token types and values from the passed tokens")
		}

		// check that token types are all the same and sum the token values
		if len(tokensToUpgrade) == 0 {
			return nil, nil, errors.New("no token types found in the passed tokens")
		}
		tokenType = tokensToUpgrade[0].Type
		var totalValue uint64
		for _, t := range tokensToUpgrade {
			if t.Type != tokenType {
				return nil, nil, errors.New("all token types should be the same")
			}
			// sum the token values
			q, err := token.NewUBigQuantity(t.Quantity, pp.QuantityPrecision)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to create quantity from [%s]", t.Quantity)
			}
			totalValue += q.Uint64()
		}
		values = []uint64{totalValue}

		s.Logger.Debugf("upgrade: extracted token type [%s] and value [%d] from the passed tokens", tokenType, totalValue)

		// fetch issuer identity
		issuerIdentity, err = opts.Wallet.GetIssuerIdentity(tokenType)
		if err != nil {
			return nil, nil, errors.WithMessagef(err, "failed getting issuer identity for type [%s]", tokenType)
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

	issuer := &issue.Issuer{}
	issuer.New(tokenType, &common.WrappedSigningIdentity{
		Identity: issuerIdentity,
		Signer:   signer,
	}, pp)

	start := time.Now()
	issueAction, zkOutputsMetadata, err := issuer.GenerateZKIssue(values, owners)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to generate zk issue")
	}
	duration := time.Since(start)
	s.Metrics.zkIssueDuration.Observe(float64(duration.Milliseconds()))

	// metadata

	var inputsMetadata []*driver.IssueInputMetadata
	if opts != nil && opts.TokensUpgradeRequest != nil && len(opts.TokensUpgradeRequest.Tokens) > 0 {
		tokens := opts.TokensUpgradeRequest.Tokens
		issueAction.Inputs = make([]*issue.ActionInput, len(tokens))
		for i, tok := range tokens {
			issueAction.Inputs[i] = &issue.ActionInput{
				ID:    tok.ID,
				Token: tok.Token,
			}
			inputsMetadata = append(inputsMetadata, &driver.IssueInputMetadata{
				TokenID: &tok.ID,
			})
		}
	}

	var outputsMetadata []*driver.IssueOutputMetadata
	for i, owner := range owners {
		raw, err := zkOutputsMetadata[i].Serialize()
		if err != nil {
			return nil, nil, errors.WithMessage(err, "failed serializing token info")
		}
		auditInfo, err := s.Deserializer.GetAuditInfo(owner, s.WalletService)
		if err != nil {
			return nil, nil, err
		}
		outputsMetadata = append(outputsMetadata, &driver.IssueOutputMetadata{
			OutputMetadata: raw,
			Receivers: []*driver.AuditableIdentity{
				{
					Identity:  owner,
					AuditInfo: auditInfo,
				},
			},
		})
	}

	issuerSerializedIdentity, err := issuer.Signer.Serialize()
	if err != nil {
		return nil, nil, err
	}
	issuerAuditInfo, err := s.Deserializer.GetAuditInfo(issuerIdentity, s.WalletService)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get audit info for issuer identity")
	}

	s.Logger.Debugf("issue with [%d] inputs", len(issueAction.Inputs))

	meta := &driver.IssueMetadata{
		Issuer: driver.AuditableIdentity{
			Identity:  issuerSerializedIdentity,
			AuditInfo: issuerAuditInfo,
		},
		Inputs:       inputsMetadata,
		Outputs:      outputsMetadata,
		ExtraSigners: nil,
	}

	return issueAction, meta, err
}

// VerifyIssue checks if the outputs of an IssueAction match the passed metadata
func (s *IssueService) VerifyIssue(ia driver.IssueAction, metadata []*driver.IssueOutputMetadata) error {
	if ia == nil {
		return errors.New("failed to verify issue: nil issue action")
	}
	action, ok := ia.(*issue.Action)
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
		pp.(*setup.PublicParams)).Verify(action.GetProof())
}

// DeserializeIssueAction un-marshals raw bytes into a zkatdlog IssueAction
func (s *IssueService) DeserializeIssueAction(raw []byte) (driver.IssueAction, error) {
	issue := &issue.Action{}
	err := issue.Deserialize(raw)
	if err != nil {
		return nil, errors.Wrap(err, "failed to deserialize issue action")
	}
	return issue, nil
}
