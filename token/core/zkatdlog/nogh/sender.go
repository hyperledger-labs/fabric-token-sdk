/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

func (s *Service) Transfer(txID string, wallet driver.OwnerWallet, ids []*token3.ID, outputTokens []*token3.Token, opts *driver.TransferOptions) (driver.TransferAction, *driver.TransferMetadata, error) {
	logger.Debugf("Prepare Transfer Action [%s,%v]", txID, ids)

	var signers []driver.Signer
	inputIDs, tokens, inputInf, signerIds, err := s.TokenLoader.LoadTokens(ids)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to load tokens")
	}

	pp := s.PublicParams()
	for _, id := range signerIds {
		// Signer
		si, err := s.identityProvider.GetSigner(id)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting signing identity for id [%v]", id)
		}
		signers = append(signers, si)
	}

	sender, err := transfer.NewSender(signers, tokens, inputIDs, inputInf, pp)
	if err != nil {
		return nil, nil, err
	}
	var values []uint64
	var owners [][]byte
	var ownerIdentities []view.Identity
	for _, output := range outputTokens {
		q, err := token3.ToQuantity(output.Quantity, pp.Precision())
		if err != nil {
			return nil, nil, err
		}
		values = append(values, q.ToBigInt().Uint64())
		owners = append(owners, output.Owner.Raw)
		ownerIdentities = append(ownerIdentities, output.Owner.Raw)
	}
	transfer, infos, err := sender.GenerateZKTransfer(values, owners)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed generating zkat proof for txid [%s]", txID)
	}

	// Prepare metadata
	infoRaws := [][]byte{}
	for _, information := range infos {
		raw, err := information.Serialize()
		if err != nil {
			return nil, nil, errors.WithMessage(err, "failed serializing token info")
		}
		infoRaws = append(infoRaws, raw)
	}

	var receiverAuditInfos [][]byte
	for _, output := range outputTokens {
		auditInfo, err := s.identityProvider.GetAuditInfo(output.Owner.Raw)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting audit info for recipient identity [%s]", view.Identity(output.Owner.Raw).String())
		}
		receiverAuditInfos = append(receiverAuditInfos, auditInfo)
	}

	var senderAuditInfos [][]byte
	for _, t := range tokens {
		auditInfo, err := s.identityProvider.GetAuditInfo(t.Owner)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting audit info for sender identity [%s]", view.Identity(t.Owner).String())
		}
		senderAuditInfos = append(senderAuditInfos, auditInfo)
	}

	outputs, err := transfer.GetSerializedOutputs()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed getting serialized outputs")
	}

	receiverIsSender := make([]bool, len(ownerIdentities))
	for i, receiver := range ownerIdentities {
		receiverIsSender[i] = s.OwnerWalletByID(receiver) != nil
	}

	metadata := &driver.TransferMetadata{
		Outputs:            outputs,
		Senders:            signerIds,
		SenderAuditInfos:   senderAuditInfos,
		TokenIDs:           ids,
		TokenInfo:          infoRaws,
		Receivers:          ownerIdentities,
		ReceiverAuditInfos: receiverAuditInfos,
		ReceiverIsSender:   receiverIsSender,
	}

	return transfer, metadata, nil
}

func (s *Service) VerifyTransfer(action driver.TransferAction, tokenInfos [][]byte) error {
	tr, ok := action.(*transfer.TransferAction)
	if !ok {
		return errors.Errorf("expected *zkatdlog.Transfer")
	}

	// get commitments from outputs
	pp := s.PublicParams()
	com := make([]*math.G1, len(tr.OutputTokens))
	for i := 0; i < len(tr.OutputTokens); i++ {

		ti := &token.TokenInformation{}
		if err := ti.Deserialize(tokenInfos[i]); err != nil {
			return errors.Wrapf(err, "failed unmarshalling token information")
		}

		com[i] = tr.OutputTokens[i].Data
		tok, err := tr.OutputTokens[i].GetTokenInTheClear(ti, pp)
		if err != nil {
			return errors.Wrapf(err, "failed getting token in the clear")
		}
		logger.Debugf("transfer output [%s,%s,%s]", tok.Type, tok.Quantity, view.Identity(tok.Owner.Raw))
	}
	return transfer.NewVerifier(tr.InputCommitments, com, pp).Verify(tr.Proof)
}

func (s *Service) DeserializeTransferAction(raw []byte) (driver.TransferAction, error) {
	transfer := &transfer.TransferAction{}
	err := transfer.Deserialize(raw)
	if err != nil {
		return nil, err
	}
	return transfer, nil
}
