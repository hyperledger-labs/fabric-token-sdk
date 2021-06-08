/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package impl

import (
	"crypto/sha256"
	"encoding/json"
	"strconv"

	"github.com/pkg/errors"
)

type Tracker struct {
	Party    string
	Channels map[string]*Channel // first key is hash of channelID (channel ID is unique)
}

// this what is used to compute the hash; party and counterparty will end up with the same hash
type Transfer struct {
	Sender   string
	Receiver string
	Type     string
	Value    uint64
}

type Net struct {
	Sender        string
	Receiver      string
	ToBeExchanged map[string]uint64
}

// receive is called after sending an ack
func (t *Tracker) Receive(id, ttype string, value uint64, sig []byte) error {
	if t.Channels[id] == nil {
		return errors.Errorf("there is no open channel with ID '%s'", id)
	}
	if t.Channels[id].Net == nil || t.Channels[id].ProofOfReceipt == nil {
		return errors.Errorf("channel with ID '%s' is not initialized properly", id)
	}
	t.Channels[id].Net[ttype] += int64(value)
	t.Channels[id].Info = append(t.Channels[id].Info, &ExchangeInfo{Type: ttype, Value: int64(value)})
	tr := &Transfer{Receiver: t.Party, Type: ttype, Value: value, Sender: t.Channels[id].Counterparty}
	raw, err := json.Marshal(tr)
	if err != nil {
		return err
	}
	bytes := append(t.Channels[id].Hash[:], raw...)
	t.Channels[id].Hash = sha256.Sum256(bytes)
	t.Channels[id].SeqNumber++
	key := strconv.Itoa(t.Channels[id].SeqNumber)
	t.Channels[id].ProofOfReceipt[key] = sig

	return nil
}

// this is called after receiving an ack
func (t *Tracker) Send(id, ttype string, value uint64) error {
	if t.Channels[id] == nil {
		return errors.Errorf("there is no open channel with ID '%s'", id)
	}
	if t.Channels[id].Net == nil {
		return errors.Errorf("channel with ID '%s' is not initialized properly", id)
	}
	t.Channels[id].Net[ttype] -= int64(value)
	t.Channels[id].Info = append(t.Channels[id].Info, &ExchangeInfo{Type: ttype, Value: -int64(value)})
	tr := &Transfer{Sender: t.Party, Type: ttype, Value: value, Receiver: t.Channels[id].Counterparty}
	raw, err := json.Marshal(tr)
	if err != nil {
		return err
	}
	bytes := append(t.Channels[id].Hash[:], raw...)
	t.Channels[id].Hash = sha256.Sum256(bytes)
	t.Channels[id].SeqNumber++

	return nil
}

func (t *Tracker) Net(id string) ([]*Transfer, error) {
	var net []*Transfer
	if t.Channels[id] == nil {
		return nil, errors.Errorf("channel with ID `%s` does not exist", id)
	}
	for k, v := range t.Channels[id].Net {
		if v > 0 {
			net = append(net, &Transfer{Type: k, Value: uint64(v), Sender: t.Channels[id].Counterparty, Receiver: t.Party})
		}
		if v < 0 {
			net = append(net, &Transfer{Type: k, Value: uint64(-v), Sender: t.Party, Receiver: t.Channels[id].Counterparty})
		}
	}

	return net, nil
}

func (t *Tracker) Open(id, counterparty string) error {
	if t.Channels[id] != nil {
		return errors.Errorf("channel with ID `%s` is already open", id)
	}
	t.Channels[id] = t.NewChannel(id, counterparty)
	return nil
}

func (t *Tracker) NewChannel(id, counterparty string) *Channel {
	var hash [32]byte
	if t.Party < counterparty {
		hash = sha256.Sum256(append([]byte(t.Party), []byte(counterparty)...))
	} else {
		hash = sha256.Sum256(append([]byte(counterparty), []byte(t.Party)...))
	}
	toBeHashed := append([]byte(id), hash[:]...)

	return &Channel{
		ID:             id,
		Counterparty:   counterparty,
		Net:            make(map[string]int64),
		Hash:           sha256.Sum256(toBeHashed),
		ProofOfReceipt: make(map[string][]byte),
	}
}

func (t *Tracker) Delete(id string) error {
	delete(t.Channels, id)
	return nil
}

// checks to be done at the view level
// waiting for ack means that discrepancy in sequence numbers is at most 1
// Alice -transfer-> Bob
// Bob -Ack-> Alice
// (Bob updates tracker) (Alice updates tracker)
// if Alice does not receive ack; then Alice won't update tracker (wait on ack)
// meanwhile Alice receives a Transfer from Bob with seqN = currentSeqN+1;
// Alice considers this an ack updates tracker and ack to Bob.
// if seqN < currentSeqN reject
// if seqN == currentSeqN ack

//  this is called after receiving an ack in the case of send or in case received seqN = currentSeqN+1
// if seqN < currentSeqN then reject
// if seqN = currentSeqN then resend ack in case I am receiver
// if seqN > currentSeN+1 the reject
// update is called after sending an ack in the case of receive

type Channel struct {
	ID             string // ID of channel
	Counterparty   string
	Net            map[string]int64  // keeps a track of the exchanges' net value
	Info           []*ExchangeInfo   // ordered list of all the information exchanged (helps with rollback) and dispute
	Hash           [32]byte          // Merkle tree or hash chain of all exchanges
	SeqNumber      int               // counter increased everytime the exchange is updated
	ProofOfReceipt map[string][]byte // counterparty signatures on their transfers (key is the corresponding sequence number)
}

type ExchangeInfo struct {
	Type  string
	Value int64 // value is negative when party needs to send; positive when party receives
}
