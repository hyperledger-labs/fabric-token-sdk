/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package issue

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
)

type Issuer interface {
	GenerateZKIssue(values []uint64, owners [][]byte) (*IssueAction, []*token.Metadata, error)

	SignTokenActions(raw []byte, txID string) ([]byte, error)

	New(ttype string, signer common.SigningIdentity, pp *crypto.PublicParams)
}
