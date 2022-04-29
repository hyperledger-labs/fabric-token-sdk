/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package tcc_test

import (
	"encoding/base64"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	chaincode2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/tcc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/tcc/mock"
)

var _ = Describe("ccvalidator", func() {
	var (
		fakestub      *mock.ChaincodeStubInterface
		chaincode     *chaincode2.TokenChaincode
		fakeValidator *mock.Validator
		fakePPM       *mock.PublicParametersManager
		ppFile        *os.File
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
		var err error
		ppFile, err = ioutil.TempFile("", "pp")
		Expect(err).NotTo(HaveOccurred())
		_, err = ppFile.Write([]byte(pp))
		Expect(err).NotTo(HaveOccurred())

		fakestub = &mock.ChaincodeStubInterface{}
		err = os.Setenv(chaincode2.PublicParamsPathVarEnv, ppFile.Name())
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.Remove(ppFile.Name())
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
