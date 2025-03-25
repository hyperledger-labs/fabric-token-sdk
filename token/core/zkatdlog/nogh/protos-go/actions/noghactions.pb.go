//
//Copyright IBM Corp. All Rights Reserved.
//
//SPDX-License-Identifier: Apache-2.0

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.33.0
// 	protoc        v5.28.1
// source: noghactions.proto

package actions

import (
	reflect "reflect"
	sync "sync"

	actions "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/protos-go/actions"
	math "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/protos-go/math"
	pp "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/protos-go/pp"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Token struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Owner []byte   `protobuf:"bytes,1,opt,name=owner,proto3" json:"owner,omitempty"` // Owner is the owner of the token
	Data  *math.G1 `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`   // Data is the Pedersen commitment to type and value
}

func (x *Token) Reset() {
	*x = Token{}
	if protoimpl.UnsafeEnabled {
		mi := &file_noghactions_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Token) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Token) ProtoMessage() {}

func (x *Token) ProtoReflect() protoreflect.Message {
	mi := &file_noghactions_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Token.ProtoReflect.Descriptor instead.
func (*Token) Descriptor() ([]byte, []int) {
	return file_noghactions_proto_rawDescGZIP(), []int{0}
}

func (x *Token) GetOwner() []byte {
	if x != nil {
		return x.Owner
	}
	return nil
}

func (x *Token) GetData() *math.G1 {
	if x != nil {
		return x.Data
	}
	return nil
}

type TokenMetadata struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Type           string       `protobuf:"bytes,1,opt,name=type,proto3" json:"type,omitempty"`                                           // Type is the type of the token
	Value          *math.Zr     `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`                                         // Value is the quantity of the token
	BlindingFactor *math.Zr     `protobuf:"bytes,3,opt,name=blinding_factor,json=blindingFactor,proto3" json:"blinding_factor,omitempty"` // BlindingFactor is the blinding factor used to commit type and value
	Issuer         *pp.Identity `protobuf:"bytes,4,opt,name=issuer,proto3" json:"issuer,omitempty"`                                       // Issuer is the issuer of the token, if defined
}

func (x *TokenMetadata) Reset() {
	*x = TokenMetadata{}
	if protoimpl.UnsafeEnabled {
		mi := &file_noghactions_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TokenMetadata) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TokenMetadata) ProtoMessage() {}

func (x *TokenMetadata) ProtoReflect() protoreflect.Message {
	mi := &file_noghactions_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TokenMetadata.ProtoReflect.Descriptor instead.
func (*TokenMetadata) Descriptor() ([]byte, []int) {
	return file_noghactions_proto_rawDescGZIP(), []int{1}
}

func (x *TokenMetadata) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *TokenMetadata) GetValue() *math.Zr {
	if x != nil {
		return x.Value
	}
	return nil
}

func (x *TokenMetadata) GetBlindingFactor() *math.Zr {
	if x != nil {
		return x.BlindingFactor
	}
	return nil
}

func (x *TokenMetadata) GetIssuer() *pp.Identity {
	if x != nil {
		return x.Issuer
	}
	return nil
}

type TokenID struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id    string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Index uint64 `protobuf:"varint,2,opt,name=index,proto3" json:"index,omitempty"`
}

func (x *TokenID) Reset() {
	*x = TokenID{}
	if protoimpl.UnsafeEnabled {
		mi := &file_noghactions_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TokenID) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TokenID) ProtoMessage() {}

func (x *TokenID) ProtoReflect() protoreflect.Message {
	mi := &file_noghactions_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TokenID.ProtoReflect.Descriptor instead.
func (*TokenID) Descriptor() ([]byte, []int) {
	return file_noghactions_proto_rawDescGZIP(), []int{2}
}

func (x *TokenID) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *TokenID) GetIndex() uint64 {
	if x != nil {
		return x.Index
	}
	return 0
}

type TransferActionInput struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TokenId        *TokenID                           `protobuf:"bytes,1,opt,name=token_id,json=tokenId,proto3" json:"token_id,omitempty"`
	Input          *Token                             `protobuf:"bytes,2,opt,name=input,proto3" json:"input,omitempty"`
	UpgradeWitness *TransferActionInputUpgradeWitness `protobuf:"bytes,3,opt,name=upgrade_witness,json=upgradeWitness,proto3" json:"upgrade_witness,omitempty"`
}

func (x *TransferActionInput) Reset() {
	*x = TransferActionInput{}
	if protoimpl.UnsafeEnabled {
		mi := &file_noghactions_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TransferActionInput) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TransferActionInput) ProtoMessage() {}

func (x *TransferActionInput) ProtoReflect() protoreflect.Message {
	mi := &file_noghactions_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TransferActionInput.ProtoReflect.Descriptor instead.
func (*TransferActionInput) Descriptor() ([]byte, []int) {
	return file_noghactions_proto_rawDescGZIP(), []int{3}
}

func (x *TransferActionInput) GetTokenId() *TokenID {
	if x != nil {
		return x.TokenId
	}
	return nil
}

func (x *TransferActionInput) GetInput() *Token {
	if x != nil {
		return x.Input
	}
	return nil
}

func (x *TransferActionInput) GetUpgradeWitness() *TransferActionInputUpgradeWitness {
	if x != nil {
		return x.UpgradeWitness
	}
	return nil
}

type TransferActionInputUpgradeWitness struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Output         *actions.Token `protobuf:"bytes,1,opt,name=output,proto3" json:"output,omitempty"`
	BlindingFactor *math.Zr       `protobuf:"bytes,2,opt,name=blinding_factor,json=blindingFactor,proto3" json:"blinding_factor,omitempty"`
}

func (x *TransferActionInputUpgradeWitness) Reset() {
	*x = TransferActionInputUpgradeWitness{}
	if protoimpl.UnsafeEnabled {
		mi := &file_noghactions_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TransferActionInputUpgradeWitness) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TransferActionInputUpgradeWitness) ProtoMessage() {}

func (x *TransferActionInputUpgradeWitness) ProtoReflect() protoreflect.Message {
	mi := &file_noghactions_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TransferActionInputUpgradeWitness.ProtoReflect.Descriptor instead.
func (*TransferActionInputUpgradeWitness) Descriptor() ([]byte, []int) {
	return file_noghactions_proto_rawDescGZIP(), []int{4}
}

func (x *TransferActionInputUpgradeWitness) GetOutput() *actions.Token {
	if x != nil {
		return x.Output
	}
	return nil
}

func (x *TransferActionInputUpgradeWitness) GetBlindingFactor() *math.Zr {
	if x != nil {
		return x.BlindingFactor
	}
	return nil
}

type TransferActionOutput struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Token *Token `protobuf:"bytes,1,opt,name=token,proto3" json:"token,omitempty"` // Token is the new token
}

func (x *TransferActionOutput) Reset() {
	*x = TransferActionOutput{}
	if protoimpl.UnsafeEnabled {
		mi := &file_noghactions_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TransferActionOutput) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TransferActionOutput) ProtoMessage() {}

func (x *TransferActionOutput) ProtoReflect() protoreflect.Message {
	mi := &file_noghactions_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TransferActionOutput.ProtoReflect.Descriptor instead.
func (*TransferActionOutput) Descriptor() ([]byte, []int) {
	return file_noghactions_proto_rawDescGZIP(), []int{5}
}

func (x *TransferActionOutput) GetToken() *Token {
	if x != nil {
		return x.Token
	}
	return nil
}

type Proof struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Proof []byte `protobuf:"bytes,1,opt,name=proof,proto3" json:"proof,omitempty"`
}

func (x *Proof) Reset() {
	*x = Proof{}
	if protoimpl.UnsafeEnabled {
		mi := &file_noghactions_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Proof) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Proof) ProtoMessage() {}

func (x *Proof) ProtoReflect() protoreflect.Message {
	mi := &file_noghactions_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Proof.ProtoReflect.Descriptor instead.
func (*Proof) Descriptor() ([]byte, []int) {
	return file_noghactions_proto_rawDescGZIP(), []int{6}
}

func (x *Proof) GetProof() []byte {
	if x != nil {
		return x.Proof
	}
	return nil
}

type TransferAction struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Version  uint64                  `protobuf:"varint,1,opt,name=version,proto3" json:"version,omitempty"`
	Inputs   []*TransferActionInput  `protobuf:"bytes,2,rep,name=inputs,proto3" json:"inputs,omitempty"`                                                                                             // inputs
	Outputs  []*TransferActionOutput `protobuf:"bytes,3,rep,name=outputs,proto3" json:"outputs,omitempty"`                                                                                           // outputs
	Proof    *Proof                  `protobuf:"bytes,4,opt,name=proof,proto3" json:"proof,omitempty"`                                                                                               // ZK Proof that shows that the transfer is correct
	Metadata map[string][]byte       `protobuf:"bytes,5,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"` // Metadata contains the transfer action's metadata
}

func (x *TransferAction) Reset() {
	*x = TransferAction{}
	if protoimpl.UnsafeEnabled {
		mi := &file_noghactions_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TransferAction) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TransferAction) ProtoMessage() {}

func (x *TransferAction) ProtoReflect() protoreflect.Message {
	mi := &file_noghactions_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TransferAction.ProtoReflect.Descriptor instead.
func (*TransferAction) Descriptor() ([]byte, []int) {
	return file_noghactions_proto_rawDescGZIP(), []int{7}
}

func (x *TransferAction) GetVersion() uint64 {
	if x != nil {
		return x.Version
	}
	return 0
}

func (x *TransferAction) GetInputs() []*TransferActionInput {
	if x != nil {
		return x.Inputs
	}
	return nil
}

func (x *TransferAction) GetOutputs() []*TransferActionOutput {
	if x != nil {
		return x.Outputs
	}
	return nil
}

func (x *TransferAction) GetProof() *Proof {
	if x != nil {
		return x.Proof
	}
	return nil
}

func (x *TransferAction) GetMetadata() map[string][]byte {
	if x != nil {
		return x.Metadata
	}
	return nil
}

type IssueActionInput struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id    *TokenID `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`       // is the token id of the token to be redeemed
	Token []byte   `protobuf:"bytes,2,opt,name=token,proto3" json:"token,omitempty"` // is the actual token to be redeemed
}

func (x *IssueActionInput) Reset() {
	*x = IssueActionInput{}
	if protoimpl.UnsafeEnabled {
		mi := &file_noghactions_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *IssueActionInput) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*IssueActionInput) ProtoMessage() {}

func (x *IssueActionInput) ProtoReflect() protoreflect.Message {
	mi := &file_noghactions_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use IssueActionInput.ProtoReflect.Descriptor instead.
func (*IssueActionInput) Descriptor() ([]byte, []int) {
	return file_noghactions_proto_rawDescGZIP(), []int{8}
}

func (x *IssueActionInput) GetId() *TokenID {
	if x != nil {
		return x.Id
	}
	return nil
}

func (x *IssueActionInput) GetToken() []byte {
	if x != nil {
		return x.Token
	}
	return nil
}

type IssueActionOutput struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Token *Token `protobuf:"bytes,1,opt,name=token,proto3" json:"token,omitempty"` // is the newly issued token
}

func (x *IssueActionOutput) Reset() {
	*x = IssueActionOutput{}
	if protoimpl.UnsafeEnabled {
		mi := &file_noghactions_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *IssueActionOutput) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*IssueActionOutput) ProtoMessage() {}

func (x *IssueActionOutput) ProtoReflect() protoreflect.Message {
	mi := &file_noghactions_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use IssueActionOutput.ProtoReflect.Descriptor instead.
func (*IssueActionOutput) Descriptor() ([]byte, []int) {
	return file_noghactions_proto_rawDescGZIP(), []int{9}
}

func (x *IssueActionOutput) GetToken() *Token {
	if x != nil {
		return x.Token
	}
	return nil
}

type IssueAction struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Version  uint64               `protobuf:"varint,1,opt,name=version,proto3" json:"version,omitempty"`
	Issuer   *pp.Identity         `protobuf:"bytes,2,opt,name=issuer,proto3" json:"issuer,omitempty"`                                                                                             // is the identity of issuer
	Inputs   []*IssueActionInput  `protobuf:"bytes,3,rep,name=inputs,proto3" json:"inputs,omitempty"`                                                                                             // are the tokens to be redeemed by this issue action
	Outputs  []*IssueActionOutput `protobuf:"bytes,4,rep,name=outputs,proto3" json:"outputs,omitempty"`                                                                                           // are the newly issued tokens
	Proof    *Proof               `protobuf:"bytes,5,opt,name=proof,proto3" json:"proof,omitempty"`                                                                                               // carries the ZKP of IssueAction validity
	Metadata map[string][]byte    `protobuf:"bytes,6,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"` // Metadata of the issue action
}

func (x *IssueAction) Reset() {
	*x = IssueAction{}
	if protoimpl.UnsafeEnabled {
		mi := &file_noghactions_proto_msgTypes[10]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *IssueAction) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*IssueAction) ProtoMessage() {}

func (x *IssueAction) ProtoReflect() protoreflect.Message {
	mi := &file_noghactions_proto_msgTypes[10]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use IssueAction.ProtoReflect.Descriptor instead.
func (*IssueAction) Descriptor() ([]byte, []int) {
	return file_noghactions_proto_rawDescGZIP(), []int{10}
}

func (x *IssueAction) GetVersion() uint64 {
	if x != nil {
		return x.Version
	}
	return 0
}

func (x *IssueAction) GetIssuer() *pp.Identity {
	if x != nil {
		return x.Issuer
	}
	return nil
}

func (x *IssueAction) GetInputs() []*IssueActionInput {
	if x != nil {
		return x.Inputs
	}
	return nil
}

func (x *IssueAction) GetOutputs() []*IssueActionOutput {
	if x != nil {
		return x.Outputs
	}
	return nil
}

func (x *IssueAction) GetProof() *Proof {
	if x != nil {
		return x.Proof
	}
	return nil
}

func (x *IssueAction) GetMetadata() map[string][]byte {
	if x != nil {
		return x.Metadata
	}
	return nil
}

var File_noghactions_proto protoreflect.FileDescriptor

var file_noghactions_proto_rawDesc = []byte{
	0x0a, 0x11, 0x6e, 0x6f, 0x67, 0x68, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x04, 0x6e, 0x6f, 0x67, 0x68, 0x1a, 0x0e, 0x6e, 0x6f, 0x67, 0x68, 0x6d,
	0x61, 0x74, 0x68, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x0f, 0x66, 0x74, 0x61, 0x63, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x0c, 0x6e, 0x6f, 0x67, 0x68,
	0x70, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x3b, 0x0a, 0x05, 0x54, 0x6f, 0x6b, 0x65,
	0x6e, 0x12, 0x14, 0x0a, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c,
	0x52, 0x05, 0x6f, 0x77, 0x6e, 0x65, 0x72, 0x12, 0x1c, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x08, 0x2e, 0x6e, 0x6f, 0x67, 0x68, 0x2e, 0x47, 0x31, 0x52,
	0x04, 0x64, 0x61, 0x74, 0x61, 0x22, 0x9e, 0x01, 0x0a, 0x0d, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x4d,
	0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x1e, 0x0a, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x08, 0x2e, 0x6e, 0x6f, 0x67,
	0x68, 0x2e, 0x5a, 0x72, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x31, 0x0a, 0x0f, 0x62,
	0x6c, 0x69, 0x6e, 0x64, 0x69, 0x6e, 0x67, 0x5f, 0x66, 0x61, 0x63, 0x74, 0x6f, 0x72, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x08, 0x2e, 0x6e, 0x6f, 0x67, 0x68, 0x2e, 0x5a, 0x72, 0x52, 0x0e,
	0x62, 0x6c, 0x69, 0x6e, 0x64, 0x69, 0x6e, 0x67, 0x46, 0x61, 0x63, 0x74, 0x6f, 0x72, 0x12, 0x26,
	0x0a, 0x06, 0x69, 0x73, 0x73, 0x75, 0x65, 0x72, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e,
	0x2e, 0x6e, 0x6f, 0x67, 0x68, 0x2e, 0x49, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x52, 0x06,
	0x69, 0x73, 0x73, 0x75, 0x65, 0x72, 0x22, 0x2f, 0x0a, 0x07, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x49,
	0x44, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69,
	0x64, 0x12, 0x14, 0x0a, 0x05, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04,
	0x52, 0x05, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x22, 0xb4, 0x01, 0x0a, 0x13, 0x54, 0x72, 0x61, 0x6e,
	0x73, 0x66, 0x65, 0x72, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x12,
	0x28, 0x0a, 0x08, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x0d, 0x2e, 0x6e, 0x6f, 0x67, 0x68, 0x2e, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x49, 0x44,
	0x52, 0x07, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x49, 0x64, 0x12, 0x21, 0x0a, 0x05, 0x69, 0x6e, 0x70,
	0x75, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0b, 0x2e, 0x6e, 0x6f, 0x67, 0x68, 0x2e,
	0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x52, 0x05, 0x69, 0x6e, 0x70, 0x75, 0x74, 0x12, 0x50, 0x0a, 0x0f,
	0x75, 0x70, 0x67, 0x72, 0x61, 0x64, 0x65, 0x5f, 0x77, 0x69, 0x74, 0x6e, 0x65, 0x73, 0x73, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x6e, 0x6f, 0x67, 0x68, 0x2e, 0x54, 0x72, 0x61,
	0x6e, 0x73, 0x66, 0x65, 0x72, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x6e, 0x70, 0x75, 0x74,
	0x55, 0x70, 0x67, 0x72, 0x61, 0x64, 0x65, 0x57, 0x69, 0x74, 0x6e, 0x65, 0x73, 0x73, 0x52, 0x0e,
	0x75, 0x70, 0x67, 0x72, 0x61, 0x64, 0x65, 0x57, 0x69, 0x74, 0x6e, 0x65, 0x73, 0x73, 0x22, 0x7f,
	0x0a, 0x21, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x65, 0x72, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e,
	0x49, 0x6e, 0x70, 0x75, 0x74, 0x55, 0x70, 0x67, 0x72, 0x61, 0x64, 0x65, 0x57, 0x69, 0x74, 0x6e,
	0x65, 0x73, 0x73, 0x12, 0x27, 0x0a, 0x06, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x0f, 0x2e, 0x66, 0x61, 0x62, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x2e, 0x54,
	0x6f, 0x6b, 0x65, 0x6e, 0x52, 0x06, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x12, 0x31, 0x0a, 0x0f,
	0x62, 0x6c, 0x69, 0x6e, 0x64, 0x69, 0x6e, 0x67, 0x5f, 0x66, 0x61, 0x63, 0x74, 0x6f, 0x72, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x08, 0x2e, 0x6e, 0x6f, 0x67, 0x68, 0x2e, 0x5a, 0x72, 0x52,
	0x0e, 0x62, 0x6c, 0x69, 0x6e, 0x64, 0x69, 0x6e, 0x67, 0x46, 0x61, 0x63, 0x74, 0x6f, 0x72, 0x22,
	0x39, 0x0a, 0x14, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x65, 0x72, 0x41, 0x63, 0x74, 0x69, 0x6f,
	0x6e, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x12, 0x21, 0x0a, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0b, 0x2e, 0x6e, 0x6f, 0x67, 0x68, 0x2e, 0x54, 0x6f,
	0x6b, 0x65, 0x6e, 0x52, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x22, 0x1d, 0x0a, 0x05, 0x50, 0x72,
	0x6f, 0x6f, 0x66, 0x12, 0x14, 0x0a, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x22, 0xb3, 0x02, 0x0a, 0x0e, 0x54, 0x72,
	0x61, 0x6e, 0x73, 0x66, 0x65, 0x72, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x18, 0x0a, 0x07,
	0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x07, 0x76,
	0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x31, 0x0a, 0x06, 0x69, 0x6e, 0x70, 0x75, 0x74, 0x73,
	0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x6e, 0x6f, 0x67, 0x68, 0x2e, 0x54, 0x72,
	0x61, 0x6e, 0x73, 0x66, 0x65, 0x72, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x6e, 0x70, 0x75,
	0x74, 0x52, 0x06, 0x69, 0x6e, 0x70, 0x75, 0x74, 0x73, 0x12, 0x34, 0x0a, 0x07, 0x6f, 0x75, 0x74,
	0x70, 0x75, 0x74, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x6e, 0x6f, 0x67,
	0x68, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x65, 0x72, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e,
	0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x52, 0x07, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x73, 0x12,
	0x21, 0x0a, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0b,
	0x2e, 0x6e, 0x6f, 0x67, 0x68, 0x2e, 0x50, 0x72, 0x6f, 0x6f, 0x66, 0x52, 0x05, 0x70, 0x72, 0x6f,
	0x6f, 0x66, 0x12, 0x3e, 0x0a, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x05,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x6e, 0x6f, 0x67, 0x68, 0x2e, 0x54, 0x72, 0x61, 0x6e,
	0x73, 0x66, 0x65, 0x72, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x4d, 0x65, 0x74, 0x61, 0x64,
	0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61,
	0x74, 0x61, 0x1a, 0x3b, 0x0a, 0x0d, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x45, 0x6e,
	0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22,
	0x47, 0x0a, 0x10, 0x49, 0x73, 0x73, 0x75, 0x65, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x6e,
	0x70, 0x75, 0x74, 0x12, 0x1d, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x0d, 0x2e, 0x6e, 0x6f, 0x67, 0x68, 0x2e, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x49, 0x44, 0x52, 0x02,
	0x69, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x22, 0x36, 0x0a, 0x11, 0x49, 0x73, 0x73, 0x75,
	0x65, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x12, 0x21, 0x0a,
	0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0b, 0x2e, 0x6e,
	0x6f, 0x67, 0x68, 0x2e, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x52, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e,
	0x22, 0xcf, 0x02, 0x0a, 0x0b, 0x49, 0x73, 0x73, 0x75, 0x65, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e,
	0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x04, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x26, 0x0a, 0x06, 0x69, 0x73,
	0x73, 0x75, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x6e, 0x6f, 0x67,
	0x68, 0x2e, 0x49, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x52, 0x06, 0x69, 0x73, 0x73, 0x75,
	0x65, 0x72, 0x12, 0x2e, 0x0a, 0x06, 0x69, 0x6e, 0x70, 0x75, 0x74, 0x73, 0x18, 0x03, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x16, 0x2e, 0x6e, 0x6f, 0x67, 0x68, 0x2e, 0x49, 0x73, 0x73, 0x75, 0x65, 0x41,
	0x63, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x52, 0x06, 0x69, 0x6e, 0x70, 0x75,
	0x74, 0x73, 0x12, 0x31, 0x0a, 0x07, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x73, 0x18, 0x04, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x6e, 0x6f, 0x67, 0x68, 0x2e, 0x49, 0x73, 0x73, 0x75, 0x65,
	0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x52, 0x07, 0x6f, 0x75,
	0x74, 0x70, 0x75, 0x74, 0x73, 0x12, 0x21, 0x0a, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x18, 0x05,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x0b, 0x2e, 0x6e, 0x6f, 0x67, 0x68, 0x2e, 0x50, 0x72, 0x6f, 0x6f,
	0x66, 0x52, 0x05, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x12, 0x3b, 0x0a, 0x08, 0x6d, 0x65, 0x74, 0x61,
	0x64, 0x61, 0x74, 0x61, 0x18, 0x06, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x6e, 0x6f, 0x67,
	0x68, 0x2e, 0x49, 0x73, 0x73, 0x75, 0x65, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x4d, 0x65,
	0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x08, 0x6d, 0x65, 0x74,
	0x61, 0x64, 0x61, 0x74, 0x61, 0x1a, 0x3b, 0x0a, 0x0d, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74,
	0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02,
	0x38, 0x01, 0x42, 0x59, 0x5a, 0x57, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x68, 0x79, 0x70, 0x65, 0x72, 0x6c, 0x65, 0x64, 0x67, 0x65, 0x72, 0x2d, 0x6c, 0x61, 0x62,
	0x73, 0x2f, 0x66, 0x61, 0x62, 0x72, 0x69, 0x63, 0x2d, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x2d, 0x73,
	0x64, 0x6b, 0x2f, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x2f, 0x63, 0x6f, 0x72, 0x65, 0x2f, 0x7a, 0x6b,
	0x61, 0x74, 0x64, 0x6c, 0x6f, 0x67, 0x2f, 0x6e, 0x6f, 0x67, 0x68, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x73, 0x2d, 0x67, 0x6f, 0x2f, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_noghactions_proto_rawDescOnce sync.Once
	file_noghactions_proto_rawDescData = file_noghactions_proto_rawDesc
)

func file_noghactions_proto_rawDescGZIP() []byte {
	file_noghactions_proto_rawDescOnce.Do(func() {
		file_noghactions_proto_rawDescData = protoimpl.X.CompressGZIP(file_noghactions_proto_rawDescData)
	})
	return file_noghactions_proto_rawDescData
}

var file_noghactions_proto_msgTypes = make([]protoimpl.MessageInfo, 13)
var file_noghactions_proto_goTypes = []interface{}{
	(*Token)(nil),                             // 0: nogh.Token
	(*TokenMetadata)(nil),                     // 1: nogh.TokenMetadata
	(*TokenID)(nil),                           // 2: nogh.TokenID
	(*TransferActionInput)(nil),               // 3: nogh.TransferActionInput
	(*TransferActionInputUpgradeWitness)(nil), // 4: nogh.TransferActionInputUpgradeWitness
	(*TransferActionOutput)(nil),              // 5: nogh.TransferActionOutput
	(*Proof)(nil),                             // 6: nogh.Proof
	(*TransferAction)(nil),                    // 7: nogh.TransferAction
	(*IssueActionInput)(nil),                  // 8: nogh.IssueActionInput
	(*IssueActionOutput)(nil),                 // 9: nogh.IssueActionOutput
	(*IssueAction)(nil),                       // 10: nogh.IssueAction
	nil,                                       // 11: nogh.TransferAction.MetadataEntry
	nil,                                       // 12: nogh.IssueAction.MetadataEntry
	(*math.G1)(nil),                           // 13: nogh.G1
	(*math.Zr)(nil),                           // 14: nogh.Zr
	(*pp.Identity)(nil),                       // 15: nogh.Identity
	(*actions.Token)(nil),                     // 16: fabtoken.Token
}
var file_noghactions_proto_depIdxs = []int32{
	13, // 0: nogh.Token.data:type_name -> nogh.G1
	14, // 1: nogh.TokenMetadata.value:type_name -> nogh.Zr
	14, // 2: nogh.TokenMetadata.blinding_factor:type_name -> nogh.Zr
	15, // 3: nogh.TokenMetadata.issuer:type_name -> nogh.Identity
	2,  // 4: nogh.TransferActionInput.token_id:type_name -> nogh.TokenID
	0,  // 5: nogh.TransferActionInput.input:type_name -> nogh.Token
	4,  // 6: nogh.TransferActionInput.upgrade_witness:type_name -> nogh.TransferActionInputUpgradeWitness
	16, // 7: nogh.TransferActionInputUpgradeWitness.output:type_name -> fabtoken.Token
	14, // 8: nogh.TransferActionInputUpgradeWitness.blinding_factor:type_name -> nogh.Zr
	0,  // 9: nogh.TransferActionOutput.token:type_name -> nogh.Token
	3,  // 10: nogh.TransferAction.inputs:type_name -> nogh.TransferActionInput
	5,  // 11: nogh.TransferAction.outputs:type_name -> nogh.TransferActionOutput
	6,  // 12: nogh.TransferAction.proof:type_name -> nogh.Proof
	11, // 13: nogh.TransferAction.metadata:type_name -> nogh.TransferAction.MetadataEntry
	2,  // 14: nogh.IssueActionInput.id:type_name -> nogh.TokenID
	0,  // 15: nogh.IssueActionOutput.token:type_name -> nogh.Token
	15, // 16: nogh.IssueAction.issuer:type_name -> nogh.Identity
	8,  // 17: nogh.IssueAction.inputs:type_name -> nogh.IssueActionInput
	9,  // 18: nogh.IssueAction.outputs:type_name -> nogh.IssueActionOutput
	6,  // 19: nogh.IssueAction.proof:type_name -> nogh.Proof
	12, // 20: nogh.IssueAction.metadata:type_name -> nogh.IssueAction.MetadataEntry
	21, // [21:21] is the sub-list for method output_type
	21, // [21:21] is the sub-list for method input_type
	21, // [21:21] is the sub-list for extension type_name
	21, // [21:21] is the sub-list for extension extendee
	0,  // [0:21] is the sub-list for field type_name
}

func init() { file_noghactions_proto_init() }
func file_noghactions_proto_init() {
	if File_noghactions_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_noghactions_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Token); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_noghactions_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TokenMetadata); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_noghactions_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TokenID); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_noghactions_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TransferActionInput); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_noghactions_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TransferActionInputUpgradeWitness); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_noghactions_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TransferActionOutput); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_noghactions_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Proof); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_noghactions_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TransferAction); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_noghactions_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*IssueActionInput); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_noghactions_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*IssueActionOutput); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_noghactions_proto_msgTypes[10].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*IssueAction); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_noghactions_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   13,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_noghactions_proto_goTypes,
		DependencyIndexes: file_noghactions_proto_depIdxs,
		MessageInfos:      file_noghactions_proto_msgTypes,
	}.Build()
	File_noghactions_proto = out.File
	file_noghactions_proto_rawDesc = nil
	file_noghactions_proto_goTypes = nil
	file_noghactions_proto_depIdxs = nil
}
