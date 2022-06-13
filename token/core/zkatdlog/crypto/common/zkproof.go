/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package common

type Prover interface {
	Prove() ([]byte, error)
}

type Verifier interface {
	Verify([]byte) error
}
