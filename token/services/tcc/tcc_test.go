/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package tcc_test

import (
	"encoding/base64"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	chaincode2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/tcc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tcc/mock"
)

var _ = Describe("ccvalidator", func() {
	var (
		fakestub      *mock.ChaincodeStubInterface
		chaincode     *chaincode2.TokenChaincode
		fakeValidator *mock.Validator
		fakePPM       *mock.PublicParametersManager
	)
	BeforeEach(func() {
		fakeValidator = &mock.Validator{}
		fakePPM = &mock.PublicParametersManager{}
		chaincode = &chaincode2.TokenChaincode{
			TokenServicesFactory: func(i []byte) (chaincode2.PublicParametersManager, chaincode2.Validator, error) {
				return fakePPM, fakeValidator, nil
			},
		}

		pp := base64.StdEncoding.EncodeToString([]byte("public parameters"))

		fakestub = &mock.ChaincodeStubInterface{}
		fakestub.GetStateReturnsOnCall(0, []byte("public parameters"), nil)
		fakestub.PutStateReturns(nil)
		fakestub.GetArgsReturns([][]byte{[]byte("init"), []byte(pp)})

	})
	Describe("Init", func() {
		Context("when init is called correctly", func() {
			It("Succeeds", func() {
				response := chaincode.Init(fakestub)
				Expect(response).NotTo(BeNil())
				Expect(response.Status).To(Equal(int32(200)))
			})
		})
	})

	Describe("Invoke", func() {
		Describe("Add Auditor", func() {
			BeforeEach(func() {
				var err error
				args := make([][]byte, 2)
				args[0] = []byte("addAuditor")
				args[1] = []byte("auditor")
				Expect(err).NotTo(HaveOccurred())
				fakestub.GetArgsReturns(args)
				fakePPM.SetAuditorReturns([]byte("auditor was added"), nil)

			})
			When("addAuditor is called correctly", func() {
				It("succeeds", func() {
					response := chaincode.Invoke(fakestub)
					Expect(response).NotTo(BeNil())
					Expect(response.Status).To(Equal(int32(200)))
					Expect(response.Payload).To(Equal([]byte("auditor was added")))

				})
			})
			When("addAuditor fails", func() {
				BeforeEach(func() {
					args := make([][]byte, 2)
					args[0] = []byte("addAuditor")
					args[1] = []byte("invalid auditor")
					fakestub.GetArgsReturns(args)
					fakePPM.SetAuditorReturns(nil, errors.New("failed to add auditor"))
				})
				It("fails", func() {
					response := chaincode.Invoke(fakestub)
					Expect(response).NotTo(BeNil())
					Expect(response.Status).To(Equal(int32(500)))
					Expect(response.Message).To(ContainSubstring("failed to add auditor"))

				})
			})
		})
		Describe("addIssuer", func() {
			BeforeEach(func() {
				args := make([][]byte, 2)
				args[0] = []byte("addIssuer")
				args[1] = []byte("issuer")
				fakestub.GetArgsReturns(args)
			})
			Context("Invoke is called correctly to add a new issuer", func() {
				It("succeeds", func() {
					response := chaincode.Invoke(fakestub)
					Expect(response).NotTo(BeNil())
					Expect(response.Status).To(Equal(int32(200)))
					Expect(fakestub.GetStateCallCount()).To(Equal(1))
				})
			})
			Context("chaincode fails to add issuer", func() {
				BeforeEach(func() {
					fakePPM.AddIssuerReturns(nil, errors.Errorf(""))
				})
				It("fails", func() {
					response := chaincode.Invoke(fakestub)
					Expect(response).NotTo(BeNil())
					Expect(response.Status).To(Equal(int32(500)))
					Expect(response.Message).To(ContainSubstring("failed to serialize public parameters"))
					Expect(fakestub.GetStateCallCount()).To(Equal(1))

				})
			})
		})
		Describe("Add Certifier", func() {
			BeforeEach(func() {
				var err error
				args := make([][]byte, 2)
				args[0] = []byte("addCertifier")
				args[1] = []byte("certifier")
				Expect(err).NotTo(HaveOccurred())
				fakestub.GetArgsReturns(args)
				fakePPM.SetCertifierReturns([]byte("certifier was added"), nil)

			})
			When("addAuditor is called correctly", func() {
				It("succeeds", func() {
					response := chaincode.Invoke(fakestub)
					Expect(response).NotTo(BeNil())
					Expect(response.Status).To(Equal(int32(200)))
					Expect(response.Payload).To(Equal([]byte("certifier was added")))

				})
			})
			When("addCertifier fails", func() {
				BeforeEach(func() {
					args := make([][]byte, 2)
					args[0] = []byte("addCertifier")
					args[1] = []byte("invalid certifier")
					fakestub.GetArgsReturns(args)
					fakePPM.SetCertifierReturns(nil, errors.New("flying monkeys"))
				})
				It("fails", func() {
					response := chaincode.Invoke(fakestub)
					Expect(response).NotTo(BeNil())
					Expect(response.Status).To(Equal(int32(500)))
					Expect(response.Message).To(ContainSubstring("flying monkeys"))

				})
			})
		})

		Context("Invoke is called correctly with a token request", func() {
			BeforeEach(func() {
				var err error
				args := make([][]byte, 1)
				args[0] = []byte("invoke")
				Expect(err).NotTo(HaveOccurred())
				fakestub.GetArgsReturns(args)
				fakestub.GetTransientReturns(map[string][]byte{"token_request": []byte("token request")}, nil)
				fakeValidator.UnmarshallAndVerifyReturns([]interface{}{}, nil)
			})
			It("succeeds", func() {
				response := chaincode.Invoke(fakestub)
				Expect(response).NotTo(BeNil())
				Expect(response.Status).To(Equal(int32(200)))
			})
		})

		Context("When VerifyTokenRequest fails", func() {
			BeforeEach(func() {
				var err error
				args := make([][]byte, 1)
				args[0] = []byte("invoke")
				Expect(err).NotTo(HaveOccurred())
				fakestub.GetArgsReturns(args)
				fakestub.GetTransientReturns(map[string][]byte{"token_request": []byte("token request")}, nil)
				fakeValidator.UnmarshallAndVerifyReturns(nil, errors.Errorf("flying monkeys"))
			})
			It("fails", func() {
				response := chaincode.Invoke(fakestub)
				Expect(response).NotTo(BeNil())
				Expect(response.Status).To(Equal(int32(500)))
				Expect(response.Message).To(ContainSubstring("flying monkeys"))
			})
		})

	})
})
