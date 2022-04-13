// Code generated by protoc-gen-go. DO NOT EDIT.
// source: witness.proto

package types // import "github.com/33cn/plugin/plugin/dapp/zkspot/types"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type ZkSignature struct {
	PubKey               *ZkPubKey `protobuf:"bytes,1,opt,name=pubKey,proto3" json:"pubKey,omitempty"`
	SignInfo             string    `protobuf:"bytes,2,opt,name=signInfo,proto3" json:"signInfo,omitempty"`
	Msg                  *ZkMsg    `protobuf:"bytes,3,opt,name=msg,proto3" json:"msg,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *ZkSignature) Reset()         { *m = ZkSignature{} }
func (m *ZkSignature) String() string { return proto.CompactTextString(m) }
func (*ZkSignature) ProtoMessage()    {}
func (*ZkSignature) Descriptor() ([]byte, []int) {
	return fileDescriptor_witness_3f43f15220bd2970, []int{0}
}
func (m *ZkSignature) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ZkSignature.Unmarshal(m, b)
}
func (m *ZkSignature) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ZkSignature.Marshal(b, m, deterministic)
}
func (dst *ZkSignature) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ZkSignature.Merge(dst, src)
}
func (m *ZkSignature) XXX_Size() int {
	return xxx_messageInfo_ZkSignature.Size(m)
}
func (m *ZkSignature) XXX_DiscardUnknown() {
	xxx_messageInfo_ZkSignature.DiscardUnknown(m)
}

var xxx_messageInfo_ZkSignature proto.InternalMessageInfo

func (m *ZkSignature) GetPubKey() *ZkPubKey {
	if m != nil {
		return m.PubKey
	}
	return nil
}

func (m *ZkSignature) GetSignInfo() string {
	if m != nil {
		return m.SignInfo
	}
	return ""
}

func (m *ZkSignature) GetMsg() *ZkMsg {
	if m != nil {
		return m.Msg
	}
	return nil
}

type ZkMsg struct {
	First                string   `protobuf:"bytes,1,opt,name=first,proto3" json:"first,omitempty"`
	Second               string   `protobuf:"bytes,2,opt,name=second,proto3" json:"second,omitempty"`
	Third                string   `protobuf:"bytes,3,opt,name=third,proto3" json:"third,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ZkMsg) Reset()         { *m = ZkMsg{} }
func (m *ZkMsg) String() string { return proto.CompactTextString(m) }
func (*ZkMsg) ProtoMessage()    {}
func (*ZkMsg) Descriptor() ([]byte, []int) {
	return fileDescriptor_witness_3f43f15220bd2970, []int{1}
}
func (m *ZkMsg) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ZkMsg.Unmarshal(m, b)
}
func (m *ZkMsg) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ZkMsg.Marshal(b, m, deterministic)
}
func (dst *ZkMsg) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ZkMsg.Merge(dst, src)
}
func (m *ZkMsg) XXX_Size() int {
	return xxx_messageInfo_ZkMsg.Size(m)
}
func (m *ZkMsg) XXX_DiscardUnknown() {
	xxx_messageInfo_ZkMsg.DiscardUnknown(m)
}

var xxx_messageInfo_ZkMsg proto.InternalMessageInfo

func (m *ZkMsg) GetFirst() string {
	if m != nil {
		return m.First
	}
	return ""
}

func (m *ZkMsg) GetSecond() string {
	if m != nil {
		return m.Second
	}
	return ""
}

func (m *ZkMsg) GetThird() string {
	if m != nil {
		return m.Third
	}
	return ""
}

type ZkPubKey struct {
	X                    string   `protobuf:"bytes,1,opt,name=x,proto3" json:"x,omitempty"`
	Y                    string   `protobuf:"bytes,2,opt,name=y,proto3" json:"y,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ZkPubKey) Reset()         { *m = ZkPubKey{} }
func (m *ZkPubKey) String() string { return proto.CompactTextString(m) }
func (*ZkPubKey) ProtoMessage()    {}
func (*ZkPubKey) Descriptor() ([]byte, []int) {
	return fileDescriptor_witness_3f43f15220bd2970, []int{2}
}
func (m *ZkPubKey) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ZkPubKey.Unmarshal(m, b)
}
func (m *ZkPubKey) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ZkPubKey.Marshal(b, m, deterministic)
}
func (dst *ZkPubKey) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ZkPubKey.Merge(dst, src)
}
func (m *ZkPubKey) XXX_Size() int {
	return xxx_messageInfo_ZkPubKey.Size(m)
}
func (m *ZkPubKey) XXX_DiscardUnknown() {
	xxx_messageInfo_ZkPubKey.DiscardUnknown(m)
}

var xxx_messageInfo_ZkPubKey proto.InternalMessageInfo

func (m *ZkPubKey) GetX() string {
	if m != nil {
		return m.X
	}
	return ""
}

func (m *ZkPubKey) GetY() string {
	if m != nil {
		return m.Y
	}
	return ""
}

type SiblingPath struct {
	Path                 []string `protobuf:"bytes,1,rep,name=path,proto3" json:"path,omitempty"`
	Helper               []string `protobuf:"bytes,2,rep,name=helper,proto3" json:"helper,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SiblingPath) Reset()         { *m = SiblingPath{} }
func (m *SiblingPath) String() string { return proto.CompactTextString(m) }
func (*SiblingPath) ProtoMessage()    {}
func (*SiblingPath) Descriptor() ([]byte, []int) {
	return fileDescriptor_witness_3f43f15220bd2970, []int{3}
}
func (m *SiblingPath) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SiblingPath.Unmarshal(m, b)
}
func (m *SiblingPath) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SiblingPath.Marshal(b, m, deterministic)
}
func (dst *SiblingPath) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SiblingPath.Merge(dst, src)
}
func (m *SiblingPath) XXX_Size() int {
	return xxx_messageInfo_SiblingPath.Size(m)
}
func (m *SiblingPath) XXX_DiscardUnknown() {
	xxx_messageInfo_SiblingPath.DiscardUnknown(m)
}

var xxx_messageInfo_SiblingPath proto.InternalMessageInfo

func (m *SiblingPath) GetPath() []string {
	if m != nil {
		return m.Path
	}
	return nil
}

func (m *SiblingPath) GetHelper() []string {
	if m != nil {
		return m.Helper
	}
	return nil
}

type AccountWitness struct {
	ID                   uint64       `protobuf:"varint,1,opt,name=ID,proto3" json:"ID,omitempty"`
	EthAddr              string       `protobuf:"bytes,2,opt,name=ethAddr,proto3" json:"ethAddr,omitempty"`
	Chain33Addr          string       `protobuf:"bytes,3,opt,name=chain33Addr,proto3" json:"chain33Addr,omitempty"`
	TokenTreeRoot        string       `protobuf:"bytes,4,opt,name=tokenTreeRoot,proto3" json:"tokenTreeRoot,omitempty"`
	PubKey               *ZkPubKey    `protobuf:"bytes,5,opt,name=pubKey,proto3" json:"pubKey,omitempty"`
	Sibling              *SiblingPath `protobuf:"bytes,6,opt,name=sibling,proto3" json:"sibling,omitempty"`
	XXX_NoUnkeyedLiteral struct{}     `json:"-"`
	XXX_unrecognized     []byte       `json:"-"`
	XXX_sizecache        int32        `json:"-"`
}

func (m *AccountWitness) Reset()         { *m = AccountWitness{} }
func (m *AccountWitness) String() string { return proto.CompactTextString(m) }
func (*AccountWitness) ProtoMessage()    {}
func (*AccountWitness) Descriptor() ([]byte, []int) {
	return fileDescriptor_witness_3f43f15220bd2970, []int{4}
}
func (m *AccountWitness) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AccountWitness.Unmarshal(m, b)
}
func (m *AccountWitness) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AccountWitness.Marshal(b, m, deterministic)
}
func (dst *AccountWitness) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AccountWitness.Merge(dst, src)
}
func (m *AccountWitness) XXX_Size() int {
	return xxx_messageInfo_AccountWitness.Size(m)
}
func (m *AccountWitness) XXX_DiscardUnknown() {
	xxx_messageInfo_AccountWitness.DiscardUnknown(m)
}

var xxx_messageInfo_AccountWitness proto.InternalMessageInfo

func (m *AccountWitness) GetID() uint64 {
	if m != nil {
		return m.ID
	}
	return 0
}

func (m *AccountWitness) GetEthAddr() string {
	if m != nil {
		return m.EthAddr
	}
	return ""
}

func (m *AccountWitness) GetChain33Addr() string {
	if m != nil {
		return m.Chain33Addr
	}
	return ""
}

func (m *AccountWitness) GetTokenTreeRoot() string {
	if m != nil {
		return m.TokenTreeRoot
	}
	return ""
}

func (m *AccountWitness) GetPubKey() *ZkPubKey {
	if m != nil {
		return m.PubKey
	}
	return nil
}

func (m *AccountWitness) GetSibling() *SiblingPath {
	if m != nil {
		return m.Sibling
	}
	return nil
}

type TokenWitness struct {
	ID                   uint64       `protobuf:"varint,1,opt,name=ID,proto3" json:"ID,omitempty"`
	Balance              string       `protobuf:"bytes,2,opt,name=balance,proto3" json:"balance,omitempty"`
	Sibling              *SiblingPath `protobuf:"bytes,3,opt,name=sibling,proto3" json:"sibling,omitempty"`
	XXX_NoUnkeyedLiteral struct{}     `json:"-"`
	XXX_unrecognized     []byte       `json:"-"`
	XXX_sizecache        int32        `json:"-"`
}

func (m *TokenWitness) Reset()         { *m = TokenWitness{} }
func (m *TokenWitness) String() string { return proto.CompactTextString(m) }
func (*TokenWitness) ProtoMessage()    {}
func (*TokenWitness) Descriptor() ([]byte, []int) {
	return fileDescriptor_witness_3f43f15220bd2970, []int{5}
}
func (m *TokenWitness) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TokenWitness.Unmarshal(m, b)
}
func (m *TokenWitness) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TokenWitness.Marshal(b, m, deterministic)
}
func (dst *TokenWitness) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TokenWitness.Merge(dst, src)
}
func (m *TokenWitness) XXX_Size() int {
	return xxx_messageInfo_TokenWitness.Size(m)
}
func (m *TokenWitness) XXX_DiscardUnknown() {
	xxx_messageInfo_TokenWitness.DiscardUnknown(m)
}

var xxx_messageInfo_TokenWitness proto.InternalMessageInfo

func (m *TokenWitness) GetID() uint64 {
	if m != nil {
		return m.ID
	}
	return 0
}

func (m *TokenWitness) GetBalance() string {
	if m != nil {
		return m.Balance
	}
	return ""
}

func (m *TokenWitness) GetSibling() *SiblingPath {
	if m != nil {
		return m.Sibling
	}
	return nil
}

// one operation branch
type OperationMetaBranch struct {
	AccountWitness       *AccountWitness `protobuf:"bytes,1,opt,name=accountWitness,proto3" json:"accountWitness,omitempty"`
	TokenWitness         *TokenWitness   `protobuf:"bytes,2,opt,name=tokenWitness,proto3" json:"tokenWitness,omitempty"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func (m *OperationMetaBranch) Reset()         { *m = OperationMetaBranch{} }
func (m *OperationMetaBranch) String() string { return proto.CompactTextString(m) }
func (*OperationMetaBranch) ProtoMessage()    {}
func (*OperationMetaBranch) Descriptor() ([]byte, []int) {
	return fileDescriptor_witness_3f43f15220bd2970, []int{6}
}
func (m *OperationMetaBranch) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OperationMetaBranch.Unmarshal(m, b)
}
func (m *OperationMetaBranch) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OperationMetaBranch.Marshal(b, m, deterministic)
}
func (dst *OperationMetaBranch) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OperationMetaBranch.Merge(dst, src)
}
func (m *OperationMetaBranch) XXX_Size() int {
	return xxx_messageInfo_OperationMetaBranch.Size(m)
}
func (m *OperationMetaBranch) XXX_DiscardUnknown() {
	xxx_messageInfo_OperationMetaBranch.DiscardUnknown(m)
}

var xxx_messageInfo_OperationMetaBranch proto.InternalMessageInfo

func (m *OperationMetaBranch) GetAccountWitness() *AccountWitness {
	if m != nil {
		return m.AccountWitness
	}
	return nil
}

func (m *OperationMetaBranch) GetTokenWitness() *TokenWitness {
	if m != nil {
		return m.TokenWitness
	}
	return nil
}

// before and after operation data
type OperationPairBranch struct {
	Before               *OperationMetaBranch `protobuf:"bytes,1,opt,name=before,proto3" json:"before,omitempty"`
	After                *OperationMetaBranch `protobuf:"bytes,2,opt,name=after,proto3" json:"after,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *OperationPairBranch) Reset()         { *m = OperationPairBranch{} }
func (m *OperationPairBranch) String() string { return proto.CompactTextString(m) }
func (*OperationPairBranch) ProtoMessage()    {}
func (*OperationPairBranch) Descriptor() ([]byte, []int) {
	return fileDescriptor_witness_3f43f15220bd2970, []int{7}
}
func (m *OperationPairBranch) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OperationPairBranch.Unmarshal(m, b)
}
func (m *OperationPairBranch) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OperationPairBranch.Marshal(b, m, deterministic)
}
func (dst *OperationPairBranch) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OperationPairBranch.Merge(dst, src)
}
func (m *OperationPairBranch) XXX_Size() int {
	return xxx_messageInfo_OperationPairBranch.Size(m)
}
func (m *OperationPairBranch) XXX_DiscardUnknown() {
	xxx_messageInfo_OperationPairBranch.DiscardUnknown(m)
}

var xxx_messageInfo_OperationPairBranch proto.InternalMessageInfo

func (m *OperationPairBranch) GetBefore() *OperationMetaBranch {
	if m != nil {
		return m.Before
	}
	return nil
}

func (m *OperationPairBranch) GetAfter() *OperationMetaBranch {
	if m != nil {
		return m.After
	}
	return nil
}

type OperationInfo struct {
	BlockHeight uint64       `protobuf:"varint,1,opt,name=blockHeight,proto3" json:"blockHeight,omitempty"`
	TxIndex     uint32       `protobuf:"varint,2,opt,name=txIndex,proto3" json:"txIndex,omitempty"`
	OpIndex     uint32       `protobuf:"varint,3,opt,name=opIndex,proto3" json:"opIndex,omitempty"`
	TxType      uint32       `protobuf:"varint,4,opt,name=txType,proto3" json:"txType,omitempty"`
	TxHash      string       `protobuf:"bytes,5,opt,name=txHash,proto3" json:"txHash,omitempty"`
	AccountID   uint64       `protobuf:"varint,6,opt,name=accountID,proto3" json:"accountID,omitempty"`
	TokenID     uint64       `protobuf:"varint,7,opt,name=tokenID,proto3" json:"tokenID,omitempty"`
	Amount      string       `protobuf:"bytes,8,opt,name=amount,proto3" json:"amount,omitempty"`
	FeeAmount   string       `protobuf:"bytes,9,opt,name=feeAmount,proto3" json:"feeAmount,omitempty"`
	SigData     *ZkSignature `protobuf:"bytes,10,opt,name=sigData,proto3" json:"sigData,omitempty"`
	Roots       []string     `protobuf:"bytes,11,rep,name=roots,proto3" json:"roots,omitempty"`
	// 每个operation data由一对 操作前后数据组成，不同操作可以有多个操作数据，deposit:1,transfer:2
	OperationBranches []*OperationPairBranch `protobuf:"bytes,12,rep,name=operationBranches,proto3" json:"operationBranches,omitempty"`
	// 操作特殊数据,像订单数据
	SpecialInfo          *OperationSpecialInfo `protobuf:"bytes,13,opt,name=specialInfo,proto3" json:"specialInfo,omitempty"`
	XXX_NoUnkeyedLiteral struct{}              `json:"-"`
	XXX_unrecognized     []byte                `json:"-"`
	XXX_sizecache        int32                 `json:"-"`
}

func (m *OperationInfo) Reset()         { *m = OperationInfo{} }
func (m *OperationInfo) String() string { return proto.CompactTextString(m) }
func (*OperationInfo) ProtoMessage()    {}
func (*OperationInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_witness_3f43f15220bd2970, []int{8}
}
func (m *OperationInfo) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OperationInfo.Unmarshal(m, b)
}
func (m *OperationInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OperationInfo.Marshal(b, m, deterministic)
}
func (dst *OperationInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OperationInfo.Merge(dst, src)
}
func (m *OperationInfo) XXX_Size() int {
	return xxx_messageInfo_OperationInfo.Size(m)
}
func (m *OperationInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_OperationInfo.DiscardUnknown(m)
}

var xxx_messageInfo_OperationInfo proto.InternalMessageInfo

func (m *OperationInfo) GetBlockHeight() uint64 {
	if m != nil {
		return m.BlockHeight
	}
	return 0
}

func (m *OperationInfo) GetTxIndex() uint32 {
	if m != nil {
		return m.TxIndex
	}
	return 0
}

func (m *OperationInfo) GetOpIndex() uint32 {
	if m != nil {
		return m.OpIndex
	}
	return 0
}

func (m *OperationInfo) GetTxType() uint32 {
	if m != nil {
		return m.TxType
	}
	return 0
}

func (m *OperationInfo) GetTxHash() string {
	if m != nil {
		return m.TxHash
	}
	return ""
}

func (m *OperationInfo) GetAccountID() uint64 {
	if m != nil {
		return m.AccountID
	}
	return 0
}

func (m *OperationInfo) GetTokenID() uint64 {
	if m != nil {
		return m.TokenID
	}
	return 0
}

func (m *OperationInfo) GetAmount() string {
	if m != nil {
		return m.Amount
	}
	return ""
}

func (m *OperationInfo) GetFeeAmount() string {
	if m != nil {
		return m.FeeAmount
	}
	return ""
}

func (m *OperationInfo) GetSigData() *ZkSignature {
	if m != nil {
		return m.SigData
	}
	return nil
}

func (m *OperationInfo) GetRoots() []string {
	if m != nil {
		return m.Roots
	}
	return nil
}

func (m *OperationInfo) GetOperationBranches() []*OperationPairBranch {
	if m != nil {
		return m.OperationBranches
	}
	return nil
}

func (m *OperationInfo) GetSpecialInfo() *OperationSpecialInfo {
	if m != nil {
		return m.SpecialInfo
	}
	return nil
}

type OperationSpecialInfo struct {
	SpecialDatas         []*OperationSpecialData `protobuf:"bytes,1,rep,name=specialDatas,proto3" json:"specialDatas,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                `json:"-"`
	XXX_unrecognized     []byte                  `json:"-"`
	XXX_sizecache        int32                   `json:"-"`
}

func (m *OperationSpecialInfo) Reset()         { *m = OperationSpecialInfo{} }
func (m *OperationSpecialInfo) String() string { return proto.CompactTextString(m) }
func (*OperationSpecialInfo) ProtoMessage()    {}
func (*OperationSpecialInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_witness_3f43f15220bd2970, []int{9}
}
func (m *OperationSpecialInfo) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OperationSpecialInfo.Unmarshal(m, b)
}
func (m *OperationSpecialInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OperationSpecialInfo.Marshal(b, m, deterministic)
}
func (dst *OperationSpecialInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OperationSpecialInfo.Merge(dst, src)
}
func (m *OperationSpecialInfo) XXX_Size() int {
	return xxx_messageInfo_OperationSpecialInfo.Size(m)
}
func (m *OperationSpecialInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_OperationSpecialInfo.DiscardUnknown(m)
}

var xxx_messageInfo_OperationSpecialInfo proto.InternalMessageInfo

func (m *OperationSpecialInfo) GetSpecialDatas() []*OperationSpecialData {
	if m != nil {
		return m.SpecialDatas
	}
	return nil
}

type OrderPricePair struct {
	Sell                 uint64   `protobuf:"varint,1,opt,name=sell,proto3" json:"sell,omitempty"`
	Buy                  uint64   `protobuf:"varint,2,opt,name=buy,proto3" json:"buy,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *OrderPricePair) Reset()         { *m = OrderPricePair{} }
func (m *OrderPricePair) String() string { return proto.CompactTextString(m) }
func (*OrderPricePair) ProtoMessage()    {}
func (*OrderPricePair) Descriptor() ([]byte, []int) {
	return fileDescriptor_witness_3f43f15220bd2970, []int{10}
}
func (m *OrderPricePair) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OrderPricePair.Unmarshal(m, b)
}
func (m *OrderPricePair) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OrderPricePair.Marshal(b, m, deterministic)
}
func (dst *OrderPricePair) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OrderPricePair.Merge(dst, src)
}
func (m *OrderPricePair) XXX_Size() int {
	return xxx_messageInfo_OrderPricePair.Size(m)
}
func (m *OrderPricePair) XXX_DiscardUnknown() {
	xxx_messageInfo_OrderPricePair.DiscardUnknown(m)
}

var xxx_messageInfo_OrderPricePair proto.InternalMessageInfo

func (m *OrderPricePair) GetSell() uint64 {
	if m != nil {
		return m.Sell
	}
	return 0
}

func (m *OrderPricePair) GetBuy() uint64 {
	if m != nil {
		return m.Buy
	}
	return 0
}

type OperationSpecialData struct {
	AccountID            uint64            `protobuf:"varint,1,opt,name=accountID,proto3" json:"accountID,omitempty"`
	RecipientID          uint64            `protobuf:"varint,2,opt,name=recipientID,proto3" json:"recipientID,omitempty"`
	RecipientAddr        string            `protobuf:"bytes,3,opt,name=recipientAddr,proto3" json:"recipientAddr,omitempty"`
	Amount               []string          `protobuf:"bytes,4,rep,name=amount,proto3" json:"amount,omitempty"`
	ChainID              []uint32          `protobuf:"varint,5,rep,packed,name=chainID,proto3" json:"chainID,omitempty"`
	TokenID              []uint64          `protobuf:"varint,6,rep,packed,name=tokenID,proto3" json:"tokenID,omitempty"`
	PricePair            []*OrderPricePair `protobuf:"bytes,7,rep,name=pricePair,proto3" json:"pricePair,omitempty"`
	SigData              *ZkSignature      `protobuf:"bytes,8,opt,name=sigData,proto3" json:"sigData,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *OperationSpecialData) Reset()         { *m = OperationSpecialData{} }
func (m *OperationSpecialData) String() string { return proto.CompactTextString(m) }
func (*OperationSpecialData) ProtoMessage()    {}
func (*OperationSpecialData) Descriptor() ([]byte, []int) {
	return fileDescriptor_witness_3f43f15220bd2970, []int{11}
}
func (m *OperationSpecialData) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_OperationSpecialData.Unmarshal(m, b)
}
func (m *OperationSpecialData) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_OperationSpecialData.Marshal(b, m, deterministic)
}
func (dst *OperationSpecialData) XXX_Merge(src proto.Message) {
	xxx_messageInfo_OperationSpecialData.Merge(dst, src)
}
func (m *OperationSpecialData) XXX_Size() int {
	return xxx_messageInfo_OperationSpecialData.Size(m)
}
func (m *OperationSpecialData) XXX_DiscardUnknown() {
	xxx_messageInfo_OperationSpecialData.DiscardUnknown(m)
}

var xxx_messageInfo_OperationSpecialData proto.InternalMessageInfo

func (m *OperationSpecialData) GetAccountID() uint64 {
	if m != nil {
		return m.AccountID
	}
	return 0
}

func (m *OperationSpecialData) GetRecipientID() uint64 {
	if m != nil {
		return m.RecipientID
	}
	return 0
}

func (m *OperationSpecialData) GetRecipientAddr() string {
	if m != nil {
		return m.RecipientAddr
	}
	return ""
}

func (m *OperationSpecialData) GetAmount() []string {
	if m != nil {
		return m.Amount
	}
	return nil
}

func (m *OperationSpecialData) GetChainID() []uint32 {
	if m != nil {
		return m.ChainID
	}
	return nil
}

func (m *OperationSpecialData) GetTokenID() []uint64 {
	if m != nil {
		return m.TokenID
	}
	return nil
}

func (m *OperationSpecialData) GetPricePair() []*OrderPricePair {
	if m != nil {
		return m.PricePair
	}
	return nil
}

func (m *OperationSpecialData) GetSigData() *ZkSignature {
	if m != nil {
		return m.SigData
	}
	return nil
}

func init() {
	proto.RegisterType((*ZkSignature)(nil), "types.ZkSignature")
	proto.RegisterType((*ZkMsg)(nil), "types.ZkMsg")
	proto.RegisterType((*ZkPubKey)(nil), "types.ZkPubKey")
	proto.RegisterType((*SiblingPath)(nil), "types.SiblingPath")
	proto.RegisterType((*AccountWitness)(nil), "types.AccountWitness")
	proto.RegisterType((*TokenWitness)(nil), "types.TokenWitness")
	proto.RegisterType((*OperationMetaBranch)(nil), "types.OperationMetaBranch")
	proto.RegisterType((*OperationPairBranch)(nil), "types.OperationPairBranch")
	proto.RegisterType((*OperationInfo)(nil), "types.OperationInfo")
	proto.RegisterType((*OperationSpecialInfo)(nil), "types.OperationSpecialInfo")
	proto.RegisterType((*OrderPricePair)(nil), "types.OrderPricePair")
	proto.RegisterType((*OperationSpecialData)(nil), "types.OperationSpecialData")
}

func init() { proto.RegisterFile("witness.proto", fileDescriptor_witness_3f43f15220bd2970) }

var fileDescriptor_witness_3f43f15220bd2970 = []byte{
	// 811 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x55, 0xcd, 0x6e, 0xe3, 0x36,
	0x10, 0x86, 0x2c, 0xff, 0xc4, 0x23, 0x3b, 0x6d, 0xb9, 0xdb, 0x42, 0xd8, 0x16, 0x85, 0x20, 0x14,
	0x6d, 0x0e, 0x45, 0xdc, 0xc6, 0x40, 0x8b, 0x1e, 0x16, 0x45, 0x16, 0x3e, 0xc4, 0x58, 0x2c, 0x36,
	0x60, 0x02, 0x2c, 0x90, 0x1b, 0x2d, 0xd3, 0x12, 0x61, 0x47, 0x14, 0x28, 0x1a, 0xb5, 0xdb, 0x67,
	0xe8, 0x93, 0xf4, 0x65, 0x7a, 0xed, 0xdb, 0x14, 0x1c, 0x52, 0x36, 0x95, 0xcd, 0x26, 0x27, 0xeb,
	0x9b, 0xf9, 0xf4, 0xcd, 0x70, 0xf8, 0x69, 0x0c, 0xe3, 0x3f, 0x84, 0x2e, 0x79, 0x5d, 0x9f, 0x57,
	0x4a, 0x6a, 0x49, 0x7a, 0x7a, 0x5f, 0xf1, 0x3a, 0x55, 0x10, 0xdd, 0xad, 0x6f, 0x44, 0x5e, 0x32,
	0xbd, 0x55, 0x9c, 0xfc, 0x00, 0xfd, 0x6a, 0xbb, 0x78, 0xcb, 0xf7, 0x71, 0x90, 0x04, 0x67, 0xd1,
	0xc5, 0x67, 0xe7, 0x48, 0x3b, 0xbf, 0x5b, 0x5f, 0x63, 0x98, 0xba, 0x34, 0x79, 0x05, 0x27, 0xb5,
	0xc8, 0xcb, 0x79, 0xb9, 0x92, 0x71, 0x27, 0x09, 0xce, 0x86, 0xf4, 0x80, 0xc9, 0xb7, 0x10, 0xde,
	0xd7, 0x79, 0x1c, 0xa2, 0xc2, 0xe8, 0xa0, 0xf0, 0xae, 0xce, 0xa9, 0x49, 0xa4, 0x6f, 0xa1, 0x87,
	0x88, 0xbc, 0x84, 0xde, 0x4a, 0xa8, 0x5a, 0x63, 0xb1, 0x21, 0xb5, 0x80, 0x7c, 0x05, 0xfd, 0x9a,
	0x67, 0xb2, 0x5c, 0x3a, 0x61, 0x87, 0x0c, 0x5b, 0x17, 0x42, 0x2d, 0x51, 0x78, 0x48, 0x2d, 0x48,
	0xbf, 0x87, 0x93, 0xa6, 0x39, 0x32, 0x82, 0x60, 0xe7, 0xb4, 0x82, 0x9d, 0x41, 0x7b, 0x27, 0x11,
	0xec, 0xd3, 0xdf, 0x20, 0xba, 0x11, 0x8b, 0x8d, 0x28, 0xf3, 0x6b, 0xa6, 0x0b, 0x42, 0xa0, 0x5b,
	0x31, 0x5d, 0xc4, 0x41, 0x12, 0x9e, 0x0d, 0x29, 0x3e, 0x9b, 0xc2, 0x05, 0xdf, 0x54, 0x5c, 0xc5,
	0x1d, 0x8c, 0x3a, 0x94, 0xfe, 0x17, 0xc0, 0xe9, 0x65, 0x96, 0xc9, 0x6d, 0xa9, 0x3f, 0xd8, 0x19,
	0x92, 0x53, 0xe8, 0xcc, 0x67, 0x58, 0xaa, 0x4b, 0x3b, 0xf3, 0x19, 0x89, 0x61, 0xc0, 0x75, 0x71,
	0xb9, 0x5c, 0x2a, 0x57, 0xb1, 0x81, 0x24, 0x81, 0x28, 0x2b, 0x98, 0x28, 0xa7, 0x53, 0xcc, 0xda,
	0xde, 0xfd, 0x10, 0xf9, 0x0e, 0xc6, 0x5a, 0xae, 0x79, 0x79, 0xab, 0x38, 0xa7, 0x52, 0xea, 0xb8,
	0x8b, 0x9c, 0x76, 0xd0, 0xbb, 0x99, 0xde, 0xd3, 0x37, 0xf3, 0x23, 0x0c, 0x6a, 0x7b, 0xd0, 0xb8,
	0x8f, 0x4c, 0xe2, 0x98, 0xde, 0xf1, 0x69, 0x43, 0x49, 0x57, 0x30, 0xba, 0x35, 0x75, 0x9e, 0x38,
	0xd8, 0x82, 0x6d, 0x58, 0x99, 0xf1, 0xe6, 0x60, 0x0e, 0xfa, 0x75, 0xc2, 0xe7, 0xeb, 0xfc, 0x1d,
	0xc0, 0x8b, 0xf7, 0x15, 0x57, 0x4c, 0x0b, 0x59, 0xbe, 0xe3, 0x9a, 0xbd, 0x51, 0xac, 0xcc, 0x0a,
	0xf2, 0x1a, 0x4e, 0x59, 0x6b, 0xb4, 0xce, 0x78, 0x5f, 0x3a, 0xb1, 0xf6, 0xdc, 0xe9, 0x03, 0x32,
	0xf9, 0x15, 0x46, 0xda, 0x6b, 0x1f, 0x7b, 0x8c, 0x2e, 0x5e, 0xb8, 0x97, 0xfd, 0x93, 0xd1, 0x16,
	0x31, 0xfd, 0xcb, 0x6b, 0xe7, 0x9a, 0x09, 0xe5, 0xda, 0xb9, 0x80, 0xfe, 0x82, 0xaf, 0xa4, 0xe2,
	0xae, 0x8d, 0x57, 0x4e, 0xe9, 0x91, 0xd6, 0xa9, 0x63, 0x92, 0x9f, 0xa0, 0xc7, 0x56, 0x9a, 0x2b,
	0x57, 0xfc, 0xa9, 0x57, 0x2c, 0x31, 0xfd, 0x37, 0x84, 0xf1, 0x21, 0x8d, 0x9f, 0x4c, 0x02, 0xd1,
	0x62, 0x23, 0xb3, 0xf5, 0x15, 0x17, 0x79, 0xa1, 0xdd, 0xfc, 0xfd, 0x90, 0xb9, 0x08, 0xbd, 0x9b,
	0x97, 0x4b, 0xbe, 0xc3, 0x3a, 0x63, 0xda, 0x40, 0x93, 0x91, 0x95, 0xcd, 0x84, 0x36, 0xe3, 0xa0,
	0x31, 0xb4, 0xde, 0xdd, 0xee, 0x2b, 0x8e, 0x96, 0x1a, 0x53, 0x87, 0x6c, 0xfc, 0x8a, 0xd5, 0x05,
	0x7a, 0x69, 0x48, 0x1d, 0x22, 0xdf, 0xc0, 0xd0, 0xcd, 0x77, 0x3e, 0x43, 0xf3, 0x74, 0xe9, 0x31,
	0x80, 0x1d, 0x98, 0x11, 0xce, 0x67, 0xf1, 0x00, 0x73, 0x0d, 0x34, 0x7a, 0xec, 0xde, 0xb0, 0xe2,
	0x13, 0xab, 0x67, 0x91, 0xd1, 0x5b, 0x71, 0x7e, 0x69, 0x53, 0x43, 0x4c, 0x1d, 0x03, 0xd6, 0x40,
	0xf9, 0x8c, 0x69, 0x16, 0x43, 0xcb, 0x40, 0xde, 0x42, 0xa2, 0x0d, 0xc5, 0x7c, 0xfd, 0x4a, 0x4a,
	0x5d, 0xc7, 0x11, 0x7e, 0x9b, 0x16, 0x90, 0x2b, 0xf8, 0x42, 0x36, 0x83, 0xb4, 0x33, 0xe6, 0x75,
	0x3c, 0x4a, 0xc2, 0xc7, 0xee, 0xe1, 0x78, 0xcd, 0xf4, 0xe3, 0x97, 0xc8, 0x6b, 0x88, 0xea, 0x8a,
	0x67, 0x82, 0x6d, 0x70, 0xa7, 0x8d, 0xb1, 0xa3, 0xaf, 0x1f, 0x6a, 0xdc, 0x1c, 0x29, 0xd4, 0xe7,
	0xa7, 0x1f, 0xe0, 0xe5, 0x63, 0x24, 0xf2, 0x3b, 0x8c, 0x1c, 0xcd, 0x9c, 0xa2, 0xc6, 0x7d, 0xf3,
	0x69, 0x5d, 0xc3, 0xa1, 0xad, 0x17, 0xd2, 0x5f, 0xe0, 0xf4, 0xbd, 0x5a, 0x72, 0x75, 0xad, 0x44,
	0xc6, 0xcd, 0x11, 0xcc, 0xea, 0xaa, 0xf9, 0x66, 0xe3, 0x4c, 0x82, 0xcf, 0xe4, 0x73, 0x08, 0x17,
	0x5b, 0xbb, 0xed, 0xba, 0xd4, 0x3c, 0xa6, 0xff, 0x74, 0x3e, 0xee, 0x08, 0x07, 0xd9, 0xba, 0xe4,
	0xe0, 0xe1, 0x25, 0x27, 0x10, 0x29, 0x9e, 0x89, 0x4a, 0x70, 0xcc, 0x5b, 0x41, 0x3f, 0x64, 0xd6,
	0xd5, 0x01, 0x7a, 0x2b, 0xad, 0x1d, 0xf4, 0x2c, 0xd1, 0xb5, 0xbb, 0xd4, 0x59, 0x22, 0x86, 0x01,
	0xee, 0xbe, 0xf9, 0x2c, 0xee, 0x25, 0xa1, 0x31, 0xab, 0x83, 0xbe, 0xbd, 0xfa, 0x49, 0xe8, 0xdb,
	0x6b, 0x0a, 0xc3, 0xaa, 0x39, 0x7d, 0x3c, 0xc0, 0x01, 0x36, 0xeb, 0xa1, 0x3d, 0x1a, 0x7a, 0xe4,
	0xf9, 0xee, 0x3a, 0x79, 0xd6, 0x5d, 0x6f, 0x7e, 0xbe, 0x9b, 0xe4, 0x42, 0x17, 0xdb, 0xc5, 0x79,
	0x26, 0xef, 0x27, 0xd3, 0x69, 0x56, 0x4e, 0xaa, 0xcd, 0x36, 0x17, 0x87, 0x9f, 0x25, 0xab, 0xaa,
	0xc9, 0x9f, 0xeb, 0xba, 0x92, 0x7a, 0x82, 0x3a, 0x8b, 0x3e, 0xfe, 0x8f, 0x4e, 0xff, 0x0f, 0x00,
	0x00, 0xff, 0xff, 0x9a, 0x9c, 0x22, 0x45, 0x58, 0x07, 0x00, 0x00,
}
