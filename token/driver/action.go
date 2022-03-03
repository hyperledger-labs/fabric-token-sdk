/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package driver

type SetupAction interface {
	GetSetupParameters() ([]byte, error)
}

type IssueAction interface {
	Serialize() ([]byte, error)
	NumOutputs() int
	GetSerializedOutputs() ([][]byte, error)
	GetOutputs() []Output
	IsAnonymous() bool
	GetIssuer() []byte
	GetMetadata() []byte
}

type Output interface {
	Serialize() ([]byte, error)
	IsRedeem() bool
}

type TransferAction interface {
	Serialize() ([]byte, error)
	NumOutputs() int
	GetSerializedOutputs() ([][]byte, error)
	GetOutputs() []Output
	IsRedeemAt(index int) bool
	SerializeOutputAt(index int) ([]byte, error)
	GetInputs() ([]string, error)
	IsGraphHiding() bool
	GetMetadata() []byte
}
