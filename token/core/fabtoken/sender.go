/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

func (s *Service) Transfer(txID string, wallet driver.OwnerWallet, ids []*token2.ID, Outputs []*token2.Token, opts *driver.TransferOptions) (driver.TransferAction, *driver.TransferMetadata, error) {
	id, err := wallet.GetRecipientIdentity()
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed getting sender identity")
	}

	// select inputs
	inputIDs, inputTokens, err := s.TokenLoader.GetTokens(ids)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed loading input tokens")
	}
	var signerIds []view.Identity
	for _, tok := range inputTokens {
		logger.Debugf("Selected output [%s,%s,%s]", tok.Type, tok.Quantity, view.Identity(tok.Owner.Raw))
		signerIds = append(signerIds, tok.Owner.Raw)
	}

	// prepare outputs
	var outs []*TransferOutput
	var infos [][]byte
	for _, output := range Outputs {
		outs = append(outs, &TransferOutput{
			Output: output,
		})
		ti := &TokenInformation{}
		tiRaw, err := ti.Serialize()
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed serializing token information")
		}
		infos = append(infos, tiRaw)
	}

	// assemble transfer action
	transfer := &TransferAction{
		Sender:  id,
		Inputs:  inputIDs,
		Outputs: outs,
	}

	// assemble transfer metadata
	var receivers []view.Identity
	for _, output := range outs {
		receivers = append(receivers, output.Output.Owner.Raw)
	}

	var senderAuditInfos [][]byte
	for _, t := range inputTokens {
		auditInfo, err := s.IP.GetAuditInfo(t.Owner.Raw)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting audit info for sender identity [%s]", view.Identity(t.Owner.Raw).String())
		}
		senderAuditInfos = append(senderAuditInfos, auditInfo)
	}

	var receiverAuditInfos [][]byte
	for _, output := range outs {
		auditInfo, err := s.IP.GetAuditInfo(output.Output.Owner.Raw)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting audit info for recipient identity [%s]", view.Identity(output.Output.Owner.Raw).String())
		}
		receiverAuditInfos = append(receiverAuditInfos, auditInfo)
	}
	outputs, err := transfer.GetSerializedOutputs()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed getting serialized outputs")
	}

	receiverIsSender := make([]bool, len(receivers))
	for i, receiver := range receivers {
		receiverIsSender[i] = s.OwnerWalletByID(receiver) != nil
	}

	metadata := &driver.TransferMetadata{
		Outputs:            outputs,
		Senders:            signerIds,
		SenderAuditInfos:   senderAuditInfos,
		TokenIDs:           ids,
		TokenInfo:          infos,
		Receivers:          receivers,
		ReceiverIsSender:   receiverIsSender,
		ReceiverAuditInfos: receiverAuditInfos,
	}

	logger.Debugf("Transfer metadata: [out:%d, rec:%d]", len(metadata.Outputs), len(metadata.Receivers))

	// done
	return transfer, metadata, nil
}

func (s *Service) VerifyTransfer(tr driver.TransferAction, tokenInfos [][]byte) error {
	// TODO:
	return nil
}

func (s *Service) DeserializeTransferAction(raw []byte) (driver.TransferAction, error) {
	t := &TransferAction{}
	if err := t.Deserialize(raw); err != nil {
		return nil, errors.Wrap(err, "failed deserializing transfer action")
	}
	return t, nil
}
