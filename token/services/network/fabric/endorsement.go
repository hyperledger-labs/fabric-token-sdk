/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/chaincode"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/fts"
	"github.com/pkg/errors"
)

type Endorsement interface {
	Endorse(context view.Context, tms *token2.ManagementService, requestRaw []byte, signer view.Identity, txID driver.TxID) (driver.Envelope, error)
}

type FSCEndorsement struct {
	Endorsers   []view.Identity
	ViewManager ViewManager
}

func (e *FSCEndorsement) Endorse(context view.Context, tms *token2.ManagementService, requestRaw []byte, signer view.Identity, txID driver.TxID) (driver.Envelope, error) {
	logger.Debugf("request approval via fts endrosers...")
	envBoxed, err := e.ViewManager.InitiateView(&fts.RequestApprovalView{
		TMS:        tms,
		RequestRaw: requestRaw,
		TxID:       txID,
		Endorsers:  e.Endorsers,
	})
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to request approval")
	}
	env, ok := envBoxed.(driver.Envelope)
	if !ok {
		return nil, errors.Errorf("expected driver.Envelope, got [%T]", envBoxed)
	}
	return env, nil
}

type ChaincodeEndorsement struct {
}

func (e *ChaincodeEndorsement) Endorse(context view.Context, tms *token2.ManagementService, requestRaw []byte, signer view.Identity, txID driver.TxID) (driver.Envelope, error) {
	tmsID := tms.ID()
	env, err := chaincode.NewEndorseView(
		tms.Namespace(),
		InvokeFunction,
	).WithNetwork(
		tmsID.Network,
	).WithChannel(
		tmsID.Channel,
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
