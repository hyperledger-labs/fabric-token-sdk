/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rp

type ProofType int32

const (
	_ ProofType = iota
	RangeProofType
	CSPRangeProofType
)
