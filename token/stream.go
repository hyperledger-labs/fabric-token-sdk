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

type QueryService interface {
	IsMine(id *token2.ID) (bool, error)
}

// Output models the output of a token action
type Output struct {
	ActionIndex  int
	Owner        view.Identity
	EnrollmentID string
	Type         string
	Quantity     token2.Quantity
}

// Input models an input of a token action
type Input struct {
	ActionIndex  int
	Id           *token2.ID
	Owner        view.Identity
	EnrollmentID string
	Type         string
	Quantity     token2.Quantity
}

// OutputStream models a stream over a set of outputs (Output).
type OutputStream struct {
	Precision uint64
	outputs   []*Output
}

// NewOutputStream creates a new OutputStream for the passed outputs and Precision.
func NewOutputStream(outputs []*Output, precision uint64) *OutputStream {
	return &OutputStream{outputs: outputs, Precision: precision}
}

// Filter filters the OutputStream to only include outputs that match the passed predicate.
func (o *OutputStream) Filter(f func(t *Output) bool) *OutputStream {
	var filtered []*Output
	for _, output := range o.outputs {
		if f(output) {
			filtered = append(filtered, output)
		}
	}
	return &OutputStream{outputs: filtered, Precision: o.Precision}
}

// ByRecipient filters the OutputStream to only include outputs that match the passed recipient.
func (o *OutputStream) ByRecipient(id view.Identity) *OutputStream {
	return o.Filter(func(t *Output) bool {
		return id.Equal(t.Owner)
	})
}

// ByType filters the OutputStream to only include outputs that match the passed type.
func (o *OutputStream) ByType(typ string) *OutputStream {
	return o.Filter(func(t *Output) bool {
		return t.Type == typ
	})
}

// Outputs returns the outputs of the OutputStream.
func (o *OutputStream) Outputs() []*Output {
	return o.outputs
}

// Count returns the number of outputs in the OutputStream.
func (o *OutputStream) Count() int {
	return len(o.outputs)
}

// Sum returns the sum of the quantity of all outputs in the OutputStream.
func (o *OutputStream) Sum() token2.Quantity {
	sum := token2.NewZeroQuantity(o.Precision)
	for _, output := range o.outputs {
		sum = sum.Add(output.Quantity)
	}
	return sum
}

// At returns the output at the passed index.
func (o *OutputStream) At(i int) *Output {
	return o.outputs[i]
}

// ByEnrollmentID filters to only include outputs that match the passed enrollment ID.
func (o *OutputStream) ByEnrollmentID(id string) *OutputStream {
	return o.Filter(func(t *Output) bool {
		return t.EnrollmentID == id
	})
}

// EnrollmentIDs returns the enrollment IDs of the outputs in the OutputStream.
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

// TokenTypes returns the token types of the outputs in the OutputStream.
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

// InputStream models a stream over a set of inputs (Input).
type InputStream struct {
	qs        QueryService
	inputs    []*Input
	precision uint64
}

// NewInputStream creates a new InputStream for the passed inputs and query service.
func NewInputStream(qs QueryService, inputs []*Input, precision uint64) *InputStream {
	return &InputStream{qs: qs, inputs: inputs, precision: precision}
}

// Filter returns a new InputStream with only the inputs that satisfy the predicate
func (is *InputStream) Filter(f func(t *Input) bool) *InputStream {
	var filtered []*Input
	for _, item := range is.inputs {
		if f(item) {
			filtered = append(filtered, item)
		}
	}
	return &InputStream{inputs: filtered, precision: is.precision}
}

// Count returns the number of inputs in the stream
func (is *InputStream) Count() int {
	return len(is.inputs)
}

// Owners returns a list of identities that own the tokens in the stream
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

// IsAnyMine returns true if any of the inputs are mine
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

// String returns a string representation of the input stream
func (is *InputStream) String() string {
	return fmt.Sprintf("[%v]", is.inputs)
}

// At returns the input at the given index.
func (is *InputStream) At(i int) *Input {
	return is.inputs[i]
}

// IDs returns the IDs of the inputs.
func (is *InputStream) IDs() []*token2.ID {
	var res []*token2.ID
	for _, input := range is.inputs {
		res = append(res, input.Id)
	}
	return res
}

// EnrollmentIDs returns the enrollment IDs of the owners of the inputs.
// It might be empty, if not available.
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

// TokenTypes returns the token types of the inputs.
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

// ByEnrollmentID filters by enrollment ID.
func (is *InputStream) ByEnrollmentID(id string) *InputStream {
	return is.Filter(func(t *Input) bool {
		return t.EnrollmentID == id
	})
}

// ByType filters by token type.
func (is *InputStream) ByType(tokenType string) *InputStream {
	return is.Filter(func(t *Input) bool {
		return t.Type == tokenType
	})
}

// Sum returns the sum of the quantities of the inputs.
func (is *InputStream) Sum() token2.Quantity {
	sum := token2.NewZeroQuantity(is.precision)
	for _, input := range is.inputs {
		sum = sum.Add(input.Quantity)
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
