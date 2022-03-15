/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nftcc

// LinearState models a state with a unique identifier that does not change through the evolution
// of the state.
type LinearState interface {
	// SetLinearID assigns the passed id to the state
	SetLinearID(id string) string
}

type AutoLinearState interface {
	GetLinearID() (string, error)
}
