/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package token

import "github.com/hyperledger-labs/fabric-token-sdk/token/driver"

type IssueAction struct {
	a driver.IssueAction
}

func (i *IssueAction) Serialize() ([]byte, error) {
	return i.a.Serialize()
}

func (i *IssueAction) NumOutputs() int {
	return i.a.NumOutputs()
}

func (i *IssueAction) GetSerializedOutputs() ([][]byte, error) {
	return i.a.GetSerializedOutputs()
}

func (i *IssueAction) IsAnonymous() bool {
	return i.a.IsAnonymous()
}

func (i *IssueAction) GetIssuer() []byte {
	return i.a.GetIssuer()
}

func (i *IssueAction) GetMetadata() []byte {
	return i.a.GetMetadata()
}

type TransferAction struct {
	a driver.TransferAction
}

func (t *TransferAction) Serialize() ([]byte, error) {
	return t.a.Serialize()
}

func (t *TransferAction) NumOutputs() int {
	return t.a.NumOutputs()
}

func (t *TransferAction) GetSerializedOutputs() ([][]byte, error) {
	return t.a.GetSerializedOutputs()
}

func (t *TransferAction) IsRedeemAt(index int) bool {
	return t.a.IsRedeemAt(index)
}

func (t *TransferAction) SerializeOutputAt(index int) ([]byte, error) {
	return t.a.SerializeOutputAt(index)
}

func (t *TransferAction) GetInputs() ([]string, error) {
	return t.a.GetInputs()
}

func (t *TransferAction) IsGraphHiding() bool {
	return t.a.IsGraphHiding()
}

func (t *TransferAction) GetMetadata() []byte {
	return t.a.GetMetadata()
}
