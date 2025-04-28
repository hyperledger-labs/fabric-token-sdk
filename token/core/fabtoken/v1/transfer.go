/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/meta"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TransferService struct {
	Logger                  logging.Logger
	PublicParametersManager common.PublicParametersManager[*setup.PublicParams]
	WalletService           driver.WalletService
	TokenLoader             TokenLoader
	Deserializer            driver.Deserializer
}

func NewTransferService(
	logger logging.Logger,
	publicParametersManager common.PublicParametersManager[*setup.PublicParams],
	walletService driver.WalletService,
	tokenLoader TokenLoader,
	deserializer driver.Deserializer,
) *TransferService {
	return &TransferService{
		Logger:                  logger,
		PublicParametersManager: publicParametersManager,
		WalletService:           walletService,
		TokenLoader:             tokenLoader,
		Deserializer:            deserializer,
	}
}

// Transfer returns a TransferAction as a function of the passed arguments
// It also returns the corresponding TransferMetadata
func (s *TransferService) Transfer(ctx context.Context, anchor driver.TokenRequestAnchor, wallet driver.OwnerWallet, ids []*token.ID, outputs []*token.Token, opts *driver.TransferOptions) (driver.TransferAction, *driver.TransferMetadata, error) {
	// select inputs
	inputTokens, err := s.TokenLoader.GetTokens(ctx, ids)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to load tokens")
	}

	var inputs []*actions.Output
	for _, tok := range inputTokens {
		s.Logger.Debugf("Selected output [%s,%s,%s]", tok.Type, tok.Quantity, driver.Identity(tok.Owner))
		t := actions.Output(*tok)
		inputs = append(inputs, &t)
	}

	// prepare outputs
	var isRedeem bool
	var outs []*actions.Output
	for _, output := range outputs {
		outs = append(outs, &actions.Output{
			Owner:    output.Owner,
			Type:     output.Type,
			Quantity: output.Quantity,
		})

		if len(output.Owner) == 0 {
			isRedeem = true
		}
	}

	// assemble transfer transferMetadata
	ws := s.WalletService

	// inputs
	transferInputsMetadata := make([]*driver.TransferInputMetadata, 0, len(inputTokens))
	for i, t := range inputTokens {
		auditInfo, err := s.Deserializer.GetAuditInfo(ctx, t.Owner, ws)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting audit info for sender identity [%s]", driver.Identity(t.Owner).String())
		}
		transferInputsMetadata = append(transferInputsMetadata, &driver.TransferInputMetadata{
			TokenID: ids[i],
			Senders: []*driver.AuditableIdentity{
				{
					Identity:  t.Owner,
					AuditInfo: auditInfo,
				},
			},
		})
	}

	// outputs
	outputMetadata := &actions.OutputMetadata{}
	outputMetadataRaw, err := outputMetadata.Serialize()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed serializing output information")
	}
	transferOutputsMetadata := make([]*driver.TransferOutputMetadata, 0, len(outs))
	for _, output := range outs {
		var outputAudiInfo []byte
		var receivers []driver.Identity
		var receiversAuditInfo [][]byte
		var outputReceivers []*driver.AuditableIdentity

		if len(output.Owner) == 0 { // redeem
			outputAudiInfo = nil
			receivers = append(receivers, output.Owner)
			receiversAuditInfo = append(receiversAuditInfo, []byte{})
			outputReceivers = make([]*driver.AuditableIdentity, 0, 1)
		} else {
			outputAudiInfo, err = s.Deserializer.GetAuditInfo(ctx, output.Owner, ws)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed getting audit info for sender identity [%s]", driver.Identity(output.Owner).String())
			}
			recipients, err := s.Deserializer.Recipients(output.Owner)
			if err != nil {
				return nil, nil, errors.Wrap(err, "failed getting recipients")
			}
			receivers = append(receivers, recipients...)
			for _, receiver := range receivers {
				receiverAudiInfo, err := s.Deserializer.GetAuditInfo(ctx, receiver, ws)
				if err != nil {
					return nil, nil, errors.Wrapf(err, "failed getting audit info for receiver identity [%s]", receiver)
				}
				receiversAuditInfo = append(receiversAuditInfo, receiverAudiInfo)
			}
			outputReceivers = make([]*driver.AuditableIdentity, 0, len(recipients))
		}
		for i, receiver := range receivers {
			outputReceivers = append(outputReceivers, &driver.AuditableIdentity{
				Identity:  receiver,
				AuditInfo: receiversAuditInfo[i],
			})
		}

		transferOutputsMetadata = append(transferOutputsMetadata, &driver.TransferOutputMetadata{
			OutputMetadata:  outputMetadataRaw,
			OutputAuditInfo: outputAudiInfo,
			Receivers:       outputReceivers,
		})
	}

	// return
	actionInputs := make([]*actions.TransferActionInput, len(ids))
	for i, id := range ids {
		actionInputs[i] = &actions.TransferActionInput{
			ID:    id,
			Input: inputs[i],
		}
	}
	transfer := &actions.TransferAction{
		Inputs:   actionInputs,
		Outputs:  outs,
		Metadata: meta.TransferActionMetadata(opts.Attributes),
		Issuer:   nil,
	}
	transferMetadata := &driver.TransferMetadata{
		Inputs:       transferInputsMetadata,
		Outputs:      transferOutputsMetadata,
		ExtraSigners: nil,
		Issuer:       nil,
	}

	if isRedeem {
		var issuer driver.Identity
		issuerNetworkIdentity, err := ttx.GetFSCIssuerIdentityFromOpts(opts.Attributes)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to get issuer network identity")
		}
		if !issuerNetworkIdentity.IsNone() {
			issuerSigningKey, err := ttx.GetIssuerSigningKeyFromOpts(opts.Attributes)
			if (err != nil) || (issuerSigningKey == nil) {
				return nil, nil, errors.Wrap(err, "failed to get issuer signing key")
			}
			issuer = issuerSigningKey
		} else {
			issuers := s.PublicParametersManager.PublicParameters().Issuers()
			if len(issuers) < 1 {
				return nil, nil, errors.New("no issuer found")
			}
			issuer = issuers[0]
		}
		transfer.Issuer = issuer
		transferMetadata.Issuer = issuer
	}

	return transfer, transferMetadata, nil
}

// VerifyTransfer checks the outputs in the TransferAction against the passed tokenInfos
func (s *TransferService) VerifyTransfer(ctx context.Context, tr driver.TransferAction, outputMetadata []*driver.TransferOutputMetadata) error {
	// TODO:
	return nil
}

// DeserializeTransferAction un-marshals a TransferAction from the passed array of bytes.
// DeserializeTransferAction returns an error, if the un-marshalling fails.
func (s *TransferService) DeserializeTransferAction(raw []byte) (driver.TransferAction, error) {
	t := &actions.TransferAction{}
	if err := t.Deserialize(raw); err != nil {
		return nil, errors.Wrap(err, "failed deserializing transfer action")
	}
	return t, nil
}
