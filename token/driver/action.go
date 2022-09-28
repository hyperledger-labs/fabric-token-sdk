/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

// SetupAction is the action used to update the public parameters
type SetupAction interface {
	GetSetupParameters() ([]byte, error)
}

// IssueAction is the action used to issue tokens
type IssueAction interface {
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
}

// Output models an output of an action
type Output interface {
	// Serialize returns the serialized version of the output
	Serialize() ([]byte, error)
	// IsRedeem returns true if the output is a redeem output
	IsRedeem() bool
}

// TransferAction is the action used to transfer tokens
type TransferAction interface {
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
	// GetInputs returns the identifiers of the inputs in the action.
	GetInputs() ([]string, error)
	// IsGraphHiding returns true if the action is graph hiding
	IsGraphHiding() bool
	// GetMetadata returns the action's metadata
	GetMetadata() map[string][]byte
}
