/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package translator

import "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

//go:generate counterfeiter -o mock/issuing_validator.go -fake-name IssuingValidator . IssuingValidator

// IssuingValidator is used to establish if the creator can issue tokens of the passed type.
type IssuingValidator interface {
	// Validate returns no error if the passed creator can issue tokens of the passed type,, an error otherwise.
	Validate(creator view.Identity, tokenType string) error
}
