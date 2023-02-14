/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// Transfer returns a TransferAction as a function of the passed arguments
// It also returns the corresponding TransferMetadata
func (s *Service) Transfer(txID string, wallet driver.OwnerWallet, ids []*token3.ID, outputTokens []*token3.Token, opts *driver.TransferOptions) (driver.TransferAction, *driver.TransferMetadata, error) {
	logger.Debugf("Prepare Transfer Action [%s,%v]", txID, ids)
	var signers []driver.Signer
	// load tokens with the passed token identifiers
	inputIDs, tokens, inputInf, signerIds, err := s.TokenLoader.LoadTokens(ids)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to load tokens")
	}
	pp := s.PublicParams()
	if pp == nil {
		return nil, nil, errors.Errorf("public parameters not inizialized")
	}
	for _, id := range signerIds {
		// get signers for each input token
		si, err := s.identityProvider.GetSigner(id)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting signing identity for id [%v]", id)
		}
		signers = append(signers, si)
	}

	// get sender
	sender, err := transfer.NewSender(signers, tokens, inputIDs, inputInf, pp)
	if err != nil {
		return nil, nil, err
	}
	var values []uint64
	var owners [][]byte
	var ownerIdentities []view.Identity
	// get values and owners of outputs
	for i, output := range outputTokens {
		q, err := token3.ToQuantity(output.Quantity, pp.Precision())
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get value for %dth output", i)
		}
		values = append(values, q.ToBigInt().Uint64())
		if output.Owner == nil {
			return nil, nil, errors.Errorf("failed to get owner for %dth output: nil owner", i)
		}
		owners = append(owners, output.Owner.Raw)
		if len(output.Owner.Raw) == 0 { // redeem
			ownerIdentities = append(ownerIdentities, output.Owner.Raw)
			continue
		}
		owner, err := identity.UnmarshallRawOwner(output.Owner.Raw)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to unmarshal owner of the output token")
		}
		if owner.Type == identity.SerializedIdentityType {
			ownerIdentities = append(ownerIdentities, output.Owner.Raw)
			continue
		}
		_, recipient, err := htlc.GetScriptSenderAndRecipient(owner)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed getting script sender and recipient")
		}
		ownerIdentities = append(ownerIdentities, recipient)
	}
	// produce zkatdlog transfer action
	// return for each output its information in the clear
	transfer, outputMetadata, err := sender.GenerateZKTransfer(values, owners)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to generate zkatdlog transfer action for txid [%s]", txID)
	}

	// add transfer action's metadata
	common.SetTransferActionMetadata(opts.Attributes, transfer.Metadata)

	// prepare metadata
	var outputMetadataRaw [][]byte
	for _, information := range outputMetadata {
		raw, err := information.Serialize()
		if err != nil {
			return nil, nil, errors.WithMessage(err, "failed serializing token info for zkatdlog transfer action")
		}
		outputMetadataRaw = append(outputMetadataRaw, raw)
	}
	// audit info for receivers
	var receiverAuditInfos [][]byte
	for _, output := range outputTokens {
		auditInfo, err := htlc.GetOwnerAuditInfo(output.Owner.Raw, s)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting audit info for recipient identity [%s]", view.Identity(output.Owner.Raw).String())
		}
		receiverAuditInfos = append(receiverAuditInfos, auditInfo)
	}

	// audit info for senders
	var senderAuditInfos [][]byte
	for i, t := range tokens {
		auditInfo, err := htlc.GetOwnerAuditInfo(t.Owner, s)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting audit info for sender identity [%s]", view.Identity(t.Owner).String())
		}
		if len(auditInfo) == 0 {
			logger.Errorf("empty audit info for the owner [%s] of the i^th token [%s]", ids[i].String(), view.Identity(t.Owner))
		}
		senderAuditInfos = append(senderAuditInfos, auditInfo)
	}

	outputs, err := transfer.GetSerializedOutputs()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed getting serialized outputs")
	}

	receiverIsSender := make([]bool, len(ownerIdentities))
	for i, receiver := range ownerIdentities {
		_, err := s.OwnerWalletByID(receiver)
		receiverIsSender[i] = err == nil
	}

	logger.Debugf("Transfer Action Prepared [id:%s,ins:%d:%d,outs:%d]", txID, len(ids), len(senderAuditInfos), len(outputs))

	metadata := &driver.TransferMetadata{
		Outputs:            outputs,
		Senders:            signerIds,
		SenderAuditInfos:   senderAuditInfos,
		TokenIDs:           ids,
		OutputsMetadata:    outputMetadataRaw,
		Receivers:          ownerIdentities,
		ReceiverAuditInfos: receiverAuditInfos,
		ReceiverIsSender:   receiverIsSender,
	}

	return transfer, metadata, nil
}

// VerifyTransfer checks the outputs in the TransferAction against the passed metadata
func (s *Service) VerifyTransfer(action driver.TransferAction, outputsMetadata [][]byte) error {
	if action == nil {
		return errors.New("failed to verify transfer: nil transfer action")
	}
	tr, ok := action.(*transfer.TransferAction)
	if !ok {
		return errors.New("failed to verify transfer: expected *zkatdlog.TransferAction")
	}

	// get commitments from outputs
	pp := s.PublicParams()
	if pp == nil {
		return errors.Errorf("public parameters not inizialized")
	}
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
		logger.Debugf("transfer output [%s,%s,%s]", tok.Type, tok.Quantity, view.Identity(tok.Owner.Raw))
	}

	return transfer.NewVerifier(tr.InputCommitments, com, pp).Verify(tr.Proof)
}

// DeserializeTransferAction un-marshals a TransferAction from the passed array of bytes.
// DeserializeTransferAction returns an error, if the un-marshalling fails.
func (s *Service) DeserializeTransferAction(raw []byte) (driver.TransferAction, error) {
	transfer := &transfer.TransferAction{}
	err := transfer.Deserialize(raw)
	if err != nil {
		return nil, err
	}
	return transfer, nil
}
