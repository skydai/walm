// Code generated by protoc-gen-go. DO NOT EDIT.
// source: google/ads/googleads/v0/enums/geo_target_constant_status.proto

package enums // import "google.golang.org/genproto/googleapis/ads/googleads/v0/enums"

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

// The possible statuses of a geo target constant.
type GeoTargetConstantStatusEnum_GeoTargetConstantStatus int32

const (
	// No value has been specified.
	GeoTargetConstantStatusEnum_UNSPECIFIED GeoTargetConstantStatusEnum_GeoTargetConstantStatus = 0
	// The received value is not known in this version.
	//
	// This is a response-only value.
	GeoTargetConstantStatusEnum_UNKNOWN GeoTargetConstantStatusEnum_GeoTargetConstantStatus = 1
	// The geo target constant is valid.
	GeoTargetConstantStatusEnum_ENABLED GeoTargetConstantStatusEnum_GeoTargetConstantStatus = 2
	// The geo target constant is obsolete and will be removed.
	GeoTargetConstantStatusEnum_REMOVAL_PLANNED GeoTargetConstantStatusEnum_GeoTargetConstantStatus = 3
)

var GeoTargetConstantStatusEnum_GeoTargetConstantStatus_name = map[int32]string{
	0: "UNSPECIFIED",
	1: "UNKNOWN",
	2: "ENABLED",
	3: "REMOVAL_PLANNED",
}
var GeoTargetConstantStatusEnum_GeoTargetConstantStatus_value = map[string]int32{
	"UNSPECIFIED":     0,
	"UNKNOWN":         1,
	"ENABLED":         2,
	"REMOVAL_PLANNED": 3,
}

func (x GeoTargetConstantStatusEnum_GeoTargetConstantStatus) String() string {
	return proto.EnumName(GeoTargetConstantStatusEnum_GeoTargetConstantStatus_name, int32(x))
}
func (GeoTargetConstantStatusEnum_GeoTargetConstantStatus) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_geo_target_constant_status_610eb5bb0ff353bb, []int{0, 0}
}

// Container for describing the status of a geo target constant.
type GeoTargetConstantStatusEnum struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GeoTargetConstantStatusEnum) Reset()         { *m = GeoTargetConstantStatusEnum{} }
func (m *GeoTargetConstantStatusEnum) String() string { return proto.CompactTextString(m) }
func (*GeoTargetConstantStatusEnum) ProtoMessage()    {}
func (*GeoTargetConstantStatusEnum) Descriptor() ([]byte, []int) {
	return fileDescriptor_geo_target_constant_status_610eb5bb0ff353bb, []int{0}
}
func (m *GeoTargetConstantStatusEnum) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GeoTargetConstantStatusEnum.Unmarshal(m, b)
}
func (m *GeoTargetConstantStatusEnum) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GeoTargetConstantStatusEnum.Marshal(b, m, deterministic)
}
func (dst *GeoTargetConstantStatusEnum) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GeoTargetConstantStatusEnum.Merge(dst, src)
}
func (m *GeoTargetConstantStatusEnum) XXX_Size() int {
	return xxx_messageInfo_GeoTargetConstantStatusEnum.Size(m)
}
func (m *GeoTargetConstantStatusEnum) XXX_DiscardUnknown() {
	xxx_messageInfo_GeoTargetConstantStatusEnum.DiscardUnknown(m)
}

var xxx_messageInfo_GeoTargetConstantStatusEnum proto.InternalMessageInfo

func init() {
	proto.RegisterType((*GeoTargetConstantStatusEnum)(nil), "google.ads.googleads.v0.enums.GeoTargetConstantStatusEnum")
	proto.RegisterEnum("google.ads.googleads.v0.enums.GeoTargetConstantStatusEnum_GeoTargetConstantStatus", GeoTargetConstantStatusEnum_GeoTargetConstantStatus_name, GeoTargetConstantStatusEnum_GeoTargetConstantStatus_value)
}

func init() {
	proto.RegisterFile("google/ads/googleads/v0/enums/geo_target_constant_status.proto", fileDescriptor_geo_target_constant_status_610eb5bb0ff353bb)
}

var fileDescriptor_geo_target_constant_status_610eb5bb0ff353bb = []byte{
	// 306 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x90, 0xc1, 0x4a, 0xc3, 0x30,
	0x18, 0xc7, 0x5d, 0x07, 0x0a, 0xd9, 0x61, 0xa5, 0x1e, 0x3c, 0xe8, 0x0e, 0xdb, 0x03, 0xa4, 0x05,
	0x6f, 0x11, 0x84, 0x74, 0x8b, 0x63, 0x58, 0xb3, 0xe2, 0x5c, 0x45, 0x29, 0x94, 0xb8, 0x86, 0x20,
	0xac, 0xc9, 0x68, 0xd2, 0xe1, 0xf3, 0x78, 0xf4, 0x51, 0x7c, 0x14, 0x6f, 0xbe, 0x81, 0x34, 0xdd,
	0x7a, 0xab, 0x97, 0xf0, 0x4f, 0xfe, 0xdf, 0xef, 0xcb, 0xf7, 0xfd, 0xc1, 0xad, 0x50, 0x4a, 0x6c,
	0xb9, 0xcf, 0x72, 0xed, 0x37, 0xb2, 0x56, 0xfb, 0xc0, 0xe7, 0xb2, 0x2a, 0xb4, 0x2f, 0xb8, 0xca,
	0x0c, 0x2b, 0x05, 0x37, 0xd9, 0x46, 0x49, 0x6d, 0x98, 0x34, 0x99, 0x36, 0xcc, 0x54, 0x1a, 0xee,
	0x4a, 0x65, 0x94, 0x37, 0x6a, 0x20, 0xc8, 0x72, 0x0d, 0x5b, 0x1e, 0xee, 0x03, 0x68, 0xf9, 0xc9,
	0x07, 0xb8, 0x9c, 0x73, 0xf5, 0x64, 0x3b, 0x4c, 0x0f, 0x0d, 0x56, 0x96, 0x27, 0xb2, 0x2a, 0x26,
	0x2f, 0xe0, 0xa2, 0xc3, 0xf6, 0x86, 0x60, 0xb0, 0xa6, 0xab, 0x98, 0x4c, 0x17, 0x77, 0x0b, 0x32,
	0x73, 0x4f, 0xbc, 0x01, 0x38, 0x5b, 0xd3, 0x7b, 0xba, 0x7c, 0xa6, 0x6e, 0xaf, 0xbe, 0x10, 0x8a,
	0xc3, 0x88, 0xcc, 0x5c, 0xc7, 0x3b, 0x07, 0xc3, 0x47, 0xf2, 0xb0, 0x4c, 0x70, 0x94, 0xc5, 0x11,
	0xa6, 0x94, 0xcc, 0xdc, 0x7e, 0xf8, 0xdb, 0x03, 0xe3, 0x8d, 0x2a, 0xe0, 0xbf, 0xf3, 0x85, 0x57,
	0x1d, 0xdf, 0xc7, 0xf5, 0x72, 0x71, 0xef, 0x35, 0x3c, 0xe0, 0x42, 0x6d, 0x99, 0x14, 0x50, 0x95,
	0xc2, 0x17, 0x5c, 0xda, 0xd5, 0x8f, 0x71, 0xed, 0xde, 0x75, 0x47, 0x7a, 0x37, 0xf6, 0xfc, 0x74,
	0xfa, 0x73, 0x8c, 0xbf, 0x9c, 0xd1, 0xbc, 0x69, 0x85, 0x73, 0x0d, 0x1b, 0x59, 0xab, 0x24, 0x80,
	0x75, 0x12, 0xfa, 0xfb, 0xe8, 0xa7, 0x38, 0xd7, 0x69, 0xeb, 0xa7, 0x49, 0x90, 0x5a, 0xff, 0xc7,
	0x19, 0x37, 0x8f, 0x08, 0xe1, 0x5c, 0x23, 0xd4, 0x56, 0x20, 0x94, 0x04, 0x08, 0xd9, 0x9a, 0xb7,
	0x53, 0x3b, 0xd8, 0xf5, 0x5f, 0x00, 0x00, 0x00, 0xff, 0xff, 0x37, 0xda, 0x4a, 0x2b, 0xd5, 0x01,
	0x00, 0x00,
}
