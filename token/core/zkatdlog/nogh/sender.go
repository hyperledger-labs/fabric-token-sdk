/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package nogh

import (
	"strconv"

	"github.com/IBM/mathlib"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

func (s *Service) Transfer(txID string, wallet driver.OwnerWallet, ids []*token3.Id, outputTokens ...*token3.Token) (driver.TransferAction, *driver.TransferMetadata, error) {
	logger.Debugf("Prepare Transfer Action [%s,%v]", txID, ids)

	var tokens []*token.Token
	var inputIDs []string
	var inputInf []*token.TokenInformation
	var signers []driver.Signer
	var signerIds []view.Identity

	qe, err := s.Channel.Vault().NewQueryExecutor()
	if err != nil {
		return nil, nil, err
	}
	defer qe.Done()

	pp := s.PublicParams()
	for _, id := range ids {
		// Token Info
		outputID, err := keys.CreateFabtokenKey(id.TxId, int(id.Index))
		if err != nil {
			return nil, nil, errors.Wrapf(err, "error creating output ID: %v", id)
		}
		meta, _, _, err := qe.GetStateMetadata(s.Namespace, outputID)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting metadata for id [%v]", id)
		}
		ti := &token.TokenInformation{}
		err = ti.Deserialize(meta[info])
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed deserializeing token info for id [%v]", id)
		}

		// Token and InputID
		outputID, err = keys.CreateTokenKey(id.TxId, int(id.Index))
		if err != nil {
			return nil, nil, errors.Wrapf(err, "error creating output ID: %v", id)
		}
		val, err := qe.GetState(s.Namespace, outputID)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting state [%s]", outputID)
		}

		logger.Debugf("loaded transfer input [%s]", hash.Hashable(val).String())
		token := &token.Token{}
		err = token.Deserialize(val)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed unmarshalling token for id [%v]", id)
		}
		tok, err := token.GetTokenInTheClear(ti, pp)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "invalid token, cannot get it in clear [%v]", id)
		}
		logger.Debugf("Selected output [%s,%s,%s]", tok.Type, tok.Quantity, view.Identity(tok.Owner.Raw))

		// Signer
		si, err := s.identityProvider.GetSigner(token.Owner)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting signing identity for id [%v]", id)
		}

		inputIDs = append(inputIDs, outputID)
		tokens = append(tokens, token)
		inputInf = append(inputInf, ti)
		signers = append(signers, si)
		signerIds = append(signerIds, token.Owner)
	}

	sender, err := transfer.NewSender(signers, tokens, inputIDs, inputInf, pp)
	if err != nil {
		return nil, nil, err
	}
	var values []uint64
	var owners [][]byte
	var ownerIdentities []view.Identity
	for _, output := range outputTokens {
		q, err := token3.ToQuantity(output.Quantity, keys.Precision)
		if err != nil {
			return nil, nil, err
		}
		v, err := strconv.ParseUint(q.Decimal(), 10, 64)
		if err != nil {
			return nil, nil, err
		}
		values = append(values, v)
		owners = append(owners, output.Owner.Raw)

		// add owner identity if not present already
		found := false
		for _, identity := range ownerIdentities {
			if identity.Equal(output.Owner.Raw) {
				found = true
				break
			}
		}
		if !found {
			ownerIdentities = append(ownerIdentities, output.Owner.Raw)
		}
	}
	transfer, infos, err := sender.GenerateZKTransfer(values, owners)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed generating zkat proof for txid [%s]", txID)
	}


	var tis [][]byte
	for _, info := range infos {
		rawTI, err := info.Serialize()
		if err != nil {
			panic(err)
		}
		tis = append(tis, rawTI)
	}

	err = s.VerifyTransfer(transfer, tis)
	if err != nil {
		panic(err)
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
		auditInfo, err := view2.GetSigService(s.SP).GetAuditInfo(output.Owner.Raw)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting audit info for recipient identity [%s]", view.Identity(output.Owner.Raw).String())
		}
		receiverAuditInfos = append(receiverAuditInfos, auditInfo)
	}

	var senderAuditInfos [][]byte
	for _, t := range tokens {
		auditInfo, err := view2.GetSigService(s.SP).GetAuditInfo(t.Owner)
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
