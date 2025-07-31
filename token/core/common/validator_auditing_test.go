/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/stretchr/testify/assert"
)

type (
	TestContext = Context[driver.PublicParameters, any, driver.TransferAction, driver.IssueAction, driver.Deserializer]
	TestCheck   = func() bool
)

func TestAuditingSignaturesValidate(t *testing.T) {
	tests := []struct {
		name    string
		err     bool
		errMsg  string
		context func() (*TestContext, TestCheck)
	}{
		{
			name:   "No auditors but token requests with auditor signatures",
			err:    true,
			errMsg: "auditor signatures are not empty",
			context: func() (*TestContext, TestCheck) {
				pp := &mock.PublicParameters{}
				pp.AuditorsReturns(nil)
				return &TestContext{
					PP: pp,
					TokenRequest: &driver.TokenRequest{
						AuditorSignatures: []*driver.AuditorSignature{
							{
								Identity:  driver.Identity("auditor"),
								Signature: []byte("auditor's signature"),
							},
						},
					},
				}, nil
			},
		},
		{
			name: "No auditors and no token requests with auditor signatures",
			err:  false,
			context: func() (*TestContext, TestCheck) {
				pp := &mock.PublicParameters{}
				pp.AuditorsReturns(nil)
				return &TestContext{
					PP:           pp,
					TokenRequest: &driver.TokenRequest{},
				}, nil
			},
		},
		{
			name:   "it is not an auditor",
			err:    true,
			errMsg: "auditor [LERVQYVKJM22xRRnp0G1rmcuYpOTY4x0mWJ5V21ZQ5I=] is not in auditors",
			context: func() (*TestContext, TestCheck) {
				pp := &mock.PublicParameters{}
				pp.AuditorsReturns([]identity.Identity{driver.Identity("auditor1")})
				return &TestContext{
					PP: pp,
					TokenRequest: &driver.TokenRequest{
						AuditorSignatures: []*driver.AuditorSignature{
							{
								Identity:  driver.Identity("auditor2"),
								Signature: []byte("auditor 2's signature"),
							},
						},
					},
				}, nil
			},
		},
		{
			name:   "it is an auditor but I cannot deserialize it",
			err:    true,
			errMsg: "failed to deserialize auditor's public key: auditor deserialize fail",
			context: func() (*TestContext, TestCheck) {
				pp := &mock.PublicParameters{}
				pp.AuditorsReturns([]identity.Identity{driver.Identity("auditor")})

				des := &mock.Deserializer{}
				des.GetAuditorVerifierReturns(nil, errors.Errorf("auditor deserialize fail"))
				return &TestContext{
					PP: pp,
					TokenRequest: &driver.TokenRequest{
						AuditorSignatures: []*driver.AuditorSignature{
							{
								Identity:  driver.Identity("auditor"),
								Signature: []byte("auditor's signature"),
							},
						},
					},
					Deserializer: des,
				}, nil
			},
		},
		{
			name:   "it is an auditor but I cannot verify its signature",
			err:    true,
			errMsg: "failed to verify auditor's signature: signature is not valid",
			context: func() (*TestContext, TestCheck) {
				auditor := driver.Identity("auditor")
				pp := &mock.PublicParameters{}
				pp.AuditorsReturns([]identity.Identity{auditor})
				ver := &mock.Verifier{}
				ver.VerifyReturns(errors.New("signature is not valid"))
				des := &mock.Deserializer{}
				des.GetAuditorVerifierReturns(ver, nil)
				sp := &mock.SignatureProvider{}
				sp.HasBeenSignedByReturns(nil, errors.New("signature is not valid"))
				return &TestContext{
						PP: pp,
						TokenRequest: &driver.TokenRequest{
							AuditorSignatures: []*driver.AuditorSignature{
								{
									Identity:  auditor,
									Signature: []byte("auditor's signature"),
								},
							},
						},
						Deserializer:      des,
						SignatureProvider: sp,
					}, func() bool {
						id, ver2 := sp.HasBeenSignedByArgsForCall(0)
						if ver2 != ver {
							return false
						}
						return auditor.Equal(id)
					}
			},
		},
		{
			name: "it is an auditor and the signature is valid",
			err:  false,
			context: func() (*TestContext, TestCheck) {
				auditor := driver.Identity("auditor")
				pp := &mock.PublicParameters{}
				pp.AuditorsReturns([]identity.Identity{auditor})
				ver := &mock.Verifier{}
				ver.VerifyReturns(errors.New("signature is not valid"))
				des := &mock.Deserializer{}
				des.GetAuditorVerifierReturns(ver, nil)
				sp := &mock.SignatureProvider{}
				sp.HasBeenSignedByReturns(nil, nil)
				return &TestContext{
						PP: pp,
						TokenRequest: &driver.TokenRequest{
							AuditorSignatures: []*driver.AuditorSignature{
								{
									Identity:  auditor,
									Signature: []byte("auditor's signature"),
								},
							},
						},
						Deserializer:      des,
						SignatureProvider: sp,
					}, func() bool {
						id, ver2 := sp.HasBeenSignedByArgsForCall(0)
						if ver2 != ver {
							return false
						}
						return auditor.Equal(id)
					}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, check := tt.context()
			err := AuditingSignaturesValidate(t.Context(), ctx)
			if tt.err {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
			if check != nil {
				assert.True(t, check())
			}
		})
	}
}
