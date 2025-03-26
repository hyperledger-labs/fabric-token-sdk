/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package actions

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// IssueAction encodes a fabtoken Issue
type IssueAction struct {
	// issuer's public key
	Issuer driver.Identity
	// new tokens to be issued
	Outputs []*Output
	// metadata of the issue action
	Metadata map[string][]byte
}

func (i *IssueAction) NumInputs() int {
	return 0
}

func (i *IssueAction) GetInputs() []*token.ID {
	return nil
}

func (i *IssueAction) GetSerializedInputs() ([][]byte, error) {
	return nil, nil
}

func (i *IssueAction) GetSerialNumbers() []string {
	return nil
}

// Serialize marshals IssueAction
func (i *IssueAction) Serialize() ([]byte, error) {
	return json.Marshal(i)
}

// Deserialize un-marshals IssueAction
func (i *IssueAction) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, i)
}

// NumOutputs returns the number of outputs in an IssueAction
func (i *IssueAction) NumOutputs() int {
	return len(i.Outputs)
}

// GetSerializedOutputs returns the serialization of the outputs in an IssueAction
func (i *IssueAction) GetSerializedOutputs() ([][]byte, error) {
	var res [][]byte
	for k, output := range i.Outputs {
		if output == nil {
			return nil, errors.Errorf("cannot serialize issue action outputs: nil output at index [%d]", k)
		}
		ser, err := output.Serialize()
		if err != nil {
			return nil, errors.Errorf("failed to serialize output [%d] in issue action", k)
		}
		res = append(res, ser)
	}
	return res, nil
}

// GetOutputs returns the outputs in an IssueAction
func (i *IssueAction) GetOutputs() []driver.Output {
	var res []driver.Output
	for _, output := range i.Outputs {
		res = append(res, output)
	}
	return res
}

// IsAnonymous returns false, indicating that the identity of issuers in fabtoken
// is revealed during issue
func (i *IssueAction) IsAnonymous() bool {
	return false
}

// GetIssuer returns the issuer encoded in IssueAction
func (i *IssueAction) GetIssuer() []byte {
	return i.Issuer
}

// GetMetadata returns the IssueAction metadata
func (i *IssueAction) GetMetadata() map[string][]byte {
	return i.Metadata
}

// IsGraphHiding returns false, indicating that fabtoken does not hide the transaction graph
func (i *IssueAction) IsGraphHiding() bool {
	return false
}

func (i *IssueAction) Validate() error {
	if i.Issuer.IsNone() {
		return errors.Errorf("issuer is not set")
	}
	if len(i.Outputs) == 0 {
		return errors.Errorf("no outputs in issue action")
	}
	for _, output := range i.Outputs {
		if output == nil {
			return errors.Errorf("nil output in issue action")
		}
	}
	return nil
}

func (i *IssueAction) ExtraSigners() []driver.Identity {
	return nil
}
