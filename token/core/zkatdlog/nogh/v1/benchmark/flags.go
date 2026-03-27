/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package benchmark

import (
	"flag"
	"fmt"
	"strings"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/rp"
)

var (
	proofType = flag.String("proof_type", "1", "1 or bulletproof or bf, 2 or csp")
)

// ProofType returns the proof type flag value (0 = RangeProof, 1 = CSPRangeProof).
func ProofType() rp.ProofType {
	str := *proofType
	if len(str) == 0 {
		return rp.RangeProofType
	}

	switch strings.ToLower(str) {
	case "1":
		return rp.RangeProofType
	case "bf", "bulletproof":
		return rp.RangeProofType
	case "2":
		return rp.CSPRangeProofType
	case "csp":
		return rp.CSPRangeProofType
	}
	panic(fmt.Errorf("invalid proof_type: %s", str))
}
