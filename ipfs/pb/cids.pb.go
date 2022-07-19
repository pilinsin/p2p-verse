// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.0
// 	protoc        v3.19.4
// source: cids.proto

package pb

import (
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

type BlockCids struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Cids []string `protobuf:"bytes,1,rep,name=cids,proto3" json:"cids,omitempty"`
}

func (x *BlockCids) Reset() {
	*x = BlockCids{}
	if protoimpl.UnsafeEnabled {
		mi := &file_cids_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BlockCids) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BlockCids) ProtoMessage() {}

func (x *BlockCids) ProtoReflect() protoreflect.Message {
	mi := &file_cids_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BlockCids.ProtoReflect.Descriptor instead.
func (*BlockCids) Descriptor() ([]byte, []int) {
	return file_cids_proto_rawDescGZIP(), []int{0}
}

func (x *BlockCids) GetCids() []string {
	if x != nil {
		return x.Cids
	}
	return nil
}

var File_cids_proto protoreflect.FileDescriptor

var file_cids_proto_rawDesc = []byte{
	0x0a, 0x0a, 0x63, 0x69, 0x64, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x02, 0x70, 0x62,
	0x22, 0x1f, 0x0a, 0x09, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x43, 0x69, 0x64, 0x73, 0x12, 0x12, 0x0a,
	0x04, 0x63, 0x69, 0x64, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x04, 0x63, 0x69, 0x64,
	0x73, 0x42, 0x07, 0x5a, 0x05, 0x2e, 0x2f, 0x3b, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_cids_proto_rawDescOnce sync.Once
	file_cids_proto_rawDescData = file_cids_proto_rawDesc
)

func file_cids_proto_rawDescGZIP() []byte {
	file_cids_proto_rawDescOnce.Do(func() {
		file_cids_proto_rawDescData = protoimpl.X.CompressGZIP(file_cids_proto_rawDescData)
	})
	return file_cids_proto_rawDescData
}

var file_cids_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_cids_proto_goTypes = []interface{}{
	(*BlockCids)(nil), // 0: pb.BlockCids
}
var file_cids_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_cids_proto_init() }
func file_cids_proto_init() {
	if File_cids_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_cids_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BlockCids); i {
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
			RawDescriptor: file_cids_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_cids_proto_goTypes,
		DependencyIndexes: file_cids_proto_depIdxs,
		MessageInfos:      file_cids_proto_msgTypes,
	}.Build()
	File_cids_proto = out.File
	file_cids_proto_rawDesc = nil
	file_cids_proto_goTypes = nil
	file_cids_proto_depIdxs = nil
}