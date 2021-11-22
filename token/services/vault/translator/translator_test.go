/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package translator_test

import (
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	writer2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/translator/mock"
)

const (
	tokenNameSpace = "zkat"
	action         = "action"
	actionIssue    = "issue"
	actionTransfer = "transfer"
)

var _ = Describe("Translator", func() {
	var (
		fakeIssuingValidator *mock.IssuingValidator
		fakeRWSet            *mock.RWSet

		writer *writer2.Translator

		fakeissue    *mock.IssueAction
		sn           []string
		faketransfer *mock.TransferAction
	)

	BeforeEach(func() {
		fakeIssuingValidator = &mock.IssuingValidator{}
		fakeRWSet = &mock.RWSet{}

		writer = writer2.New(fakeIssuingValidator, "0", fakeRWSet, "zkat")

		fakeRWSet.GetStateReturns(nil, nil)
		fakeRWSet.SetStateReturns(nil)
		fakeRWSet.SetStateMetadataReturns(nil)

		fakeIssuingValidator.ValidateReturns(nil)

		// fakeIssue
		fakeissue = &mock.IssueAction{}
		// fakeTransfer
		faketransfer = &mock.TransferAction{}
		// serial numbers
		var err error
		sn = make([]string, 3)
		for i := 0; i < 3; i++ {
			sn[i], err = keys.CreateSNKey("sn" + strconv.Itoa(i))
			Expect(err).NotTo(HaveOccurred())
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
				Expect(fakeRWSet.SetStateMetadataCallCount()).To(Equal(2))

				ns, id, out := fakeRWSet.SetStateArgsForCall(0)
				Expect(ns).To(Equal(tokenNameSpace))
				Expect(out).To(Equal([]byte("output-1")))
				key, err := keys.CreateTokenKey("0", 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(key))

				ns, id, metadata := fakeRWSet.SetStateMetadataArgsForCall(0)
				Expect(ns).To(Equal(tokenNameSpace))
				Expect(id).To(Equal(key))
				Expect(metadata).To(Equal(map[string][]byte{action: []byte(actionIssue)}))

				ns, id, out = fakeRWSet.SetStateArgsForCall(1)
				Expect(ns).To(Equal(tokenNameSpace))
				Expect(out).To(Equal([]byte("output-2")))

				key, err = keys.CreateTokenKey("0", 1)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(key))

				ns, id, metadata = fakeRWSet.SetStateMetadataArgsForCall(1)
				Expect(ns).To(Equal(tokenNameSpace))
				Expect(id).To(Equal(key))
				Expect(metadata).To(Equal(map[string][]byte{action: []byte(actionIssue)}))

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

		When("issuer is not allowed to issue tokens", func() {
			BeforeEach(func() {
				fakeIssuingValidator.ValidateReturnsOnCall(0, errors.New("wild banana"))
			})
			It("issue fails", func() {
				err := writer.Write(fakeissue)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("verification of issue policy failed"))
			})
		})

	})
	Describe("Transfer: transaction graph revealed", func() {
		BeforeEach(func() {
			faketransfer.SerializeOutputAtReturnsOnCall(0, []byte("output-1"), nil)
			faketransfer.IsRedeemAtReturnsOnCall(0, false)
			faketransfer.SerializeOutputAtReturnsOnCall(1, []byte("output-2"), nil)
			faketransfer.IsRedeemAtReturnsOnCall(1, false)
			faketransfer.GetInputsReturns([]string{"key1", "key2", "key3"}, nil)
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
				Expect(fakeRWSet.SetStateMetadataCallCount()).To(Equal(5))

				ns, id, out := fakeRWSet.SetStateArgsForCall(0)
				Expect(ns).To(Equal(tokenNameSpace))
				Expect(out).To(Equal([]byte("output-1")))

				key, err := keys.CreateTokenKey("0", 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(key))

				ns, id, metadata := fakeRWSet.SetStateMetadataArgsForCall(0)
				Expect(ns).To(Equal(tokenNameSpace))
				Expect(id).To(Equal(key))
				Expect(metadata).To(Equal(map[string][]byte{action: []byte(actionTransfer)}))

				ns, id, out = fakeRWSet.SetStateArgsForCall(1)
				Expect(ns).To(Equal(tokenNameSpace))
				Expect(out).To(Equal([]byte("output-2")))

				key, err = keys.CreateTokenKey("0", 1)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(key))

				ns, id, metadata = fakeRWSet.SetStateMetadataArgsForCall(1)
				Expect(ns).To(Equal(tokenNameSpace))
				Expect(id).To(Equal(key))
				Expect(metadata).To(Equal(map[string][]byte{action: []byte(actionTransfer)}))

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
				Expect(err.Error()).To(ContainSubstring("already spent"))
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
			faketransfer.GetInputsReturns(sn, nil)
			faketransfer.NumOutputsReturns(2)
			faketransfer.IsGraphHidingReturns(true)
		})
		When("transfer is valid", func() {
			It("succeeds", func() {
				err := writer.Write(faketransfer)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeRWSet.SetStateCallCount()).To(Equal(5))
				Expect(fakeRWSet.SetStateMetadataCallCount()).To(Equal(2))

				ns, id, out := fakeRWSet.SetStateArgsForCall(0)
				Expect(ns).To(Equal(tokenNameSpace))
				Expect(out).To(Equal([]byte("output-1")))

				key, err := keys.CreateTokenKey("0", 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(key))

				ns, id, metadata := fakeRWSet.SetStateMetadataArgsForCall(0)
				Expect(ns).To(Equal(tokenNameSpace))
				Expect(id).To(Equal(key))
				Expect(metadata).To(Equal(map[string][]byte{action: []byte(actionTransfer)}))

				ns, id, out = fakeRWSet.SetStateArgsForCall(1)
				Expect(ns).To(Equal(tokenNameSpace))
				Expect(out).To(Equal([]byte("output-2")))

				key, err = keys.CreateTokenKey("0", 1)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(key))

				ns, id, metadata = fakeRWSet.SetStateMetadataArgsForCall(1)
				Expect(ns).To(Equal(tokenNameSpace))
				Expect(id).To(Equal(key))
				Expect(metadata).To(Equal(map[string][]byte{action: []byte(actionTransfer)}))

			})
		})
		When("serial numbers already exist", func() {
			BeforeEach(func() {
				fakeRWSet.GetStateReturnsOnCall(2, []byte(strconv.FormatBool(true)), nil)
			})
			It("transfer fails", func() {
				err := writer.Write(faketransfer)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("already spent"))
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
				err := writer.CommitTokenRequest([]byte("token request"), false)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeRWSet.SetStateCallCount()).To(Equal(1))

				ns, id, tr := fakeRWSet.SetStateArgsForCall(0)
				Expect(ns).To(Equal(tokenNameSpace))
				key, err := keys.CreateTokenRequestKey("0")
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
				err := writer.CommitTokenRequest([]byte("token request"), false)
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
				err := writer.CommitTokenRequest([]byte("token request"), false)
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
				err := writer.CommitTokenRequest([]byte("token request"), false)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("token request with same ID already exists"))
				Expect(fakeRWSet.SetStateCallCount()).To(Equal(0))

			})
		})
	})
})
