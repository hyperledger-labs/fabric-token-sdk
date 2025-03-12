/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/meta"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TransferService struct {
	Logger                  logging.Logger
	PublicParametersManager common.PublicParametersManager[*v1.PublicParams]
	WalletService           driver.WalletService
	TokenLoader             TokenLoader
	Deserializer            driver.Deserializer
}

func NewTransferService(
	logger logging.Logger,
	publicParametersManager common.PublicParametersManager[*v1.PublicParams],
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
func (s *TransferService) Transfer(ctx context.Context, _ string, _ driver.OwnerWallet, tokenIDs []*token.ID, Outputs []*token.Token, opts *driver.TransferOptions) (driver.TransferAction, *driver.TransferMetadata, error) {
	var isRedeem bool
	// select inputs
	inputTokens, err := s.TokenLoader.GetTokens(tokenIDs)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to load tokens")
	}

	var inputs []*v1.Output
	for _, tok := range inputTokens {
		s.Logger.Debugf("Selected output [%s,%s,%s]", tok.Type, tok.Quantity, driver.Identity(tok.Owner))
		t := v1.Output(*tok)
		inputs = append(inputs, &t)
	}

	// prepare outputs
	var outs []*v1.Output
	for _, output := range Outputs {
		if len(output.Owner) == 0 {
			isRedeem = true
		}
		outs = append(outs, &v1.Output{
			Owner:    output.Owner,
			Type:     output.Type,
			Quantity: output.Quantity,
		})
	}

	// assemble transfer transferMetadata
	ws := s.WalletService

	// inputs
	transferInputsMetadata := make([]*driver.TransferInputMetadata, 0, len(inputTokens))
	for i, t := range inputTokens {
		auditInfo, err := s.Deserializer.GetAuditInfo(t.Owner, ws)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting audit info for sender identity [%s]", driver.Identity(t.Owner).String())
		}
		transferInputsMetadata = append(transferInputsMetadata, &driver.TransferInputMetadata{
			TokenID: tokenIDs[i],
			Senders: []*driver.AuditableIdentity{
				{
					Identity:  t.Owner,
					AuditInfo: auditInfo,
				},
			},
		})
	}

	// outputs
	outputMetadata := &v1.OutputMetadata{}
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
			outputAudiInfo, err = s.Deserializer.GetAuditInfo(output.Owner, ws)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed getting audit info for sender identity [%s]", driver.Identity(output.Owner).String())
			}
			recipients, err := s.Deserializer.Recipients(output.Owner)
			if err != nil {
				return nil, nil, errors.Wrap(err, "failed getting recipients")
			}
			receivers = append(receivers, recipients...)
			for _, receiver := range receivers {
				receiverAudiInfo, err := s.Deserializer.GetAuditInfo(receiver, ws)
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

	var extraSigners []driver.Identity
	if isRedeem {
		issuer := opts.Issuer
		if issuer == nil {
			issuers := s.PublicParametersManager.PublicParameters().Issuers()
			if len(issuers) == 0 {
				return nil, nil, errors.New("no issuer found")
			}
			issuer = issuers[0]
		}
		extraSigners = append(extraSigners, issuer)
	}

	// return
	transfer := &v1.TransferAction{
		Inputs:      tokenIDs,
		InputTokens: inputs,
		Outputs:     outs,
		Metadata:    meta.TransferActionMetadata(opts.Attributes),
	}
	transferMetadata := &driver.TransferMetadata{
		Inputs:       transferInputsMetadata,
		Outputs:      transferOutputsMetadata,
		ExtraSigners: extraSigners,
	}

	return transfer, transferMetadata, nil
}

// VerifyTransfer checks the outputs in the TransferAction against the passed tokenInfos
func (s *TransferService) VerifyTransfer(tr driver.TransferAction, outputMetadata []*driver.TransferOutputMetadata) error {
	// TODO:
	return nil
}

// DeserializeTransferAction un-marshals a TransferAction from the passed array of bytes.
// DeserializeTransferAction returns an error, if the un-marshalling fails.
func (s *TransferService) DeserializeTransferAction(raw []byte) (driver.TransferAction, error) {
	t := &v1.TransferAction{}
	if err := t.Deserialize(raw); err != nil {
		return nil, errors.Wrap(err, "failed deserializing transfer action")
	}
	return t, nil
}
