/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"context"
	"time"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/meta"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type TransferService struct {
	Logger                  logging.Logger
	PublicParametersManager common.PublicParametersManager[*crypto.PublicParams]
	WalletService           driver.WalletService
	TokenLoader             TokenLoader
	Deserializer            driver.Deserializer
	Metrics                 *Metrics
	tracer                  trace.Tracer
}

func NewTransferService(
	logger logging.Logger,
	publicParametersManager common.PublicParametersManager[*crypto.PublicParams],
	walletService driver.WalletService,
	tokenLoader TokenLoader,
	deserializer driver.Deserializer,
	metrics *Metrics,
	tracerProvider trace.TracerProvider,
) *TransferService {
	return &TransferService{
		Logger:                  logger,
		PublicParametersManager: publicParametersManager,
		WalletService:           walletService,
		TokenLoader:             tokenLoader,
		Deserializer:            deserializer,
		Metrics:                 metrics,
		tracer: tracerProvider.Tracer("transfer_service", tracing.WithMetricsOpts(tracing.MetricsOpts{
			Namespace:  "tokensdk_dlog",
			LabelNames: []tracing.LabelName{},
		})),
	}
}

// Transfer returns a TransferActionMetadata as a function of the passed arguments
// It also returns the corresponding TransferMetadata
func (s *TransferService) Transfer(ctx context.Context, txID string, wallet driver.OwnerWallet, tokenIDs []*token3.ID, outputTokens []*token3.Token, opts *driver.TransferOptions) (driver.TransferAction, *driver.TransferMetadata, error) {
	newCtx, span := s.tracer.Start(ctx, "transfer")
	defer span.End()
	s.Logger.Debugf("Prepare Transfer Action [%s,%v]", txID, tokenIDs)
	// load tokens with the passed token identifiers
	span.AddEvent("load_tokens")
	tokens, inputInf, senders, err := s.TokenLoader.LoadTokens(newCtx, tokenIDs)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to load tokens")
	}
	pp := s.PublicParametersManager.PublicParams()

	// get sender
	sender, err := transfer.NewSender(nil, tokens, tokenIDs, inputInf, pp)
	if err != nil {
		return nil, nil, err
	}
	var values []uint64
	var owners [][]byte
	var receivers []driver.Identity
	var outputAuditInfos [][]byte

	// get values and owners of outputs
	span.AddEvent("prepare_output_tokens")
	for i, output := range outputTokens {
		q, err := token3.ToQuantity(output.Quantity, pp.Precision())
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get value for %dth output", i)
		}
		values = append(values, q.ToBigInt().Uint64())
		owners = append(owners, output.Owner)
		if len(output.Owner) == 0 { // redeem
			receivers = append(receivers, output.Owner)
			outputAuditInfos = append(outputAuditInfos, []byte{})
			continue
		}
		recipients, err := s.Deserializer.Recipients(output.Owner)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed getting recipients")
		}
		receivers = append(receivers, recipients...)
		auditInfo, err := s.Deserializer.GetOwnerAuditInfo(output.Owner, s.WalletService)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting audit info for sender identity [%s]", driver.Identity(output.Owner).String())
		}
		outputAuditInfos = append(outputAuditInfos, auditInfo...)
	}
	// produce zkatdlog transfer action
	// return for each output its information in the clear
	start := time.Now()
	span.AddEvent("start_generate_zk_transfer")
	zkTransfer, outputMetadata, err := sender.GenerateZKTransfer(newCtx, values, owners)
	span.AddEvent("end_generate_zk_transfer")
	duration := time.Since(start)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to generate zkatdlog transfer action for txid [%s]", txID)
	}
	s.Metrics.zkTransferDuration.Observe(float64(duration.Milliseconds()))

	// add transfer action's metadata
	zkTransfer.Metadata = meta.TransferActionMetadata(opts.Attributes)

	ws := s.WalletService

	// prepare metadata
	var outputsMetadataRaw [][]byte
	for _, information := range outputMetadata {
		raw, err := information.Serialize()
		if err != nil {
			return nil, nil, errors.WithMessage(err, "failed serializing token info for zkatdlog transfer action")
		}
		outputsMetadataRaw = append(outputsMetadataRaw, raw)
	}
	// audit info for receivers
	var receiverAuditInfos [][]byte
	for _, receiver := range receivers {
		if len(receiver) == 0 {
			receiverAuditInfos = append(receiverAuditInfos, []byte{})
			continue
		}
		auditInfo, err := s.Deserializer.GetOwnerAuditInfo(receiver, ws)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting audit info for recipient identity [%s]", receiver.String())
		}
		receiverAuditInfos = append(receiverAuditInfos, auditInfo...)
	}

	// audit info for senders
	var senderAuditInfos [][]byte
	for i, t := range tokens {
		auditInfo, err := s.Deserializer.GetOwnerAuditInfo(t.Owner, ws)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting audit info for sender identity [%s]", driver.Identity(t.Owner).String())
		}
		if len(auditInfo) == 0 {
			s.Logger.Errorf("empty audit info for the owner [%s] of the i^th token [%s]", tokenIDs[i].String(), driver.Identity(t.Owner))
		}
		senderAuditInfos = append(senderAuditInfos, auditInfo...)
	}

	outputs, err := zkTransfer.GetSerializedOutputs()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed getting serialized outputs")
	}

	receiverIsSender := make([]bool, len(receivers))
	for i, receiver := range receivers {
		_, err := ws.OwnerWallet(receiver)
		receiverIsSender[i] = err == nil
	}

	s.Logger.Debugf("Transfer Action Prepared [id:%s,ins:%d:%d,outs:%d]", txID, len(tokenIDs), len(senderAuditInfos), len(outputs))

	metadata := &driver.TransferMetadata{
		TokenIDs:           tokenIDs,
		Senders:            senders,
		SenderAuditInfos:   senderAuditInfos,
		Outputs:            outputs,
		OutputsMetadata:    outputsMetadataRaw,
		OutputAuditInfos:   outputAuditInfos,
		Receivers:          receivers,
		ReceiverAuditInfos: receiverAuditInfos,
		ReceiverIsSender:   receiverIsSender,
	}

	return zkTransfer, metadata, nil
}

// VerifyTransfer checks the outputs in the TransferActionMetadata against the passed metadata
func (s *TransferService) VerifyTransfer(action driver.TransferAction, outputsMetadata [][]byte) error {
	if action == nil {
		return errors.New("failed to verify transfer: nil transfer action")
	}
	tr, ok := action.(*transfer.Action)
	if !ok {
		return errors.New("failed to verify transfer: expected *zkatdlog.TransferActionMetadata")
	}

	// get commitments from outputs
	pp := s.PublicParametersManager.PublicParams()
	com := make([]*math.G1, len(tr.OutputTokens))
	for i := 0; i < len(tr.OutputTokens); i++ {
		com[i] = tr.OutputTokens[i].Data

		if len(outputsMetadata[i]) == 0 {
			continue
		}
		// TODO: complete this check...
		// token information in cleartext
		meta := &token.Metadata{}
		if err := meta.Deserialize(outputsMetadata[i]); err != nil {
			return errors.Wrap(err, "failed unmarshalling token information")
		}

		// check that token info matches output. If so, return token in cleartext. Else return an error.
		tok, err := tr.OutputTokens[i].GetTokenInTheClear(meta, pp)
		if err != nil {
			return errors.Wrap(err, "failed getting token in the clear")
		}
		s.Logger.Debugf("transfer output [%s,%s,%s]", tok.Type, tok.Quantity, driver.Identity(tok.Owner))
	}

	return transfer.NewVerifier(getTokenData(tr.InputTokens), com, pp).Verify(tr.Proof)
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

func getTokenData(tokens []*token.Token) []*math.G1 {
	tokenData := make([]*math.G1, len(tokens))
	for i := 0; i < len(tokens); i++ {
		tokenData[i] = tokens[i].Data
	}
	return tokenData
}
