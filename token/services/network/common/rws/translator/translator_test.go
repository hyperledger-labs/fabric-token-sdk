/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package translator_test

import (
	"strconv"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

const (
	tokenNameSpace = ttx.TokenNamespace
)

var _ = Describe("Translator", func() {
	var (
		fakeRWSet     *mock.RWSet
		keyTranslator translator.KeyTranslator

		writer *translator.Translator

		fakeissue    *mock.IssueAction
		sn           []string
		faketransfer *mock.TransferAction
	)

	BeforeEach(func() {
		fakeRWSet = &mock.RWSet{}
		keyTranslator = &keys.Translator{}

		writer = translator.New("0", translator.NewRWSetWrapper(fakeRWSet, tokenNameSpace, "0"))

		fakeRWSet.GetStateReturns(nil, nil)
		fakeRWSet.SetStateReturns(nil)

		// fakeIssue
		fakeissue = &mock.IssueAction{}
		// fakeTransfer
		faketransfer = &mock.TransferAction{}
		// serial numbers
		sn = make([]string, 3)
		for i := 0; i < 3; i++ {
			sn[i] = "sn" + strconv.Itoa(i)
		}
	})

	Describe("Issue", func() {
		BeforeEach(func() {
			fakeissue.GetSerializedOutputsReturns([][]byte{[]byte("output-1"), []byte("output-2")}, nil)
			fakeissue.NumOutputsReturns(2)
		})
		When("issue action is valid", func() {
			It("succeeds", func() {
				err := writer.Write(fakeissue)
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeRWSet.SetStateCallCount()).To(Equal(2))

				ns, id, out := fakeRWSet.SetStateArgsForCall(0)
				Expect(ns).To(Equal(tokenNameSpace))
				Expect(out).To(Equal([]byte("output-1")))
				key, err := keyTranslator.CreateTokenKey("0", 0, nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(key))

				ns, id, out = fakeRWSet.SetStateArgsForCall(1)
				Expect(ns).To(Equal(tokenNameSpace))
				Expect(out).To(Equal([]byte("output-2")))

				key, err = keyTranslator.CreateTokenKey("0", 1, nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(key))
			})
		})

		When("created tokens already exist", func() {
			BeforeEach(func() {
				fakeRWSet.GetStateReturnsOnCall(0, []byte("this is already occupied"), nil)
			})
			It("issue fails", func() {
				err := writer.Write(fakeissue)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("token already exists"))
				Expect(fakeRWSet.GetStateCallCount()).To(Equal(1))

			})
		})

		When("created tokens cannot be added", func() {
			BeforeEach(func() {
				fakeRWSet.SetStateReturnsOnCall(1, errors.New("flying monkeys"))
			})
			It("issue fails", func() {
				err := writer.Write(fakeissue)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("flying monkeys"))
				Expect(fakeRWSet.SetStateCallCount()).To(Equal(2))

			})
		})
	})

	Describe("Transfer: transaction graph revealed", func() {
		BeforeEach(func() {
			faketransfer.SerializeOutputAtReturnsOnCall(0, []byte("output-1"), nil)
			faketransfer.IsRedeemAtReturnsOnCall(0, false)
			faketransfer.SerializeOutputAtReturnsOnCall(1, []byte("output-2"), nil)
			faketransfer.IsRedeemAtReturnsOnCall(1, false)
			faketransfer.GetInputsReturns([]*token.ID{{TxId: "key1"}, {TxId: "key2"}, {TxId: "key3"}})
			faketransfer.NumOutputsReturns(2)
			fakeRWSet.GetStateReturnsOnCall(0, []byte("token-1"), nil)
			fakeRWSet.GetStateReturnsOnCall(1, []byte("token-2"), nil)
			fakeRWSet.GetStateReturnsOnCall(2, []byte("token-3"), nil)
		})
		When("transfer is valid", func() {
			It("succeeds", func() {
				err := writer.Write(faketransfer)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeRWSet.SetStateCallCount()).To(Equal(2))

				ns, id, out := fakeRWSet.SetStateArgsForCall(0)
				Expect(ns).To(Equal(tokenNameSpace))
				Expect(out).To(Equal([]byte("output-1")))

				key, err := keyTranslator.CreateTokenKey("0", 0, nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(key))

				ns, id, out = fakeRWSet.SetStateArgsForCall(1)
				Expect(ns).To(Equal(tokenNameSpace))
				Expect(out).To(Equal([]byte("output-2")))

				key, err = keyTranslator.CreateTokenKey("0", 1, nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(key))
			})
		})
		When("created tokens already exist", func() {
			BeforeEach(func() {
				fakeRWSet.GetStateReturnsOnCall(3, []byte("this is already occupied"), nil)
			})
			It("transfer fails", func() {
				err := writer.Write(faketransfer)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("token already exists"))
				Expect(fakeRWSet.GetStateCallCount()).To(Equal(4))

			})
		})
		When("created tokens cannot be added", func() {
			BeforeEach(func() {
				fakeRWSet.SetStateReturnsOnCall(1, errors.New("camel camel"))
			})
			It("transfer fails", func() {
				err := writer.Write(faketransfer)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("camel camel"))
				Expect(fakeRWSet.SetStateCallCount()).To(Equal(2))

			})
		})
		When("input tokens do exist", func() {
			BeforeEach(func() {
				fakeRWSet.GetStateReturnsOnCall(2, nil, nil)
			})
			It("transfer fails", func() {
				err := writer.Write(faketransfer)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid transfer: input must exist: state [tns:\u0000ztoken\u0000key3\u00000\u0000] does not exist for [0]"))
				Expect(fakeRWSet.GetStateCallCount()).To(Equal(3))
			})
		})
	})
	Describe("transfer: transaction graph is hidden", func() {
		BeforeEach(func() {
			faketransfer.SerializeOutputAtReturnsOnCall(0, []byte("output-1"), nil)
			faketransfer.IsRedeemAtReturnsOnCall(0, false)
			faketransfer.SerializeOutputAtReturnsOnCall(1, []byte("output-2"), nil)
			faketransfer.IsRedeemAtReturnsOnCall(1, false)
			fakeRWSet.GetStateReturnsOnCall(0, nil, nil)
			fakeRWSet.GetStateReturnsOnCall(1, nil, nil)
			fakeRWSet.GetStateReturnsOnCall(2, nil, nil)
			faketransfer.GetInputsReturns([]*token.ID{{TxId: "key1"}, {TxId: "key2"}, {TxId: "key3"}})
			faketransfer.GetSerialNumbersReturns(sn)
			faketransfer.NumOutputsReturns(2)
			faketransfer.IsGraphHidingReturns(true)
		})
		When("transfer is valid", func() {
			It("succeeds", func() {
				err := writer.Write(faketransfer)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeRWSet.SetStateCallCount()).To(Equal(5))

				ns, id, out := fakeRWSet.SetStateArgsForCall(0)
				Expect(ns).To(Equal(tokenNameSpace))
				Expect(out).To(Equal([]byte("output-1")))

				key, err := keyTranslator.CreateTokenKey("0", 0, nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(key))

				ns, id, out = fakeRWSet.SetStateArgsForCall(1)
				Expect(ns).To(Equal(tokenNameSpace))
				Expect(out).To(Equal([]byte("output-2")))

				key, err = keyTranslator.CreateTokenKey("0", 1, nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(key))
			})
		})
		When("serial numbers already exist", func() {
			BeforeEach(func() {
				fakeRWSet.GetStateReturnsOnCall(2, []byte(strconv.FormatBool(true)), nil)
			})
			It("transfer fails", func() {
				err := writer.Write(faketransfer)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid transfer: serial number must not exist: state [tns:sn2] already exists for [0]"))
				Expect(fakeRWSet.GetStateCallCount()).To(Equal(3))
				ns, snkey := fakeRWSet.GetStateArgsForCall(2)
				Expect(ns).To(Equal(tokenNameSpace))
				Expect(snkey).To(Equal(sn[2]))
			})
		})
		When("serial numbers cannot be added", func() {
			BeforeEach(func() {
				fakeRWSet.SetStateReturnsOnCall(3, errors.Errorf("flying zebras"))
			})
			It("transfer fails", func() {
				err := writer.Write(faketransfer)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("flying zebras"))
				Expect(err.Error()).To(ContainSubstring("failed to add serial number " + sn[1]))
				Expect(fakeRWSet.SetStateCallCount()).To(Equal(4))
			})
		})
	})

	Describe("Commit Token Request", func() {
		When("set state succeeds", func() {
			It("succeeds", func() {
				_, err := writer.CommitTokenRequest([]byte("token request"), false)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeRWSet.SetStateCallCount()).To(Equal(1))

				ns, id, tr := fakeRWSet.SetStateArgsForCall(0)
				Expect(ns).To(Equal(tokenNameSpace))
				key, err := keyTranslator.CreateTokenRequestKey("0")
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(key))
				Expect(tr).To(Equal([]byte("token request")))
			})
		})
		When("set state fails", func() {
			BeforeEach(func() {
				fakeRWSet.SetStateReturns(errors.New("space monkeys"))
			})
			It("commit token request fails", func() {
				_, err := writer.CommitTokenRequest([]byte("token request"), false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("space monkeys"))
				Expect(fakeRWSet.SetStateCallCount()).To(Equal(1))

			})
		})
		When("get state fails", func() {
			BeforeEach(func() {
				fakeRWSet.GetStateReturns(nil, errors.New("space cheetah"))
			})
			It("commit token request fails", func() {
				_, err := writer.CommitTokenRequest([]byte("token request"), false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("space cheetah"))
				Expect(fakeRWSet.SetStateCallCount()).To(Equal(0))

			})
		})
		When("token request already exists", func() {
			BeforeEach(func() {
				fakeRWSet.GetStateReturns([]byte("occupied"), nil)
			})
			It("commit token request fails", func() {
				_, err := writer.CommitTokenRequest([]byte("token request"), false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to read token request: state [tns:\u0000ztoken\u0000token_request\u00000\u0000] already exists for [0]"))
				Expect(fakeRWSet.SetStateCallCount()).To(Equal(0))

			})
		})
	})
})
