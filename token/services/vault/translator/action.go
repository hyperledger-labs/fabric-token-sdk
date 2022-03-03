/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package translator

type SetupAction interface {
	GetSetupParameters() ([]byte, error)
}

//go:generate counterfeiter -o mock/issue_action.go -fake-name IssueAction . IssueAction

type IssueAction interface {
	Serialize() ([]byte, error)
	NumOutputs() int
	GetSerializedOutputs() ([][]byte, error)
	IsAnonymous() bool
	GetIssuer() []byte
	GetMetadata() []byte
}

//go:generate counterfeiter -o mock/transfer_action.go -fake-name TransferAction . TransferAction

type TransferAction interface {
	Serialize() ([]byte, error)
	NumOutputs() int
	GetSerializedOutputs() ([][]byte, error)
	IsRedeemAt(index int) bool
	SerializeOutputAt(index int) ([]byte, error)
	GetInputs() ([]string, error)
	IsGraphHiding() bool
	GetMetadata() []byte
}

type Signature interface {
	Metadata() map[string][]byte
}
