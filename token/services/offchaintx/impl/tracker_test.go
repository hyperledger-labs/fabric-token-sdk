/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package impl_test

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/offchaintx/impl"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Transaction Tracker", func() {
	var (
		tracker *impl.Tracker
	)
	Describe("Open Channel", func() {
		Context("Channel is not already open", func() {
			BeforeEach(func() {
				tracker = &impl.Tracker{Party: "alice", Channels: make(map[string]*impl.Channel)}
			})
			It("Succeeds", func() {
				err := tracker.Open("ChannelID", "bob")
				Expect(err).NotTo(HaveOccurred())
				Expect(tracker.Channels["ChannelID"]).NotTo(BeNil())
			})
		})
		Context("Channel is already open", func() {
			BeforeEach(func() {
				tracker = &impl.Tracker{Party: "alice", Channels: make(map[string]*impl.Channel)}
				tracker.Channels["ChannelID"] = &impl.Channel{}
			})
			It("fails", func() {
				err := tracker.Open("ChannelID", "bob")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("channel with ID `ChannelID` is already open"))
			})
		})
	})

	Describe("Send", func() {
		BeforeEach(func() {
			tracker = &impl.Tracker{Party: "alice", Channels: make(map[string]*impl.Channel)}
		})
		Context("When the channel is intialized correctly", func() {
			BeforeEach(func() {
				err := tracker.Open("ChannelID", "bob")
				Expect(err).NotTo(HaveOccurred())
			})
			It("Succeeds", func() {
				err := tracker.Send("ChannelID", "USD", 100)
				Expect(err).NotTo(HaveOccurred())
				Expect(tracker.Channels["ChannelID"].SeqNumber).To(Equal(1))
				Expect(tracker.Channels["ChannelID"].Net["USD"]).To(Equal(int64(-100)))
				Expect(tracker.Channels["ChannelID"].Info[0].Value).To(Equal(int64(-100)))
				Expect(tracker.Channels["ChannelID"].Info[0].Type).To(Equal("USD"))
			})
		})
		Context("When the channel is not open", func() {
			It("fails", func() {
				err := tracker.Send("ChannelID", "USD", 100)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("there is no open channel with ID 'ChannelID'"))
			})
		})
		Context("When channel.Net is not initialized", func() {
			BeforeEach(func() {
				tracker = &impl.Tracker{Party: "alice", Channels: make(map[string]*impl.Channel)}
				tracker.Channels["ChannelID"] = &impl.Channel{}
			})
			It("fails", func() {
				err := tracker.Send("ChannelID", "USD", 100)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("channel with ID 'ChannelID' is not initialized properly"))
			})
		})
	})
	Describe("Receive", func() {
		BeforeEach(func() {
			tracker = &impl.Tracker{Party: "alice", Channels: make(map[string]*impl.Channel)}
		})
		Context("When the channel is intialized correctly", func() {
			BeforeEach(func() {
				err := tracker.Open("ChannelID", "bob")
				Expect(err).NotTo(HaveOccurred())
			})
			It("Succeeds", func() {
				err := tracker.Receive("ChannelID", "USD", 50, []byte("signature"))
				Expect(err).NotTo(HaveOccurred())
				Expect(tracker.Channels["ChannelID"].SeqNumber).To(Equal(1))
				Expect(tracker.Channels["ChannelID"].Net["USD"]).To(Equal(int64(50)))
				Expect(tracker.Channels["ChannelID"].Info[0].Value).To(Equal(int64(50)))
				Expect(tracker.Channels["ChannelID"].Info[0].Type).To(Equal("USD"))
				Expect(tracker.Channels["ChannelID"].ProofOfReceipt["1"]).To(Equal([]byte("signature")))

			})
		})
		Context("When the channel is not open", func() {
			It("fails", func() {
				err := tracker.Receive("ChannelID", "USD", 50, []byte("signature"))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("there is no open channel with ID 'ChannelID'"))
			})
		})
		Context("When channel.Net is not initialized", func() {
			BeforeEach(func() {
				tracker = &impl.Tracker{Party: "alice", Channels: make(map[string]*impl.Channel)}
				tracker.Channels["ChannelID"] = &impl.Channel{}
			})
			It("fails", func() {
				err := tracker.Receive("ChannelID", "USD", 50, []byte("signature"))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("channel with ID 'ChannelID' is not initialized properly"))
			})
		})

	})
	Describe("Net", func() {
		BeforeEach(func() {
			tracker = &impl.Tracker{Party: "alice", Channels: make(map[string]*impl.Channel)}
			err := tracker.Open("ChannelID", "bob")
			Expect(err).NotTo(HaveOccurred())
			Expect(tracker.Channels["ChannelID"].Net["USD"]).To(Equal(int64(0)))
			err = tracker.Receive("ChannelID", "USD", 50, []byte("signature"))
			Expect(err).NotTo(HaveOccurred())
		})
		Context("When the channel receives funds", func() {
			It("Increments", func() {
				tr, err := tracker.Net("ChannelID")
				Expect(err).NotTo(HaveOccurred())
				Expect(tr).NotTo(BeNil())
				Expect(len(tr)).To(Equal(1))
				Expect(tr[0].Type).To(Equal("USD"))
				Expect(tr[0].Value).To(Equal(uint64(50)))
				Expect(tr[0].Receiver).To(Equal("alice"))
				Expect(tr[0].Sender).To(Equal("bob"))
			})
		})
		Context("When the channel sends funds", func() {
			BeforeEach(func() {
				err := tracker.Send("ChannelID", "USD", 100)
				Expect(err).NotTo(HaveOccurred())
			})
			It("Decrements", func() {
				tr, err := tracker.Net("ChannelID")
				Expect(err).NotTo(HaveOccurred())
				Expect(tr).NotTo(BeNil())
				Expect(len(tr)).To(Equal(1))
				Expect(tr[0].Type).To(Equal("USD"))
				Expect(tr[0].Value).To(Equal(uint64(50)))
				Expect(tr[0].Receiver).To(Equal("bob"))
				Expect(tr[0].Sender).To(Equal("alice"))
			})
		})
		Context("When the channel is not open", func() {
			BeforeEach(func() {
				tracker = &impl.Tracker{Party: "alice", Channels: make(map[string]*impl.Channel)}
			})
			It("fails", func() {
				tr, err := tracker.Net("ChannelID")
				Expect(tr).To(BeNil())
				Expect(err).To(HaveOccurred())
			})
		})
	})
	Describe("Delete", func() {
		BeforeEach(func() {
			tracker = &impl.Tracker{Party: "alice", Channels: make(map[string]*impl.Channel)}
			err := tracker.Open("ChannelID", "bob")
			Expect(err).NotTo(HaveOccurred())
		})
		It("Succeeds", func() {
			err := tracker.Delete("ChannelID")
			Expect(err).NotTo(HaveOccurred())
			Expect(tracker.Channels["ChannelID"]).To(BeNil())
		})
	})

})
