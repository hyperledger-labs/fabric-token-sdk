/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package token

import "fmt"

// ID identifies a token as a function of the identifier of the transaction (issue, transfer)
// that created it and its index in that transaction
type ID struct {
	// TxId is the transaction ID of the transaction that created the token
	TxId string `protobuf:"bytes,1,opt,name=tx_id,json=txId,proto3" json:"tx_id,omitempty"`
	// Index is the index of the token in the transaction that created it
	Index uint64 `protobuf:"varint,2,opt,name=index,proto3" json:"index,omitempty"`
}

func (id *ID) String() string {
	return fmt.Sprintf("[%s:%d]", id.TxId, id.Index)
}

// Owner holds the identity of a token owner
type Owner struct {
	// Raw is the serialization of the identity
	Raw []byte `protobuf:"bytes,2,opt,name=raw,proto3" json:"raw,omitempty"`
}

// Token is the result of issue and transfer transactions
type Token struct {
	// Owner is the token owner
	Owner *Owner `protobuf:"bytes,1,opt,name=owner,proto3" json:"owner,omitempty"`
	// Type is the type of the token
	Type string `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	// Quantity is the number of units of Type carried in the token.
	// It is encoded as a string containing a number in base 16. The string has prefix ``0x''.
	Quantity string `protobuf:"bytes,3,opt,name=quantity,proto3" json:"quantity,omitempty"`
}

type IssuedToken struct {
	// Id is used to uniquely identify the token in the ledger
	Id *ID `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// Owner is the token owner
	Owner *Owner `protobuf:"bytes,1,opt,name=owner,proto3" json:"owner,omitempty"`
	// Type is the type of the token
	Type string `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	// Quantity represents the number of units of Type that this unspent token holds.
	// It is formatted in decimal representation
	Quantity string `protobuf:"bytes,3,opt,name=quantity,proto3" json:"quantity,omitempty"`

	Issuer *Owner
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

func (it *IssuedTokens) ByType(typ string) *IssuedTokens {
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

// UnspentToken is used to specify a token returned by ListRequest
type UnspentToken struct {
	// Id is used to uniquely identify the token in the ledger
	Id *ID
	// Owner is the token owner
	Owner *Owner
	// Type is the type of the token
	Type string
	// DecimalQuantity represents the number of units of Type that this unspent token holds.
	// It is formatted in decimal representation
	DecimalQuantity string
	// Quantity represents the number of units of Type that this unspent token holds.
	// It might be nil.
	Quantity Quantity `json:"-"`
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
		q, err := ToQuantity(token.DecimalQuantity, precision)
		if err != nil {
			panic(err)
		}
		sum = sum.Add(q)
	}
	return sum
}

func (it *UnspentTokens) ByType(typ string) *UnspentTokens {
	res := &UnspentTokens{Tokens: []*UnspentToken{}}
	for _, token := range it.Tokens {
		if token.Type == typ {
			res.Tokens = append(res.Tokens, token)
		}
	}
	return res
}
