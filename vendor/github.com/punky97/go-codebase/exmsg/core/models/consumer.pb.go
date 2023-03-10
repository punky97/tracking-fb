// Code generated by protoc-gen-go. DO NOT EDIT.
// source: proto/core/models/consumer.proto

package core_models // import "bitbucket.org/brodev/beekit/exmsg/core/models"

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

type RmqExchQueueInfo struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	Type                 string   `protobuf:"bytes,2,opt,name=type" json:"type,omitempty"`
	AutoDelete           bool     `protobuf:"varint,3,opt,name=auto_delete,json=autoDelete" json:"auto_delete,omitempty"`
	Durable              bool     `protobuf:"varint,4,opt,name=durable" json:"durable,omitempty"`
	Internal             bool     `protobuf:"varint,5,opt,name=internal" json:"internal,omitempty"`
	Exclusive            bool     `protobuf:"varint,6,opt,name=exclusive" json:"exclusive,omitempty"`
	Nowait               bool     `protobuf:"varint,7,opt,name=nowait" json:"nowait,omitempty"`
	RoutingKey           string   `protobuf:"bytes,8,opt,name=routing_key,json=routingKey" json:"routing_key,omitempty"`
	BatchAck             bool     `protobuf:"varint,9,opt,name=batch_ack,json=batchAck" json:"batch_ack,omitempty"`
	Qos                  int64    `protobuf:"varint,10,opt,name=qos" json:"qos,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RmqExchQueueInfo) Reset()         { *m = RmqExchQueueInfo{} }
func (m *RmqExchQueueInfo) String() string { return proto.CompactTextString(m) }
func (*RmqExchQueueInfo) ProtoMessage()    {}
func (*RmqExchQueueInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_consumer_c4c806bed6e9ed0d, []int{0}
}
func (m *RmqExchQueueInfo) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RmqExchQueueInfo.Unmarshal(m, b)
}
func (m *RmqExchQueueInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RmqExchQueueInfo.Marshal(b, m, deterministic)
}
func (dst *RmqExchQueueInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RmqExchQueueInfo.Merge(dst, src)
}
func (m *RmqExchQueueInfo) XXX_Size() int {
	return xxx_messageInfo_RmqExchQueueInfo.Size(m)
}
func (m *RmqExchQueueInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_RmqExchQueueInfo.DiscardUnknown(m)
}

var xxx_messageInfo_RmqExchQueueInfo proto.InternalMessageInfo

func (m *RmqExchQueueInfo) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *RmqExchQueueInfo) GetType() string {
	if m != nil {
		return m.Type
	}
	return ""
}

func (m *RmqExchQueueInfo) GetAutoDelete() bool {
	if m != nil {
		return m.AutoDelete
	}
	return false
}

func (m *RmqExchQueueInfo) GetDurable() bool {
	if m != nil {
		return m.Durable
	}
	return false
}

func (m *RmqExchQueueInfo) GetInternal() bool {
	if m != nil {
		return m.Internal
	}
	return false
}

func (m *RmqExchQueueInfo) GetExclusive() bool {
	if m != nil {
		return m.Exclusive
	}
	return false
}

func (m *RmqExchQueueInfo) GetNowait() bool {
	if m != nil {
		return m.Nowait
	}
	return false
}

func (m *RmqExchQueueInfo) GetRoutingKey() string {
	if m != nil {
		return m.RoutingKey
	}
	return ""
}

func (m *RmqExchQueueInfo) GetBatchAck() bool {
	if m != nil {
		return m.BatchAck
	}
	return false
}

func (m *RmqExchQueueInfo) GetQos() int64 {
	if m != nil {
		return m.Qos
	}
	return 0
}

type ConsumerGroup struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ConsumerGroup) Reset()         { *m = ConsumerGroup{} }
func (m *ConsumerGroup) String() string { return proto.CompactTextString(m) }
func (*ConsumerGroup) ProtoMessage()    {}
func (*ConsumerGroup) Descriptor() ([]byte, []int) {
	return fileDescriptor_consumer_c4c806bed6e9ed0d, []int{1}
}
func (m *ConsumerGroup) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ConsumerGroup.Unmarshal(m, b)
}
func (m *ConsumerGroup) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ConsumerGroup.Marshal(b, m, deterministic)
}
func (dst *ConsumerGroup) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ConsumerGroup.Merge(dst, src)
}
func (m *ConsumerGroup) XXX_Size() int {
	return xxx_messageInfo_ConsumerGroup.Size(m)
}
func (m *ConsumerGroup) XXX_DiscardUnknown() {
	xxx_messageInfo_ConsumerGroup.DiscardUnknown(m)
}

var xxx_messageInfo_ConsumerGroup proto.InternalMessageInfo

func (m *ConsumerGroup) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

type RmqInputConf struct {
	Exch                 *RmqExchQueueInfo `protobuf:"bytes,1,opt,name=exch" json:"exch,omitempty"`
	Queue                *RmqExchQueueInfo `protobuf:"bytes,2,opt,name=queue" json:"queue,omitempty"`
	BatchAck             bool              `protobuf:"varint,3,opt,name=batch_ack,json=batchAck" json:"batch_ack,omitempty"`
	Mode                 string            `protobuf:"bytes,4,opt,name=mode" json:"mode,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *RmqInputConf) Reset()         { *m = RmqInputConf{} }
func (m *RmqInputConf) String() string { return proto.CompactTextString(m) }
func (*RmqInputConf) ProtoMessage()    {}
func (*RmqInputConf) Descriptor() ([]byte, []int) {
	return fileDescriptor_consumer_c4c806bed6e9ed0d, []int{2}
}
func (m *RmqInputConf) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RmqInputConf.Unmarshal(m, b)
}
func (m *RmqInputConf) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RmqInputConf.Marshal(b, m, deterministic)
}
func (dst *RmqInputConf) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RmqInputConf.Merge(dst, src)
}
func (m *RmqInputConf) XXX_Size() int {
	return xxx_messageInfo_RmqInputConf.Size(m)
}
func (m *RmqInputConf) XXX_DiscardUnknown() {
	xxx_messageInfo_RmqInputConf.DiscardUnknown(m)
}

var xxx_messageInfo_RmqInputConf proto.InternalMessageInfo

func (m *RmqInputConf) GetExch() *RmqExchQueueInfo {
	if m != nil {
		return m.Exch
	}
	return nil
}

func (m *RmqInputConf) GetQueue() *RmqExchQueueInfo {
	if m != nil {
		return m.Queue
	}
	return nil
}

func (m *RmqInputConf) GetBatchAck() bool {
	if m != nil {
		return m.BatchAck
	}
	return false
}

func (m *RmqInputConf) GetMode() string {
	if m != nil {
		return m.Mode
	}
	return ""
}

type RmqOutRoutingConf struct {
	RoutingKey           string   `protobuf:"bytes,1,opt,name=routing_key,json=routingKey" json:"routing_key,omitempty"`
	Type                 string   `protobuf:"bytes,2,opt,name=type" json:"type,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RmqOutRoutingConf) Reset()         { *m = RmqOutRoutingConf{} }
func (m *RmqOutRoutingConf) String() string { return proto.CompactTextString(m) }
func (*RmqOutRoutingConf) ProtoMessage()    {}
func (*RmqOutRoutingConf) Descriptor() ([]byte, []int) {
	return fileDescriptor_consumer_c4c806bed6e9ed0d, []int{3}
}
func (m *RmqOutRoutingConf) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RmqOutRoutingConf.Unmarshal(m, b)
}
func (m *RmqOutRoutingConf) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RmqOutRoutingConf.Marshal(b, m, deterministic)
}
func (dst *RmqOutRoutingConf) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RmqOutRoutingConf.Merge(dst, src)
}
func (m *RmqOutRoutingConf) XXX_Size() int {
	return xxx_messageInfo_RmqOutRoutingConf.Size(m)
}
func (m *RmqOutRoutingConf) XXX_DiscardUnknown() {
	xxx_messageInfo_RmqOutRoutingConf.DiscardUnknown(m)
}

var xxx_messageInfo_RmqOutRoutingConf proto.InternalMessageInfo

func (m *RmqOutRoutingConf) GetRoutingKey() string {
	if m != nil {
		return m.RoutingKey
	}
	return ""
}

func (m *RmqOutRoutingConf) GetType() string {
	if m != nil {
		return m.Type
	}
	return ""
}

type RmqOutputConf struct {
	Exch                 *RmqExchQueueInfo    `protobuf:"bytes,1,opt,name=exch" json:"exch,omitempty"`
	Outputdefs           []*RmqOutRoutingConf `protobuf:"bytes,2,rep,name=outputdefs" json:"outputdefs,omitempty"`
	Mode                 string               `protobuf:"bytes,4,opt,name=mode" json:"mode,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *RmqOutputConf) Reset()         { *m = RmqOutputConf{} }
func (m *RmqOutputConf) String() string { return proto.CompactTextString(m) }
func (*RmqOutputConf) ProtoMessage()    {}
func (*RmqOutputConf) Descriptor() ([]byte, []int) {
	return fileDescriptor_consumer_c4c806bed6e9ed0d, []int{4}
}
func (m *RmqOutputConf) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RmqOutputConf.Unmarshal(m, b)
}
func (m *RmqOutputConf) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RmqOutputConf.Marshal(b, m, deterministic)
}
func (dst *RmqOutputConf) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RmqOutputConf.Merge(dst, src)
}
func (m *RmqOutputConf) XXX_Size() int {
	return xxx_messageInfo_RmqOutputConf.Size(m)
}
func (m *RmqOutputConf) XXX_DiscardUnknown() {
	xxx_messageInfo_RmqOutputConf.DiscardUnknown(m)
}

var xxx_messageInfo_RmqOutputConf proto.InternalMessageInfo

func (m *RmqOutputConf) GetExch() *RmqExchQueueInfo {
	if m != nil {
		return m.Exch
	}
	return nil
}

func (m *RmqOutputConf) GetOutputdefs() []*RmqOutRoutingConf {
	if m != nil {
		return m.Outputdefs
	}
	return nil
}

func (m *RmqOutputConf) GetMode() string {
	if m != nil {
		return m.Mode
	}
	return ""
}

type RmqTaskConf struct {
	Type                 string           `protobuf:"bytes,1,opt,name=type" json:"type,omitempty"`
	Name                 string           `protobuf:"bytes,2,opt,name=name" json:"name,omitempty"`
	Group                *ConsumerGroup   `protobuf:"bytes,3,opt,name=group" json:"group,omitempty"`
	Inputs               []*RmqInputConf  `protobuf:"bytes,4,rep,name=inputs" json:"inputs,omitempty"`
	Outputs              []*RmqOutputConf `protobuf:"bytes,5,rep,name=outputs" json:"outputs,omitempty"`
	DmsClients           []string         `protobuf:"bytes,6,rep,name=dms_clients,json=dmsClients" json:"dms_clients,omitempty"`
	XXX_NoUnkeyedLiteral struct{}         `json:"-"`
	XXX_unrecognized     []byte           `json:"-"`
	XXX_sizecache        int32            `json:"-"`
}

func (m *RmqTaskConf) Reset()         { *m = RmqTaskConf{} }
func (m *RmqTaskConf) String() string { return proto.CompactTextString(m) }
func (*RmqTaskConf) ProtoMessage()    {}
func (*RmqTaskConf) Descriptor() ([]byte, []int) {
	return fileDescriptor_consumer_c4c806bed6e9ed0d, []int{5}
}
func (m *RmqTaskConf) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RmqTaskConf.Unmarshal(m, b)
}
func (m *RmqTaskConf) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RmqTaskConf.Marshal(b, m, deterministic)
}
func (dst *RmqTaskConf) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RmqTaskConf.Merge(dst, src)
}
func (m *RmqTaskConf) XXX_Size() int {
	return xxx_messageInfo_RmqTaskConf.Size(m)
}
func (m *RmqTaskConf) XXX_DiscardUnknown() {
	xxx_messageInfo_RmqTaskConf.DiscardUnknown(m)
}

var xxx_messageInfo_RmqTaskConf proto.InternalMessageInfo

func (m *RmqTaskConf) GetType() string {
	if m != nil {
		return m.Type
	}
	return ""
}

func (m *RmqTaskConf) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *RmqTaskConf) GetGroup() *ConsumerGroup {
	if m != nil {
		return m.Group
	}
	return nil
}

func (m *RmqTaskConf) GetInputs() []*RmqInputConf {
	if m != nil {
		return m.Inputs
	}
	return nil
}

func (m *RmqTaskConf) GetOutputs() []*RmqOutputConf {
	if m != nil {
		return m.Outputs
	}
	return nil
}

func (m *RmqTaskConf) GetDmsClients() []string {
	if m != nil {
		return m.DmsClients
	}
	return nil
}

type DelayQueueConf struct {
	Name                 string   `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	Output               []string `protobuf:"bytes,6,rep,name=output" json:"output,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DelayQueueConf) Reset()         { *m = DelayQueueConf{} }
func (m *DelayQueueConf) String() string { return proto.CompactTextString(m) }
func (*DelayQueueConf) ProtoMessage()    {}
func (*DelayQueueConf) Descriptor() ([]byte, []int) {
	return fileDescriptor_consumer_c4c806bed6e9ed0d, []int{6}
}
func (m *DelayQueueConf) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DelayQueueConf.Unmarshal(m, b)
}
func (m *DelayQueueConf) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DelayQueueConf.Marshal(b, m, deterministic)
}
func (dst *DelayQueueConf) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DelayQueueConf.Merge(dst, src)
}
func (m *DelayQueueConf) XXX_Size() int {
	return xxx_messageInfo_DelayQueueConf.Size(m)
}
func (m *DelayQueueConf) XXX_DiscardUnknown() {
	xxx_messageInfo_DelayQueueConf.DiscardUnknown(m)
}

var xxx_messageInfo_DelayQueueConf proto.InternalMessageInfo

func (m *DelayQueueConf) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *DelayQueueConf) GetOutput() []string {
	if m != nil {
		return m.Output
	}
	return nil
}

func init() {
	proto.RegisterType((*RmqExchQueueInfo)(nil), "exmsg.core.models.RmqExchQueueInfo")
	proto.RegisterType((*ConsumerGroup)(nil), "exmsg.core.models.ConsumerGroup")
	proto.RegisterType((*RmqInputConf)(nil), "exmsg.core.models.RmqInputConf")
	proto.RegisterType((*RmqOutRoutingConf)(nil), "exmsg.core.models.RmqOutRoutingConf")
	proto.RegisterType((*RmqOutputConf)(nil), "exmsg.core.models.RmqOutputConf")
	proto.RegisterType((*RmqTaskConf)(nil), "exmsg.core.models.RmqTaskConf")
	proto.RegisterType((*DelayQueueConf)(nil), "exmsg.core.models.DelayQueueConf")
}

func init() {
	proto.RegisterFile("proto/core/models/consumer.proto", fileDescriptor_consumer_c4c806bed6e9ed0d)
}

var fileDescriptor_consumer_c4c806bed6e9ed0d = []byte{
	// 554 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xa4, 0x54, 0x4d, 0x6f, 0xd3, 0x40,
	0x10, 0x95, 0xe3, 0xc4, 0x6d, 0x26, 0x14, 0xb5, 0x7b, 0xa8, 0x56, 0x80, 0x54, 0xcb, 0xe5, 0x90,
	0x93, 0x2d, 0x15, 0x89, 0xaa, 0x94, 0x0b, 0x24, 0x08, 0x2a, 0x0e, 0x88, 0x15, 0x27, 0x2e, 0x91,
	0x3f, 0x26, 0x89, 0xe5, 0x8f, 0x8d, 0xbd, 0xeb, 0x92, 0x9c, 0xf9, 0x11, 0xdc, 0xf9, 0x0b, 0xfc,
	0x41, 0xb4, 0x6b, 0xbb, 0xe4, 0xc3, 0x48, 0x48, 0xdc, 0x66, 0xdf, 0xcc, 0x9b, 0xcc, 0x7b, 0x33,
	0x31, 0xd8, 0xab, 0x92, 0x4b, 0xee, 0x85, 0xbc, 0x44, 0x2f, 0xe3, 0x11, 0xa6, 0xc2, 0x0b, 0x79,
	0x2e, 0xaa, 0x0c, 0x4b, 0x57, 0xa7, 0xc8, 0x19, 0xae, 0x33, 0xb1, 0x70, 0x55, 0x85, 0x5b, 0x57,
	0x38, 0x3f, 0x7a, 0x70, 0xca, 0xb2, 0xe2, 0xdd, 0x3a, 0x5c, 0x7e, 0xae, 0xb0, 0xc2, 0xbb, 0x7c,
	0xce, 0x09, 0x81, 0x7e, 0xee, 0x67, 0x48, 0x0d, 0xdb, 0x18, 0x0f, 0x99, 0x8e, 0x15, 0x26, 0x37,
	0x2b, 0xa4, 0xbd, 0x1a, 0x53, 0x31, 0xb9, 0x80, 0x91, 0x5f, 0x49, 0x3e, 0x8b, 0x30, 0x45, 0x89,
	0xd4, 0xb4, 0x8d, 0xf1, 0x31, 0x03, 0x05, 0x4d, 0x35, 0x42, 0x28, 0x1c, 0x45, 0x55, 0xe9, 0x07,
	0x29, 0xd2, 0xbe, 0x4e, 0xb6, 0x4f, 0xf2, 0x04, 0x8e, 0xe3, 0x5c, 0x62, 0x99, 0xfb, 0x29, 0x1d,
	0xe8, 0xd4, 0xc3, 0x9b, 0x3c, 0x83, 0x21, 0xae, 0xc3, 0xb4, 0x12, 0xf1, 0x3d, 0x52, 0x4b, 0x27,
	0xff, 0x00, 0xe4, 0x1c, 0xac, 0x9c, 0x7f, 0xf3, 0x63, 0x49, 0x8f, 0x74, 0xaa, 0x79, 0xa9, 0x61,
	0x4a, 0x5e, 0xc9, 0x38, 0x5f, 0xcc, 0x12, 0xdc, 0xd0, 0x63, 0x3d, 0x27, 0x34, 0xd0, 0x47, 0xdc,
	0x90, 0xa7, 0x30, 0x0c, 0x7c, 0x19, 0x2e, 0x67, 0x7e, 0x98, 0xd0, 0x61, 0xfd, 0x9b, 0x1a, 0x78,
	0x13, 0x26, 0xe4, 0x14, 0xcc, 0x82, 0x0b, 0x0a, 0xb6, 0x31, 0x36, 0x99, 0x0a, 0x9d, 0x4b, 0x38,
	0x99, 0x34, 0xf6, 0xbd, 0x2f, 0x79, 0xb5, 0xea, 0x72, 0xc5, 0xf9, 0x65, 0xc0, 0x23, 0x96, 0x15,
	0x77, 0xf9, 0xaa, 0x92, 0x13, 0x9e, 0xcf, 0xc9, 0x35, 0xf4, 0x71, 0x1d, 0x2e, 0x75, 0xd1, 0xe8,
	0xea, 0xd2, 0x3d, 0x70, 0xdc, 0xdd, 0x77, 0x9b, 0x69, 0x02, 0xb9, 0x81, 0x41, 0xa1, 0x20, 0x6d,
	0xf0, 0x3f, 0x32, 0x6b, 0xc6, 0xae, 0x30, 0x73, 0x4f, 0x18, 0x81, 0xbe, 0xa2, 0x6b, 0xff, 0x87,
	0x4c, 0xc7, 0xce, 0x07, 0x38, 0x63, 0x59, 0xf1, 0xa9, 0x92, 0xac, 0x76, 0x47, 0x4f, 0xbe, 0xe7,
	0x9f, 0x71, 0xe0, 0x5f, 0xc7, 0x05, 0x38, 0x3f, 0x0d, 0x38, 0xa9, 0x5b, 0xfd, 0xb7, 0x01, 0x53,
	0x00, 0xae, 0xdb, 0x44, 0x38, 0x17, 0xb4, 0x67, 0x9b, 0xe3, 0xd1, 0xd5, 0xf3, 0x6e, 0xfa, 0xee,
	0xe4, 0x6c, 0x8b, 0xd7, 0x29, 0xf7, 0x7b, 0x0f, 0x46, 0x2c, 0x2b, 0xbe, 0xf8, 0x22, 0xd1, 0x23,
	0xb6, 0x42, 0x8c, 0xad, 0x53, 0x6e, 0x97, 0xdb, 0xdb, 0x3a, 0xf9, 0x97, 0x30, 0x58, 0xa8, 0xcd,
	0x6b, 0x4f, 0x47, 0x57, 0x76, 0xc7, 0x30, 0x3b, 0x17, 0xc2, 0xea, 0x72, 0x72, 0x0d, 0x56, 0xac,
	0x0e, 0x42, 0xd0, 0xbe, 0x56, 0x71, 0xd1, 0xad, 0xe2, 0xe1, 0x68, 0x58, 0x53, 0x4e, 0x5e, 0xc1,
	0x51, 0x2d, 0x45, 0xd0, 0x81, 0x66, 0xda, 0x7f, 0xd5, 0xdf, 0x52, 0x5b, 0x82, 0x5a, 0x5f, 0x94,
	0x89, 0x59, 0x98, 0xc6, 0x98, 0x4b, 0x41, 0x2d, 0xdb, 0x54, 0xeb, 0x8b, 0x32, 0x31, 0xa9, 0x11,
	0xe7, 0x35, 0x3c, 0x9e, 0x62, 0xea, 0x6f, 0xb4, 0xef, 0xad, 0x0f, 0x07, 0x7f, 0xf3, 0x73, 0xb0,
	0xea, 0x8e, 0x4d, 0x87, 0xe6, 0xf5, 0xf6, 0xf6, 0xeb, 0x4d, 0x10, 0xcb, 0xa0, 0x0a, 0x13, 0x94,
	0x2e, 0x2f, 0x17, 0x5e, 0x50, 0xf2, 0x08, 0xef, 0xbd, 0x00, 0x31, 0x89, 0xa5, 0xa7, 0xc7, 0xdc,
	0xfe, 0xf4, 0xdc, 0xaa, 0x78, 0x56, 0xc7, 0x81, 0xa5, 0x3f, 0x3f, 0x2f, 0x7e, 0x07, 0x00, 0x00,
	0xff, 0xff, 0xc2, 0x8e, 0x75, 0x35, 0xa2, 0x04, 0x00, 0x00,
}
