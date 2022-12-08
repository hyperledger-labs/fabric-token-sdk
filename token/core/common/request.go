/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"encoding/asn1"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type TokenRequest struct {
	Issues            [][]byte
	Transfers         [][]byte
	Signatures        [][]byte
	AuditorSignatures [][]byte
}

func (r *TokenRequest) AppendIssue(raw []byte) {
	r.Issues = append(r.Issues, raw)
}

func (r *TokenRequest) AppendTransfer(raw []byte) {
	r.Transfers = append(r.Transfers, raw)
}

func (r *TokenRequest) AppendAuditorSignature(sigma []byte) {
	r.AuditorSignatures = append(r.AuditorSignatures, sigma)
}

func (r *TokenRequest) AppendSignature(sigma []byte) {
	r.Signatures = append(r.Signatures, sigma)
}

func (r *TokenRequest) Import(request driver.TokenRequest) {
	tr := request.(*TokenRequest)
	r.Issues = append(r.Issues, tr.Issues...)
	r.Transfers = append(r.Transfers, tr.Transfers...)
}

func (r *TokenRequest) GetTransfers() [][]byte {
	return r.Transfers
}

func (r *TokenRequest) GetIssues() [][]byte {
	return r.Issues
}

func (r *TokenRequest) Bytes() ([]byte, error) {
	return asn1.Marshal(*r)
}

func (r *TokenRequest) FromBytes(raw []byte) error {
	_, err := asn1.Unmarshal(raw, r)
	return err
}
