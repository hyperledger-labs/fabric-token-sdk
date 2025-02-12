/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/meta"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type TransferService struct {
	Logger                  logging.Logger
	PublicParametersManager common.PublicParametersManager[*PublicParams]
	WalletService           driver.WalletService
	TokenLoader             TokenLoader
	Deserializer            driver.Deserializer
}

func NewTransferService(
	logger logging.Logger,
	publicParametersManager common.PublicParametersManager[*PublicParams],
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
	// select inputs
	inputTokens, err := s.TokenLoader.GetTokens(tokenIDs)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to load tokens")
	}

	var inputs []*Output
	for _, tok := range inputTokens {
		s.Logger.Debugf("Selected output [%s,%s,%s]", tok.Type, tok.Quantity, driver.Identity(tok.Owner))
		t := Output(*tok)
		inputs = append(inputs, &t)
	}

	// prepare outputs
	var outs []*Output
	for _, output := range Outputs {
		outs = append(outs, &Output{
			Owner:    output.Owner,
			Type:     output.Type,
			Quantity: output.Quantity,
		})
	}

	// assemble transfer metadata
	ws := s.WalletService
	metadata := &driver.TransferMetadata{
		Inputs:       nil,
		Outputs:      nil,
		ExtraSigners: nil,
	}

	// inputs
	for i, t := range inputTokens {
		auditInfo, err := s.Deserializer.GetOwnerAuditInfo(t.Owner, ws)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting audit info for sender identity [%s]", driver.Identity(t.Owner).String())
		}
		metadata.Inputs = append(metadata.Inputs, &driver.TransferInputMetadata{
			TokenID: tokenIDs[i],
			Senders: []*driver.AuditableIdentity{
				{
					Identity:  t.Owner,
					AuditInfo: auditInfo[0],
				},
			},
		})
	}

	// outputs
	outputMetadata := &OutputMetadata{}
	outputMetadataRaw, err := outputMetadata.Serialize()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed serializing output information")
	}
	for _, output := range outs {
		var receivers []driver.Identity
		var receiversAuditInfo [][]byte
		transferOutputMetadata := &driver.TransferOutputMetadata{
			OutputMetadata:  outputMetadataRaw,
			OutputAuditInfo: nil,
			Receivers:       nil,
		}

		if len(output.Owner) == 0 { // redeem
			receivers = append(receivers, output.Owner)
			receiversAuditInfo = append(receiversAuditInfo, []byte{})
		} else {
			recipients, err := s.Deserializer.Recipients(output.Owner)
			if err != nil {
				return nil, nil, errors.Wrap(err, "failed getting recipients")
			}
			receivers = append(receivers, recipients...)
			receiversAuditInfo, err = s.Deserializer.GetOwnerAuditInfo(output.Owner, ws)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed getting audit info for sender identity [%s]", driver.Identity(output.Owner).String())
			}
		}
		for i, receiver := range receivers {
			transferOutputMetadata.Receivers = append(transferOutputMetadata.Receivers, &driver.AuditableIdentity{
				Identity:  receiver,
				AuditInfo: receiversAuditInfo[i],
			})
		}

		metadata.Outputs = append(metadata.Outputs, transferOutputMetadata)
	}

	// receiverIsSender := make([]bool, len(receivers))
	// for i, receiver := range receivers {
	// 	_, err = ws.OwnerWallet(receiver)
	// 	receiverIsSender[i] = err == nil
	// }
	//
	// s.Logger.Debugf("Transfer metadata: [out:%d, rec:%d]", len(metadata.OutputsMetadata), len(metadata.Receivers))

	// return
	transfer := &TransferAction{
		Inputs:      tokenIDs,
		InputTokens: inputs,
		Outputs:     outs,
		Metadata:    meta.TransferActionMetadata(opts.Attributes),
	}

	return transfer, metadata, nil
}

// VerifyTransfer checks the outputs in the TransferAction against the passed tokenInfos
func (s *TransferService) VerifyTransfer(tr driver.TransferAction, outputMetadata []*driver.TransferOutputMetadata) error {
	// TODO:
	return nil
}

// DeserializeTransferAction un-marshals a TransferAction from the passed array of bytes.
// DeserializeTransferAction returns an error, if the un-marshalling fails.
func (s *TransferService) DeserializeTransferAction(raw []byte) (driver.TransferAction, error) {
	t := &TransferAction{}
	if err := t.Deserialize(raw); err != nil {
		return nil, errors.Wrap(err, "failed deserializing transfer action")
	}
	return t, nil
}
