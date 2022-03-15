/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package fabtoken

import (
	"encoding/json"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type TokenInformation struct {
	Issuer []byte
}

func (inf *TokenInformation) Deserialize(b []byte) error {
	return json.Unmarshal(b, inf)
}

func (inf *TokenInformation) Serialize() ([]byte, error) {
	return json.Marshal(inf)
}

type TransferOutput struct {
	Output *token2.Token
}

func (t *TransferOutput) Serialize() ([]byte, error) {
	return json.Marshal(t.Output)
}

func (t *TransferOutput) IsRedeem() bool {
	return len(t.Output.Owner.Raw) == 0
}

type IssueAction struct {
	Issuer  view.Identity
	Outputs []*TransferOutput
}

func (i *IssueAction) Serialize() ([]byte, error) {
	return json.Marshal(i)
}

func (i *IssueAction) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, i)
}

func (i *IssueAction) NumOutputs() int {
	return len(i.Outputs)
}

func (i *IssueAction) GetSerializedOutputs() ([][]byte, error) {
	var res [][]byte
	for _, output := range i.Outputs {
		ser, err := output.Serialize()
		if err != nil {
			return nil, err
		}
		res = append(res, ser)
	}
	return res, nil
}

func (i *IssueAction) GetOutputs() []driver.Output {
	var res []driver.Output
	for _, output := range i.Outputs {
		res = append(res, output)
	}
	return res
}

func (i *IssueAction) IsAnonymous() bool {
	return false
}

func (i *IssueAction) GetIssuer() []byte {
	return i.Issuer
}

func (i *IssueAction) GetMetadata() []byte {
	return nil
}

type TransferAction struct {
	Sender  view.Identity
	Inputs  []string
	Outputs []*TransferOutput
}

func (t *TransferAction) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

func (t *TransferAction) NumOutputs() int {
	return len(t.Outputs)
}

func (t *TransferAction) GetSerializedOutputs() ([][]byte, error) {
	var res [][]byte
	for _, output := range t.Outputs {
		ser, err := output.Serialize()
		if err != nil {
			return nil, err
		}
		res = append(res, ser)
	}
	return res, nil
}

func (t *TransferAction) GetOutputs() []driver.Output {
	var res []driver.Output
	for _, output := range t.Outputs {
		res = append(res, output)
	}
	return res
}

func (t *TransferAction) IsRedeemAt(index int) bool {
	return t.Outputs[index].IsRedeem()
}

func (t *TransferAction) IsGraphHiding() bool {
	return false
}

func (t *TransferAction) SerializeOutputAt(index int) ([]byte, error) {
	return t.Outputs[index].Serialize()
}

func (t *TransferAction) GetInputs() ([]string, error) {
	return t.Inputs, nil
}

func (t *TransferAction) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, t)
}

func (t *TransferAction) GetMetadata() []byte {
	return nil
}
