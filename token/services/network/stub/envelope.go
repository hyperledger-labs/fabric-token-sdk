/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package stub

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/pkg/errors"
)

type Envelope struct {
	e   *common.Envelope
	upe *UnpackedEnvelope
}

func NewEnvelope() *Envelope {
	return &Envelope{
		e: &common.Envelope{},
	}
}

func NewEnvelopeFromEnv(e *common.Envelope) (*Envelope, error) {
	upe, _, err := UnpackEnvelope(e)
	if err != nil {
		return nil, err
	}
	return &Envelope{
		e:   e,
		upe: upe,
	}, nil
}

func (e *Envelope) TxID() string {
	return e.upe.TxID
}

func (e *Envelope) Nonce() []byte {
	return e.upe.Nonce
}

func (e *Envelope) Creator() []byte {
	return e.upe.Creator
}

func (e *Envelope) Results() []byte {
	return e.upe.Results
}

func (e *Envelope) Bytes() ([]byte, error) {
	return proto.Marshal(e.e)
}

func (e *Envelope) FromBytes(raw []byte) error {
	if err := proto.Unmarshal(raw, e.e); err != nil {
		return err
	}
	upe, _, err := UnpackEnvelope(e.e)
	if err != nil {
		return err
	}
	e.upe = upe
	return nil
}

func (e *Envelope) Envelope() *common.Envelope {
	return e.e
}

func (e *Envelope) String() string {
	s, err := json.MarshalIndent(e.e, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(s)
}

type UnpackedEnvelope struct {
	TxID              string
	Ch                string
	ChaincodeName     string
	ChaincodeVersion  string
	Input             *peer.ChaincodeInput
	Nonce             []byte
	Creator           []byte
	Results           []byte
	Function          string
	Args              []string
	ChannelHeader     *common.ChannelHeader
	SignatureHeader   *common.SignatureHeader
	ProposalResponses []*peer.ProposalResponse
	Envelope          []byte
}

func UnpackEnvelopeFromBytes(raw []byte) (*UnpackedEnvelope, int32, error) {
	env := &common.Envelope{}
	if err := proto.Unmarshal(raw, env); err != nil {
		return nil, -1, err
	}
	return UnpackEnvelope(env)
}

func GetChannelHeaderType(raw []byte) (common.HeaderType, error) {
	env := &common.Envelope{}
	if err := proto.Unmarshal(raw, env); err != nil {
		return -1, err
	}
	payl, err := protoutil.UnmarshalPayload(env.Payload)
	if err != nil {
		return -1, errors.Wrap(err, "failed to unmarshal payload")
	}

	chdr, err := protoutil.UnmarshalChannelHeader(payl.Header.ChannelHeader)
	if err != nil {
		return -1, errors.Wrap(err, "failed to unmarshal channel header")
	}

	return common.HeaderType(chdr.Type), nil
}

func UnpackEnvelope(env *common.Envelope) (*UnpackedEnvelope, int32, error) {
	payl, err := protoutil.UnmarshalPayload(env.Payload)
	if err != nil {
		return nil, -1, errors.Wrap(err, "failed to unmarshal payload")
	}

	chdr, err := protoutil.UnmarshalChannelHeader(payl.Header.ChannelHeader)
	if err != nil {
		return nil, -1, errors.Wrap(err, "failed to unmarshal channel header")
	}

	// validate the payload type
	if common.HeaderType(chdr.Type) != common.HeaderType_ENDORSER_TRANSACTION {
		return nil, chdr.Type, errors.Errorf("only Endorser Transactions are supported, provided type %d", chdr.Type)
	}

	sdr, err := protoutil.UnmarshalSignatureHeader(payl.Header.SignatureHeader)
	if err != nil {
		return nil, chdr.Type, errors.Wrap(err, "failed to unmarshal signature header")
	}

	// ...and the transaction...
	tx, err := protoutil.UnmarshalTransaction(payl.Data)
	if err != nil {
		return nil, chdr.Type, errors.Wrap(err, "VSCC error: GetTransaction failed")
	}

	cap, err := protoutil.UnmarshalChaincodeActionPayload(tx.Actions[0].Payload)
	if err != nil {
		return nil, chdr.Type, errors.Wrap(err, "VSCC error: GetChaincodeActionPayload failed")
	}
	cpp, err := protoutil.UnmarshalChaincodeProposalPayload(cap.ChaincodeProposalPayload)
	if err != nil {
		return nil, chdr.Type, errors.Wrap(err, "VSCC error: GetChaincodeProposalPayload failed")
	}
	cis, err := protoutil.UnmarshalChaincodeInvocationSpec(cpp.Input)
	if err != nil {
		return nil, chdr.Type, errors.Wrap(err, "VSCC error: UnmarshalChaincodeInvocationSpec failed")
	}

	pRespPayload, err := protoutil.UnmarshalProposalResponsePayload(cap.Action.ProposalResponsePayload)
	if err != nil {
		return nil, chdr.Type, errors.Wrap(err, "failed to unmarshal proposal response payload")
	}
	if pRespPayload.Extension == nil {
		return nil, chdr.Type, errors.Wrap(err, "nil pRespPayload.Extension")
	}
	respPayload, err := protoutil.UnmarshalChaincodeAction(pRespPayload.Extension)
	if err != nil {
		return nil, chdr.Type, errors.Wrap(err, "failed to unmarshal chaincode action")
	}

	var args []string
	for i := 1; i < len(cis.ChaincodeSpec.Input.Args); i++ {
		args = append(args, string(cis.ChaincodeSpec.Input.Args[i]))
	}

	var proposalResponses []*peer.ProposalResponse
	for _, endorsement := range cap.Action.Endorsements {
		proposalResponses = append(proposalResponses,
			&peer.ProposalResponse{
				Payload:     cap.Action.ProposalResponsePayload,
				Endorsement: endorsement,
			})
	}

	return &UnpackedEnvelope{
		TxID:              chdr.TxId,
		Ch:                chdr.ChannelId,
		ChaincodeName:     cis.ChaincodeSpec.ChaincodeId.Name,
		ChaincodeVersion:  cis.ChaincodeSpec.ChaincodeId.Version,
		Input:             cis.ChaincodeSpec.Input,
		Nonce:             sdr.Nonce,
		Creator:           sdr.Creator,
		Results:           respPayload.Results,
		Function:          string(cis.ChaincodeSpec.Input.Args[0]),
		Args:              args,
		ChannelHeader:     chdr,
		SignatureHeader:   sdr,
		ProposalResponses: proposalResponses,
	}, chdr.Type, nil
}

func (u *UnpackedEnvelope) ID() string {
	return u.TxID
}

func (u *UnpackedEnvelope) Channel() string {
	return u.Ch
}

func (u *UnpackedEnvelope) FunctionAndParameters() (string, []string) {
	return u.Function, u.Args
}
