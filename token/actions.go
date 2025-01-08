/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// IssueAction represents an action that issues tokens.
type IssueAction struct {
	a driver.IssueAction
}

// Serialize returns the byte representation of the action.
func (i *IssueAction) Serialize() ([]byte, error) {
	return i.a.Serialize()
}

// NumOutputs returns the number of outputs in the action.
func (i *IssueAction) NumOutputs() int {
	return i.a.NumOutputs()
}

// GetSerializedOutputs returns the serialized outputs of the action.
func (i *IssueAction) GetSerializedOutputs() ([][]byte, error) {
	return i.a.GetSerializedOutputs()
}

// IsAnonymous returns true if the action is an anonymous action.
func (i *IssueAction) IsAnonymous() bool {
	return i.a.IsAnonymous()
}

// GetIssuer returns the issuer of the action.
func (i *IssueAction) GetIssuer() []byte {
	return i.a.GetIssuer()
}

// GetMetadata returns the metadata of the action.
func (i *IssueAction) GetMetadata() map[string][]byte {
	return i.a.GetMetadata()
}

// IsGraphHiding returns true if the action supports graph hiding.
func (i *IssueAction) IsGraphHiding() bool {
	return i.a.IsGraphHiding()
}

// NumInputs returns the number of inputs in the action
func (i *IssueAction) NumInputs() int {
	return i.a.NumInputs()
}

func (i *IssueAction) Validate() error {
	return i.a.Validate()
}

// TransferAction represents an action that transfers tokens.
type TransferAction struct {
	a driver.TransferAction
}

// Serialize returns the byte representation of the action.
func (t *TransferAction) Serialize() ([]byte, error) {
	return t.a.Serialize()
}

// NumOutputs returns the number of outputs in the action.
func (t *TransferAction) NumOutputs() int {
	return t.a.NumOutputs()
}

// GetSerializedOutputs returns the serialized outputs of the action.
func (t *TransferAction) GetSerializedOutputs() ([][]byte, error) {
	return t.a.GetSerializedOutputs()
}

// IsRedeemAt returns true if the i-th output redeems.
func (t *TransferAction) IsRedeemAt(i int) bool {
	return t.a.IsRedeemAt(i)
}

// SerializeOutputAt returns the serialized output at the i-th position.
func (t *TransferAction) SerializeOutputAt(i int) ([]byte, error) {
	return t.a.SerializeOutputAt(i)
}

// GetInputs returns the input ids used in the action.
func (t *TransferAction) GetInputs() []*token.ID {
	return t.a.GetInputs()
}

func (t *TransferAction) GetSerialNumbers() []string {
	return t.a.GetSerialNumbers()
}

// IsGraphHiding returns true if the action supports graph hiding.
func (t *TransferAction) IsGraphHiding() bool {
	return t.a.IsGraphHiding()
}

func (t *TransferAction) Validate() error {
	return t.a.Validate()
}
