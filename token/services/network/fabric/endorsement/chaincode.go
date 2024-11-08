/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorsement

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/chaincode"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
)

const InvokeFunction = "invoke"

type ChaincodeEndorsementService struct {
	TMSID token2.TMSID
}

func NewChaincodeEndorsementService(tmsID token2.TMSID) *ChaincodeEndorsementService {
	return &ChaincodeEndorsementService{TMSID: tmsID}
}

func (e *ChaincodeEndorsementService) Endorse(context view.Context, requestRaw []byte, signer view.Identity, txID driver.TxID) (driver.Envelope, error) {
	env, err := chaincode.NewEndorseView(
		e.TMSID.Namespace,
		InvokeFunction,
	).WithNetwork(
		e.TMSID.Network,
	).WithChannel(
		e.TMSID.Channel,
	).WithSignerIdentity(
		signer,
	).WithTransientEntry(
		"token_request", requestRaw,
	).WithTxID(
		fabric.TxID{
			Nonce:   txID.Nonce,
			Creator: txID.Creator,
		},
	).Endorse(context)
	if err != nil {
		return nil, err
	}
	return env, nil
}

func (e *ChaincodeEndorsementService) KeyTranslator() translator.KeyTranslator {
	return &keys.Translator{}
}
