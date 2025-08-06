/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"context"
	"time"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/meta"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"go.opentelemetry.io/otel/trace"
)

type LoadedToken = common.LoadedToken[[]byte, []byte]

type PreparedTransferInput struct {
	Token          *token.Token
	Metadata       *token.Metadata
	UpgradeWitness *token.UpgradeWitness
	Owner          driver.Identity
}

type PreparedTransferInputs []PreparedTransferInput

func (p *PreparedTransferInputs) Owners() []driver.Identity {
	owners := make([]driver.Identity, len(*p))
	for i, input := range *p {
		owners[i] = input.Owner
	}
	return owners
}

func (p *PreparedTransferInputs) Tokens() []*token.Token {
	tokens := make([]*token.Token, len(*p))
	for i, input := range *p {
		tokens[i] = input.Token
	}
	return tokens
}

func (p *PreparedTransferInputs) Metadata() []*token.Metadata {
	metas := make([]*token.Metadata, len(*p))
	for i, input := range *p {
		metas[i] = input.Metadata
	}
	return metas
}

type TokenLoader interface {
	LoadTokens(ctx context.Context, ids []*token2.ID) ([]LoadedToken, error)
}

type TokenDeserializer interface {
	DeserializeToken(ctx context.Context, outputFormat token2.Format, outputRaw []byte, metadataRaw []byte) (*token.Token, *token.Metadata, *token.UpgradeWitness, error)
}

type TransferService struct {
	Logger                  logging.Logger
	PublicParametersManager common.PublicParametersManager[*setup.PublicParams]
	WalletService           driver.WalletService
	TokenLoader             TokenLoader
	IdentityDeserializer    driver.Deserializer
	TokenDeserializer       TokenDeserializer
	Metrics                 *Metrics
	tracer                  trace.Tracer
}

func NewTransferService(
	logger logging.Logger,
	publicParametersManager common.PublicParametersManager[*setup.PublicParams],
	walletService driver.WalletService,
	tokenLoader TokenLoader,
	identityDeserializer driver.Deserializer,
	metrics *Metrics,
	tracerProvider trace.TracerProvider,
	tokenDeserializer TokenDeserializer,
) *TransferService {
	return &TransferService{
		Logger:                  logger,
		PublicParametersManager: publicParametersManager,
		WalletService:           walletService,
		TokenLoader:             tokenLoader,
		IdentityDeserializer:    identityDeserializer,
		Metrics:                 metrics,
		tracer: tracerProvider.Tracer("transfer_service", tracing.WithMetricsOpts(tracing.MetricsOpts{
			Namespace:  "tokensdk_dlog",
			LabelNames: []tracing.LabelName{},
		})),
		TokenDeserializer: tokenDeserializer,
	}
}

// Transfer returns a TransferActionMetadata as a function of the passed arguments
// It also returns the corresponding TransferMetadata
func (s *TransferService) Transfer(ctx context.Context, anchor driver.TokenRequestAnchor, wallet driver.OwnerWallet, ids []*token2.ID, outputs []*token2.Token, opts *driver.TransferOptions) (driver.TransferAction, *driver.TransferMetadata, error) {
	s.Logger.DebugfContext(ctx, "Prepare Transfer Action [%s,%v]", anchor, ids)
	if common.IsAnyNil(ids...) {
		return nil, nil, errors.New("failed to prepare transfer action: nil token id")
	}
	if common.IsAnyNil(outputs...) {
		return nil, nil, errors.New("failed to prepare transfer action: nil output token")
	}

	// load tokens with the passed token identifiers
	loadedTokens, err := s.TokenLoader.LoadTokens(ctx, ids)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to load tokens")
	}
	prepareInputs, err := s.prepareInputs(ctx, loadedTokens)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to prepare inputs")
	}

	var isRedeem bool
	// get sender
	pp := s.PublicParametersManager.PublicParams()
	sender, err := transfer.NewSender(nil, prepareInputs.Tokens(), ids, prepareInputs.Metadata(), pp)
	if err != nil {
		return nil, nil, err
	}
	values := make([]uint64, 0, len(outputs))
	owners := make([][]byte, 0, len(outputs))
	// get values and owners of outputs
	s.Logger.DebugfContext(ctx, "Prepare %d output tokens", len(outputs))
	for i, output := range outputs {
		q, err := token2.ToQuantity(output.Quantity, pp.Precision())
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get value for %dth output", i)
		}
		values = append(values, q.ToBigInt().Uint64())
		owners = append(owners, output.Owner)

		if len(output.Owner) == 0 {
			isRedeem = true
		}
	}
	// produce zkatdlog transfer action
	// return for each output its information in the clear
	start := time.Now()
	s.Logger.DebugfContext(ctx, "Generate zk transfer")
	transfer, outputsMetadata, err := sender.GenerateZKTransfer(ctx, values, owners)
	s.Logger.DebugfContext(ctx, "Done generating zk transfer")
	duration := time.Since(start)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to generate zkatdlog transfer action for txid [%s]", anchor)
	}
	s.Metrics.zkTransferDuration.Observe(float64(duration.Milliseconds()))

	// add transfer action's transferMetadata
	if opts != nil {
		transfer.Metadata = meta.TransferActionMetadata(opts.Attributes)
	}

	// add upgrade witness
	for i, input := range transfer.Inputs {
		input.UpgradeWitness = prepareInputs[i].UpgradeWitness
	}

	// prepare transferMetadata
	ws := s.WalletService

	var transferInputsMetadata []*driver.TransferInputMetadata
	tokens := prepareInputs.Tokens()
	senderAuditInfos := make([][]byte, 0, len(tokens))
	for i, t := range tokens {
		auditInfo, err := s.IdentityDeserializer.GetAuditInfo(ctx, t.Owner, ws)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting audit info for sender identity [%s]", driver.Identity(t.Owner))
		}
		if len(auditInfo) == 0 {
			s.Logger.ErrorfContext(ctx, "empty audit info for the owner [%s] of the i^th token [%s]", ids[i], driver.Identity(t.Owner))
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

	var transferOutputsMetadata []*driver.TransferOutputMetadata
	for i, output := range outputs {
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
			outputAudiInfo, err = s.IdentityDeserializer.GetAuditInfo(ctx, output.Owner, ws)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed getting audit info for sender identity [%s]", driver.Identity(output.Owner))
			}
			recipients, err := s.IdentityDeserializer.Recipients(output.Owner)
			if err != nil {
				return nil, nil, errors.Wrap(err, "failed getting recipients")
			}
			receivers = append(receivers, recipients...)
			for _, receiver := range receivers {
				receiverAudiInfo, err := s.IdentityDeserializer.GetAuditInfo(ctx, receiver, ws)
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

		outputMetadata, err := outputsMetadata[i].Serialize()
		if err != nil {
			return nil, nil, errors.WithMessagef(err, "failed serializing token info for zkatdlog transfer action")
		}

		transferOutputsMetadata = append(transferOutputsMetadata, &driver.TransferOutputMetadata{
			OutputMetadata:  outputMetadata,
			OutputAuditInfo: outputAudiInfo,
			Receivers:       outputReceivers,
		})
	}

	s.Logger.DebugfContext(ctx, "Transfer Action Prepared [id:%s,ins:%d:%d,outs:%d]", anchor, len(ids), len(senderAuditInfos), transfer.NumOutputs())

	transferMetadata := &driver.TransferMetadata{
		Inputs:       transferInputsMetadata,
		Outputs:      transferOutputsMetadata,
		ExtraSigners: nil,
	}

	if isRedeem {
		issuer, err := common.SelectIssuerForRedeem(pp.Issuers(), opts)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to select issuer for redeem")
		}
		transfer.Issuer = issuer
		transferMetadata.Issuer = issuer
	}

	return transfer, transferMetadata, nil
}

// VerifyTransfer checks the outputs in the TransferActionMetadata against the passed metadata
func (s *TransferService) VerifyTransfer(ctx context.Context, action driver.TransferAction, outputMetadata []*driver.TransferOutputMetadata) error {
	if action == nil {
		return errors.New("failed to verify transfer: nil transfer action")
	}
	tr, ok := action.(*transfer.Action)
	if !ok {
		return errors.New("failed to verify transfer: expected *zkatdlog.TransferActionMetadata")
	}

	// get commitments from outputs
	pp := s.PublicParametersManager.PublicParams()
	com := make([]*math.G1, len(tr.Outputs))
	for i := range len(tr.Outputs) {
		com[i] = tr.Outputs[i].Data

		if outputMetadata[i] == nil || len(outputMetadata[i].OutputMetadata) == 0 {
			continue
		}
		// TODO: complete this check...
		// token information in cleartext
		metadata := &token.Metadata{}
		if err := metadata.Deserialize(outputMetadata[i].OutputMetadata); err != nil {
			return errors.Wrap(err, "failed unmarshalling token information")
		}

		// check that token info matches output. If so, return token in cleartext. Else return an error.
		tok, err := tr.Outputs[i].ToClear(metadata, pp)
		if err != nil {
			return errors.Wrap(err, "failed getting token in the clear")
		}
		s.Logger.DebugfContext(ctx, "transfer output [%s,%s,%s]", tok.Type, tok.Quantity, driver.Identity(tok.Owner))
	}

	return transfer.NewVerifier(getTokenData(tr.InputTokens()), com, pp).Verify(tr.Proof)
}

// DeserializeTransferAction un-marshals a TransferActionMetadata from the passed array of bytes.
// DeserializeTransferAction returns an error, if the un-marshalling fails.
func (s *TransferService) DeserializeTransferAction(raw []byte) (driver.TransferAction, error) {
	transferAction := &transfer.Action{}
	err := transferAction.Deserialize(raw)
	if err != nil {
		return nil, err
	}
	return transferAction, nil
}

func (s *TransferService) prepareInputs(ctx context.Context, loadedTokens []LoadedToken) (PreparedTransferInputs, error) {
	preparedInputs := make([]PreparedTransferInput, len(loadedTokens))
	for i, loadedToken := range loadedTokens {
		tok, tokenMetadata, upgradeWitness, err := s.TokenDeserializer.DeserializeToken(ctx, loadedToken.TokenFormat, loadedToken.Token, loadedToken.Metadata)
		if err != nil {
			return nil, errors.Wrapf(err, "failed deserializing token [%s]", string(loadedToken.Token))
		}
		preparedInputs[i] = PreparedTransferInput{
			Token:          tok,
			Metadata:       tokenMetadata,
			Owner:          tok.GetOwner(),
			UpgradeWitness: upgradeWitness,
		}
	}
	return preparedInputs, nil
}

func getTokenData(tokens []*token.Token) []*math.G1 {
	tokenData := make([]*math.G1, len(tokens))
	for i := range tokens {
		tokenData[i] = tokens[i].Data
	}
	return tokenData
}
