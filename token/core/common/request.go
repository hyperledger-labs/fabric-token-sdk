/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"encoding/asn1"

	"github.com/pkg/errors"

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

func (r *TokenRequest) MarshalToAudit(anchor string, meta *driver.TokenRequestMetadata) ([]byte, error) {
	newReq := &TokenRequest{}
	newReq.Transfers = r.Transfers
	newReq.Issues = r.Issues
	bytes, err := newReq.Bytes()
	if err != nil {
		return nil, errors.Wrapf(err, "audit of tx [%s] failed: error marshal token request for signature", anchor)
	}
	//logger.Debugf("MarshalToAudit [%s][%s]", hash.Hashable(bytes).String(), anchor)
	return append(bytes, []byte(anchor)...), nil
}

func (r *TokenRequest) MarshalTokenRequestToSign(meta *driver.TokenRequestMetadata) ([]byte, error) {
	newReq := &TokenRequest{
		Issues:    r.Issues,
		Transfers: r.Transfers,
	}
	return newReq.Bytes()
}
