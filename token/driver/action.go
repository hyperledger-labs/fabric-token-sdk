/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import "github.com/hyperledger-labs/fabric-token-sdk/token/token"

// SetupAction is the action used to update the public parameters
type SetupAction interface {
	GetSetupParameters() ([]byte, error)
}

//go:generate counterfeiter -o mock/ia.go -fake-name IssueAction . IssueAction

// IssueAction is the action used to issue tokens
type IssueAction interface {
	Action
	ActionWithInputs
	// Serialize returns the serialized version of the action
	Serialize() ([]byte, error)
	// NumOutputs returns the number of outputs of the action
	NumOutputs() int
	// GetSerializedOutputs returns the serialized outputs of the action
	GetSerializedOutputs() ([][]byte, error)
	// GetOutputs returns the outputs of the action
	GetOutputs() []Output
	// IsAnonymous returns true if the issuer is anonymous
	IsAnonymous() bool
	// GetIssuer returns the issuer of the action
	GetIssuer() []byte
	// GetMetadata returns the metadata of the action
	GetMetadata() map[string][]byte
	// IsGraphHiding returns true if the action is graph hiding
	IsGraphHiding() bool
	// ExtraSigners returns the extra signers of the action
	ExtraSigners() []Identity
}

// Input models an input of an action
type Input interface {
	// GetOwner returns the owner of this token
	GetOwner() []byte
}

// Output models an output of an action
type Output interface {
	// Serialize returns the serialized version of the output
	Serialize() ([]byte, error)
	// IsRedeem returns true if the output is a redeem output
	IsRedeem() bool
	// GetOwner returns the owner of this token
	GetOwner() []byte
}

//go:generate counterfeiter -o mock/ta.go -fake-name TransferAction . TransferAction

// TransferAction is the action used to transfer tokens
type TransferAction interface {
	Action
	ActionWithInputs
	// Serialize returns the serialized version of the action
	Serialize() ([]byte, error)
	// NumOutputs returns the number of outputs of the action
	NumOutputs() int
	// GetSerializedOutputs returns the serialized outputs of the action
	GetSerializedOutputs() ([][]byte, error)
	// GetOutputs returns the outputs of the action
	GetOutputs() []Output
	// IsRedeemAt returns true if the output is a redeem output at the passed index
	IsRedeemAt(index int) bool
	// SerializeOutputAt returns the serialized output at the passed index
	SerializeOutputAt(index int) ([]byte, error)
	// IsGraphHiding returns true if the action is graph hiding
	IsGraphHiding() bool
	// GetMetadata returns the action's metadata
	GetMetadata() map[string][]byte
	// GetIssuer returns a non-empty identity of the issuer in case the transfer contains redeeming outputs
	GetIssuer() Identity
}

//go:generate counterfeiter -o mock/action_with_inputs.go -fake-name ActionWithInputs . ActionWithInputs

// ActionWithInputs models an action with inputs
type ActionWithInputs interface {
	// NumInputs returns the number of inputs in the action
	NumInputs() int
	// GetInputs returns the identifiers of the inputs in the action.
	GetInputs() []*token.ID
	// GetSerializedInputs returns the serialized inputs of the action
	GetSerializedInputs() ([][]byte, error)
	// GetSerialNumbers returns the serial numbers of the inputs if this action supports graph hiding
	GetSerialNumbers() []string
	// IsGraphHiding returns true if the action is graph hiding
	IsGraphHiding() bool
	// GetMetadata returns the action's metadata
	GetMetadata() map[string][]byte
	// ExtraSigners returns the extra signers of the action
	ExtraSigners() []Identity
}

type Action interface {
	Validate() error
}
