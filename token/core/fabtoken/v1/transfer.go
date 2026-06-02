/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/meta"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
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
		s.Logger.DebugfContext(ctx, "Selected output [%s,%s,%s]", tok.Type, tok.Quantity, driver.Identity(tok.Owner))
		inputs = append(inputs, new(actions.Output(*tok)))
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
		Inputs:  actionInputs,
		Outputs: outs,
		Issuer:  nil,
	}
	if opts != nil {
		transfer.Metadata = meta.TransferActionMetadata(opts.Attributes)
	}
	transferMetadata := &driver.TransferMetadata{
		Inputs:       transferInputsMetadata,
		Outputs:      transferOutputsMetadata,
		ExtraSigners: nil,
		Issuer:       nil,
	}

	if isRedeem {
		issuer, err := common.SelectIssuerForRedeem(s.PublicParametersManager.PublicParameters().Issuers(), opts)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to select issuer for redeem")
		}
		transfer.Issuer = issuer
		transferMetadata.Issuer = issuer
	}

	return transfer, transferMetadata, nil
}

// VerifyTransfer checks the outputs in the TransferAction against the passed tokenInfos
func (s *TransferService) VerifyTransfer(ctx context.Context, tr driver.TransferAction, outputMetadata []*driver.TransferOutputMetadata) error {
	if tr == nil {
		return errors.New("nil transfer action")
	}

	// Type assertion to get the concrete type
	action, ok := tr.(*actions.TransferAction)
	if !ok {
		return errors.New("expected *fabtoken.TransferAction")
	}

	// Validate the transfer action structure
	if err := action.Validate(); err != nil {
		return errors.Wrap(err, "invalid transfer action")
	}

	// Verify output count matches metadata count (if metadata is provided)
	if outputMetadata != nil && len(outputMetadata) > 0 && len(action.Outputs) != len(outputMetadata) {
		return errors.Errorf("number of outputs [%d] does not match number of metadata entries [%d]", len(action.Outputs), len(outputMetadata))
	}

	// Get precision for quantity calculations
	var precision uint64 = 64 // default precision
	if s.PublicParametersManager != nil && s.PublicParametersManager.PublicParameters() != nil {
		precision = s.PublicParametersManager.PublicParameters().Precision()
	}

	// Calculate input sum
	inputSum, err := calculateInputSum(action.Inputs, precision)
	if err != nil {
		return errors.Wrap(err, "failed to calculate input sum")
	}

	// Calculate output sum
	outputSum, err := calculateOutputSum(action.Outputs, precision)
	if err != nil {
		return errors.Wrap(err, "failed to calculate output sum")
	}

	// Verify input sum equals output sum (conservation of value)
	cmp := inputSum.Cmp(outputSum)
	if cmp != 0 {
		// Check if difference is due to redeem (redeem outputs have empty owner)
		if action.IsRedeem() {
			s.Logger.DebugfContext(ctx, "redeem detected, input sum [%s] output sum [%s]", inputSum.Decimal(), outputSum.Decimal())
		} else {
			return errors.Errorf("input sum [%s] does not match output sum [%s]", inputSum.Decimal(), outputSum.Decimal())
		}
	}

	// Verify token types match across inputs and outputs
	if err := verifyTokenTypes(action.Inputs, action.Outputs); err != nil {
		return errors.Wrap(err, "token type mismatch")
	}

	// Verify each output has valid data
	for i, output := range action.Outputs {
		if output == nil {
			return errors.Errorf("nil output at index [%d]", i)
		}
		if err := output.Validate(false); err != nil {
			return errors.Wrapf(err, "invalid output at index [%d]", i)
		}
	}

	s.Logger.DebugfContext(ctx, "transfer verified successfully: inputs=%d, outputs=%d, sum=%s", len(action.Inputs), len(action.Outputs), inputSum.Decimal())

	return nil
}

// calculateInputSum calculates the total quantity of all inputs
func calculateInputSum(inputs []*actions.TransferActionInput, precision uint64) (token.Quantity, error) {
	sum := token.NewZeroQuantity(precision)
	for _, in := range inputs {
		if in == nil || in.Input == nil {
			continue
		}
		q, err := token.ToQuantity(in.Input.Quantity, precision)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse quantity from input")
		}
		sum, err = sum.Add(q)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to add input quantity")
		}
	}
	return sum, nil
}

// calculateOutputSum calculates the total quantity of all outputs
func calculateOutputSum(outputs []*actions.Output, precision uint64) (token.Quantity, error) {
	sum := token.NewZeroQuantity(precision)
	for _, out := range outputs {
		if out == nil {
			continue
		}
		q, err := token.ToQuantity(out.Quantity, precision)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse quantity from output")
		}
		sum, err = sum.Add(q)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to add output quantity")
		}
	}
	return sum, nil
}

// verifyTokenTypes ensures all inputs and outputs have the same token type
func verifyTokenTypes(inputs []*actions.TransferActionInput, outputs []*actions.Output) error {
	if len(inputs) == 0 || len(outputs) == 0 {
		return nil
	}

	// Get the first input's type
	var expectedType token.Type
	for _, in := range inputs {
		if in != nil && in.Input != nil {
			expectedType = in.Input.Type
			break
		}
	}

	// Verify all inputs have the same type
	for i, in := range inputs {
		if in != nil && in.Input != nil && in.Input.Type != expectedType {
			return errors.Errorf("input at index [%d] has different type [%s] than expected [%s]", i, in.Input.Type, expectedType)
		}
	}

	// Verify all outputs have the same type as inputs
	for i, out := range outputs {
		if out != nil && out.Type != expectedType {
			return errors.Errorf("output at index [%d] has different type [%s] than expected [%s]", i, out.Type, expectedType)
		}
	}

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
