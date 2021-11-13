/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package token

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Output struct {
	ActionIndex  int
	Owner        view.Identity
	EnrollmentID string
	Type         string
	Quantity     string
}

type Input struct {
	ActionIndex  int
	Id           *token2.ID
	Owner        view.Identity
	EnrollmentID string
	Type         string
	Quantity     string
}

type QueryService interface {
	IsMine(id *token2.ID) (bool, error)
}

type OutputStream struct {
	outputs []*Output
}

func NewOutputStream(outputs []*Output) *OutputStream {
	return &OutputStream{outputs: outputs}
}

func (o *OutputStream) Filter(f func(t *Output) bool) *OutputStream {
	var filtered []*Output
	for _, output := range o.outputs {
		if f(output) {
			filtered = append(filtered, output)
		}
	}
	return &OutputStream{outputs: filtered}
}

func (o *OutputStream) ByRecipient(id view.Identity) *OutputStream {
	return o.Filter(func(t *Output) bool {
		return id.Equal(t.Owner)
	})
}

func (o *OutputStream) ByType(typ string) *OutputStream {
	return o.Filter(func(t *Output) bool {
		return t.Type == typ
	})
}

func (o *OutputStream) Outputs() []*Output {
	return o.outputs
}

func (o *OutputStream) Count() int {
	return len(o.outputs)
}

func (o *OutputStream) Sum() token2.Quantity {
	sum := token2.NewZeroQuantity(64)
	for _, output := range o.outputs {
		q, err := token2.ToQuantity(output.Quantity, 64)
		if err != nil {
			panic(err)
		}

		sum = sum.Add(q)
	}
	return sum
}

func (o *OutputStream) At(i int) *Output {
	return o.outputs[i]
}

func (o *OutputStream) ByEnrollmentID(id string) *OutputStream {
	return o.Filter(func(t *Output) bool {
		return t.EnrollmentID == id
	})
}

func (o *OutputStream) EnrollmentIDs() []string {
	duplicates := map[string]interface{}{}
	var eIDs []string
	for _, output := range o.outputs {
		if len(output.EnrollmentID) == 0 {
			continue
		}
		if _, ok := duplicates[output.EnrollmentID]; !ok {
			eIDs = append(eIDs, output.EnrollmentID)
			duplicates[output.EnrollmentID] = true
		}
	}
	return eIDs
}

func (o *OutputStream) TokenTypes() []string {
	duplicates := map[string]interface{}{}
	var types []string
	for _, output := range o.outputs {
		if _, ok := duplicates[output.Type]; !ok {
			types = append(types, output.Type)
			duplicates[output.Type] = true
		}
	}
	return types
}

type InputStream struct {
	qs     QueryService
	inputs []*Input
}

func NewInputStream(qs QueryService, inputs []*Input) *InputStream {
	return &InputStream{qs: qs, inputs: inputs}
}

func (is *InputStream) Filter(f func(t *Input) bool) *InputStream {
	var filtered []*Input
	for _, item := range is.inputs {
		if f(item) {
			filtered = append(filtered, item)
		}
	}
	return &InputStream{inputs: filtered}
}

func (is *InputStream) Count() int {
	return len(is.inputs)
}

func (is *InputStream) Owners() *OwnerStream {
	ownerMap := map[string]bool{}
	var owners []string
	for _, input := range is.inputs {
		_, ok := ownerMap[input.Owner.UniqueID()]
		if ok {
			continue
		}
		ownerMap[input.Owner.UniqueID()] = true
		owners = append(owners, input.Owner.UniqueID())
	}

	return &OwnerStream{owners: owners}
}

func (is *InputStream) IsAnyMine() bool {
	for _, input := range is.inputs {
		mine, err := is.qs.IsMine(input.Id)
		if err != nil {
			panic(err)
		}
		if mine {
			return true
		}
	}
	return false
}

func (is *InputStream) String() string {
	return fmt.Sprintf("[%v]", is.inputs)
}

func (is *InputStream) At(i int) *Input {
	return is.inputs[i]
}

func (is *InputStream) IDs() []*token2.ID {
	var res []*token2.ID
	for _, input := range is.inputs {
		res = append(res, input.Id)
	}
	return res
}

func (is *InputStream) EnrollmentIDs() []string {
	duplicates := map[string]interface{}{}
	var eIDs []string
	for _, input := range is.inputs {
		if len(input.EnrollmentID) == 0 {
			continue
		}

		_, ok := duplicates[input.EnrollmentID]
		if !ok {
			eIDs = append(eIDs, input.EnrollmentID)
			duplicates[input.EnrollmentID] = true
		}
	}
	return eIDs
}

func (is *InputStream) TokenTypes() []string {
	duplicates := map[string]interface{}{}
	var types []string
	for _, input := range is.inputs {
		_, ok := duplicates[input.Type]
		if !ok {
			types = append(types, input.Type)
			duplicates[input.Type] = true
		}
	}
	return types
}

func (is *InputStream) ByEnrollmentID(id string) *InputStream {
	return is.Filter(func(t *Input) bool {
		return t.EnrollmentID == id
	})
}

func (is *InputStream) ByType(tokenType string) *InputStream {
	return is.Filter(func(t *Input) bool {
		return t.Type == tokenType
	})
}

func (is *InputStream) Sum() token2.Quantity {
	sum := token2.NewZeroQuantity(64)
	for _, input := range is.inputs {
		q, err := token2.ToQuantity(input.Quantity, 64)
		if err != nil {
			panic(err)
		}
		sum = sum.Add(q)
	}
	return sum
}

type OwnerStream struct {
	owners []string
}

func NewOwnerStream(owners []string) *OwnerStream {
	return &OwnerStream{owners: owners}
}

func (s *OwnerStream) Count() int {
	return len(s.owners)
}
