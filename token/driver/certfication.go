/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

//go:generate counterfeiter -o mock/cc.go -fake-name CertificationClient . CertificationClient

type CertificationClient interface {
	// IsCertified returns true if the passed token-id has been already certified
	IsCertified(ctx context.Context, id *token.ID) bool
	// RequestCertification requests the certifications of the passed tokens
	RequestCertification(ctx context.Context, ids ...*token.ID) error
}

//go:generate counterfeiter -o mock/cs.go -fake-name CertificationService . CertificationService

type CertificationService interface {
	// NewCertificationRequest creates a new certification request, in a serialized form, for the passed token ids.
	NewCertificationRequest(ids []*token.ID) ([]byte, error)
	// Certify uses the passed wallet to certify the passed token ids.
	// Certify takes in input the certification request and the token representations as available on the ledger.
	Certify(wallet CertifierWallet, ids []*token.ID, tokens [][]byte, request []byte) ([][]byte, error)
	// VerifyCertifications verifies the validity of the certifications of each token indexed by its token-id.
	// The function returns the result of any processing of these certifications.
	// In the simplest case, VerifyCertifications returns the certifications got in input
	VerifyCertifications(ids []*token.ID, certifications [][]byte) ([][]byte, error)
}
