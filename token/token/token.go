/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"bytes"
	"fmt"
)

// ID identifies a token as a function of the identifier of the transaction (issue, transfer)
// that created it and its index in that transaction
type ID struct {
	// TxId is the transaction ID of the transaction that created the token
	TxId string `protobuf:"bytes,1,opt,name=tx_id,json=txId,proto3" json:"tx_id,omitempty"`
	// Index is the index of the token in the transaction that created it
	Index uint64 `protobuf:"varint,2,opt,name=index,proto3" json:"index,omitempty"`
}

func (id ID) Equal(right ID) bool {
	return id.TxId == right.TxId && id.Index == right.Index
}

func (id ID) String() string {
	return fmt.Sprintf("[%s:%d]", id.TxId, id.Index)
}

type (
	// Type is the currency, e.g. USD
	Type string
	// Format is the encoding of a token on the ledger, e.g. fabtoken, comm.
	// It is a many-to-many relationship with the token driver,
	// i.e. a token driver can support multiple formats (e.g. fabtoken1, fabtoken2),
	// but a format can also be supported by multiple drivers (e.g. zkat, zkatlog).
	Format string
)

// Token is the result of issue and transfer transactions
type Token struct {
	// Owner is the token owner
	Owner []byte `protobuf:"bytes,1,opt,name=owner,proto3" json:"owner,omitempty"`
	// Type is the type of the token
	Type Type `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	// Quantity is the number of units of Type carried in the token.
	// It is encoded as a string containing a number in base 16. The string has prefix ``0x''.
	Quantity string `protobuf:"bytes,3,opt,name=quantity,proto3" json:"quantity,omitempty"`
}

type IssuedToken struct {
	// Id is used to uniquely identify the token in the ledger
	Id ID `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// Owner is the token owner
	Owner []byte `protobuf:"bytes,1,opt,name=owner,proto3" json:"owner,omitempty"`
	// Type is the type of the token
	Type Type `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	// Quantity represents the number of units of Type that this unspent token holds.
	// It is formatted in decimal representation
	Quantity string `protobuf:"bytes,3,opt,name=quantity,proto3" json:"quantity,omitempty"`
	// Issuer is the issuer of this token
	Issuer []byte
}

type IssuedTokens struct {
	// Tokens is an array of UnspentToken
	Tokens []*IssuedToken `protobuf:"bytes,1,rep,name=tokens,proto3" json:"tokens,omitempty"`
}

func (it *IssuedTokens) Sum(precision uint64) Quantity {
	sum := NewZeroQuantity(precision)
	for _, token := range it.Tokens {
		q, err := ToQuantity(token.Quantity, precision)
		if err != nil {
			panic(err)
		}
		sum = sum.Add(q)
	}
	return sum
}

func (it *IssuedTokens) ByType(typ Type) *IssuedTokens {
	res := &IssuedTokens{Tokens: []*IssuedToken{}}
	for _, token := range it.Tokens {
		if token.Type == typ {
			res.Tokens = append(res.Tokens, token)
		}
	}
	return res
}

func (it *IssuedTokens) Count() int {
	return len(it.Tokens)
}

// UnspentTokenInWallet models an unspent token owner solely by a given wallet
type UnspentTokenInWallet struct {
	// Id is used to uniquely identify the token in the ledger
	Id ID
	// WalletID is the ID of the wallet owning this token
	WalletID string
	// Type is the type of the token
	Type Type
	// Quantity represents the number of units of Type that this unspent token holds.
	Quantity string
}

type LedgerToken struct {
	// ID is used to uniquely identify the token in the ledger
	ID            ID
	Format        Format
	Token         []byte
	TokenMetadata []byte
}

func (t LedgerToken) Equal(right LedgerToken) bool {
	return t.ID.Equal(right.ID) &&
		bytes.Equal([]byte(t.Format), []byte(right.Format)) &&
		bytes.Equal(t.Token, right.Token) &&
		bytes.Equal(t.TokenMetadata, right.TokenMetadata)
}

// UnspentToken models an unspent token
type UnspentToken struct {
	// Id is used to uniquely identify the token in the ledger
	Id ID
	// Owner is the token owner
	Owner []byte
	// Type is the type of the token
	Type Type
	// Quantity represents the number of units of Type that this unspent token holds.
	Quantity string
}

func (ut UnspentToken) String() string {
	return ut.Id.String()
}

// UnspentTokens is used to hold the output of ListRequest
type UnspentTokens struct {
	// Tokens is an array of UnspentToken
	Tokens []*UnspentToken `protobuf:"bytes,1,rep,name=tokens,proto3" json:"tokens,omitempty"`
}

func (it *UnspentTokens) Count() int {
	return len(it.Tokens)
}

func (it *UnspentTokens) Sum(precision uint64) Quantity {
	sum := NewZeroQuantity(precision)
	for _, token := range it.Tokens {
		q, err := ToQuantity(token.Quantity, precision)
		if err != nil {
			panic(err)
		}
		sum = sum.Add(q)
	}
	return sum
}

func (it *UnspentTokens) ByType(typ Type) *UnspentTokens {
	res := &UnspentTokens{Tokens: []*UnspentToken{}}
	for _, token := range it.Tokens {
		if token.Type == typ {
			res.Tokens = append(res.Tokens, token)
		}
	}
	return res
}

// At returns the unspent token at position i.
// No boundary checks are performed.
func (it *UnspentTokens) At(i int) *UnspentToken {
	return it.Tokens[i]
}
