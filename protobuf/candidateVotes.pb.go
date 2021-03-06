// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.22.0-devel
// 	protoc        (unknown)
// source: protobuf/candidateVotes.proto

package protobuf

import (
	proto "github.com/golang/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type CandidateVotesRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// curent term
	Term uint64 `protobuf:"varint,1,opt,name=term,proto3" json:"term,omitempty"`
	// candidate id
	NodeId uint32 `protobuf:"varint,2,opt,name=nodeId,proto3" json:"nodeId,omitempty"`
	// previous entry index
	PrevEntryIndex uint64 `protobuf:"varint,3,opt,name=prevEntryIndex,proto3" json:"prevEntryIndex,omitempty"`
	// previous entry term
	PrevEntryTerm uint64 `protobuf:"varint,4,opt,name=prevEntryTerm,proto3" json:"prevEntryTerm,omitempty"`
}

func (x *CandidateVotesRequest) Reset() {
	*x = CandidateVotesRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protobuf_candidateVotes_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CandidateVotesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CandidateVotesRequest) ProtoMessage() {}

func (x *CandidateVotesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_protobuf_candidateVotes_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CandidateVotesRequest.ProtoReflect.Descriptor instead.
func (*CandidateVotesRequest) Descriptor() ([]byte, []int) {
	return file_protobuf_candidateVotes_proto_rawDescGZIP(), []int{0}
}

func (x *CandidateVotesRequest) GetTerm() uint64 {
	if x != nil {
		return x.Term
	}
	return 0
}

func (x *CandidateVotesRequest) GetNodeId() uint32 {
	if x != nil {
		return x.NodeId
	}
	return 0
}

func (x *CandidateVotesRequest) GetPrevEntryIndex() uint64 {
	if x != nil {
		return x.PrevEntryIndex
	}
	return 0
}

func (x *CandidateVotesRequest) GetPrevEntryTerm() uint64 {
	if x != nil {
		return x.PrevEntryTerm
	}
	return 0
}

type CandidateVotesResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// current term
	Term uint64 `protobuf:"varint,1,opt,name=term,proto3" json:"term,omitempty"`
	// node id
	NodeId uint32 `protobuf:"varint,2,opt,name=nodeId,proto3" json:"nodeId,omitempty"`
	// boolean tag for whether the node accepts candidate
	Accepted bool `protobuf:"varint,3,opt,name=accepted,proto3" json:"accepted,omitempty"`
}

func (x *CandidateVotesResponse) Reset() {
	*x = CandidateVotesResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_protobuf_candidateVotes_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CandidateVotesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CandidateVotesResponse) ProtoMessage() {}

func (x *CandidateVotesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_protobuf_candidateVotes_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CandidateVotesResponse.ProtoReflect.Descriptor instead.
func (*CandidateVotesResponse) Descriptor() ([]byte, []int) {
	return file_protobuf_candidateVotes_proto_rawDescGZIP(), []int{1}
}

func (x *CandidateVotesResponse) GetTerm() uint64 {
	if x != nil {
		return x.Term
	}
	return 0
}

func (x *CandidateVotesResponse) GetNodeId() uint32 {
	if x != nil {
		return x.NodeId
	}
	return 0
}

func (x *CandidateVotesResponse) GetAccepted() bool {
	if x != nil {
		return x.Accepted
	}
	return false
}

var File_protobuf_candidateVotes_proto protoreflect.FileDescriptor

var file_protobuf_candidateVotes_proto_rawDesc = []byte{
	0x0a, 0x1d, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x63, 0x61, 0x6e, 0x64, 0x69,
	0x64, 0x61, 0x74, 0x65, 0x56, 0x6f, 0x74, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x08, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x22, 0x91, 0x01, 0x0a, 0x15, 0x43, 0x61,
	0x6e, 0x64, 0x69, 0x64, 0x61, 0x74, 0x65, 0x56, 0x6f, 0x74, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x04, 0x52, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x12, 0x16, 0x0a, 0x06, 0x6e, 0x6f, 0x64, 0x65, 0x49,
	0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x06, 0x6e, 0x6f, 0x64, 0x65, 0x49, 0x64, 0x12,
	0x26, 0x0a, 0x0e, 0x70, 0x72, 0x65, 0x76, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x49, 0x6e, 0x64, 0x65,
	0x78, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x52, 0x0e, 0x70, 0x72, 0x65, 0x76, 0x45, 0x6e, 0x74,
	0x72, 0x79, 0x49, 0x6e, 0x64, 0x65, 0x78, 0x12, 0x24, 0x0a, 0x0d, 0x70, 0x72, 0x65, 0x76, 0x45,
	0x6e, 0x74, 0x72, 0x79, 0x54, 0x65, 0x72, 0x6d, 0x18, 0x04, 0x20, 0x01, 0x28, 0x04, 0x52, 0x0d,
	0x70, 0x72, 0x65, 0x76, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x54, 0x65, 0x72, 0x6d, 0x22, 0x60, 0x0a,
	0x16, 0x43, 0x61, 0x6e, 0x64, 0x69, 0x64, 0x61, 0x74, 0x65, 0x56, 0x6f, 0x74, 0x65, 0x73, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x12, 0x16, 0x0a, 0x06, 0x6e,
	0x6f, 0x64, 0x65, 0x49, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x06, 0x6e, 0x6f, 0x64,
	0x65, 0x49, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x61, 0x63, 0x63, 0x65, 0x70, 0x74, 0x65, 0x64, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x61, 0x63, 0x63, 0x65, 0x70, 0x74, 0x65, 0x64, 0x42,
	0x20, 0x5a, 0x1e, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x6c,
	0x6f, 0x63, 0x6b, 0x5f, 0x72, 0x61, 0x66, 0x74, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_protobuf_candidateVotes_proto_rawDescOnce sync.Once
	file_protobuf_candidateVotes_proto_rawDescData = file_protobuf_candidateVotes_proto_rawDesc
)

func file_protobuf_candidateVotes_proto_rawDescGZIP() []byte {
	file_protobuf_candidateVotes_proto_rawDescOnce.Do(func() {
		file_protobuf_candidateVotes_proto_rawDescData = protoimpl.X.CompressGZIP(file_protobuf_candidateVotes_proto_rawDescData)
	})
	return file_protobuf_candidateVotes_proto_rawDescData
}

var file_protobuf_candidateVotes_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_protobuf_candidateVotes_proto_goTypes = []interface{}{
	(*CandidateVotesRequest)(nil),  // 0: protobuf.CandidateVotesRequest
	(*CandidateVotesResponse)(nil), // 1: protobuf.CandidateVotesResponse
}
var file_protobuf_candidateVotes_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_protobuf_candidateVotes_proto_init() }
func file_protobuf_candidateVotes_proto_init() {
	if File_protobuf_candidateVotes_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_protobuf_candidateVotes_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CandidateVotesRequest); i {
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
		file_protobuf_candidateVotes_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CandidateVotesResponse); i {
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
			RawDescriptor: file_protobuf_candidateVotes_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_protobuf_candidateVotes_proto_goTypes,
		DependencyIndexes: file_protobuf_candidateVotes_proto_depIdxs,
		MessageInfos:      file_protobuf_candidateVotes_proto_msgTypes,
	}.Build()
	File_protobuf_candidateVotes_proto = out.File
	file_protobuf_candidateVotes_proto_rawDesc = nil
	file_protobuf_candidateVotes_proto_goTypes = nil
	file_protobuf_candidateVotes_proto_depIdxs = nil
}
