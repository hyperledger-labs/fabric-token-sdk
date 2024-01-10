/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type Info struct {
	// Source is the url of the network where the pledge is supposed to be
	Source    string
	TokenType string
	Amount    uint64
	// TokenID is the ID of the token.
	TokenID       *token2.ID
	TokenMetadata []byte
	Script        *Script
}

func (i *Info) Bytes() ([]byte, error) {
	return json.Marshal(i)
}

func (i *Info) FromBytes(raw []byte) error {
	return json.Unmarshal(raw, i)
}

type DistributePledgeView struct {
	tx *Transaction
}

func NewDistributePledgeInfoView(tx *Transaction) *DistributePledgeView {
	return &DistributePledgeView{
		tx: tx,
	}
}

func (v *DistributePledgeView) Call(context view.Context) (interface{}, error) {
	outputs, err := v.tx.Outputs()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting outputs")
	}
	if outputs.Count() < 1 {
		return nil, errors.WithMessagef(err, "expected at least one output, got [%d]", outputs.Count())
	}
	inputs, err := v.tx.TokenRequest.Inputs()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting inputs")
	}
	if inputs.Count() < 1 {
		return nil, errors.WithMessagef(err, "expected at least one input, got [%d]", inputs.Count())
	}

	var ret []*Info
	for i := 0; i < outputs.Count(); i++ {
		script := outputs.ScriptAt(i)
		if script == nil {
			continue
		}
		output := outputs.At(i)

		tokenType := output.Type
		amount := output.Quantity.ToBigInt().Uint64()

		tokenID := &token2.ID{
			TxId:  v.tx.ID(),
			Index: uint64(i),
		}
		// TODO: retrieve token's metadata

		tmsID := v.tx.TokenService().ID()
		net := network.GetInstance(context, tmsID.Network, tmsID.Channel)
		if net == nil {
			return nil, errors.Errorf("cannot find network for [%s]", tmsID)
		}
		info := &Info{
			Source:        net.InteropURL(tmsID.Namespace),
			TokenType:     tokenType,
			Amount:        amount,
			TokenID:       tokenID,
			TokenMetadata: nil,
			Script:        script,
		}

		session, err := context.GetSession(context.Initiator(), script.Recipient)
		if err != nil {
			return nil, err
		}
		infoRaw, err := info.Bytes()
		if err != nil {
			return nil, errors.Wrapf(err, "failed marshalling pledge info")
		}
		err = session.Send(infoRaw)
		if err != nil {
			return nil, err
		}

		// Wait for a signed ack, but who should sign? What if recipient is an identity that this node does
		// not recognize?
		ret = append(ret, info)
	}

	return ret, nil
}

type pledgeReceiverView struct{}

func ReceivePledgeInfo(context view.Context) (*Info, error) {
	info, err := context.RunView(&pledgeReceiverView{})
	if err != nil {
		return nil, err
	}
	return info.(*Info), nil
}

func (v *pledgeReceiverView) Call(context view.Context) (interface{}, error) {
	_, payload, err := session.ReadFirstMessage(context)
	if err != nil {
		return nil, err
	}
	info := &Info{}
	if err := info.FromBytes(payload); err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling pledge info")
	}

	return info, nil
}

type AcceptPledgeIndoView struct {
	info *Info
}

func NewAcceptPledgeIndoView(info *Info) *AcceptPledgeIndoView {
	return &AcceptPledgeIndoView{
		info: info,
	}
}

func (a *AcceptPledgeIndoView) Call(context view.Context) (interface{}, error) {
	// Store info
	if err := Vault(context).Store(a.info); err != nil {
		return nil, errors.Wrapf(err, "failed storing pledge info")
	}

	// raw, err := a.info.Bytes()
	// if err != nil {
	// 	return nil, errors.Wrapf(err, "failed marshalling info to raw")
	// }

	return nil, nil
}
