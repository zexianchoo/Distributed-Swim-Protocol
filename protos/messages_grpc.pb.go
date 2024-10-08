// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v4.25.1
// source: messages.proto

package messages

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	LogQueryService_GetLogs_FullMethodName = "/messages.LogQueryService/GetLogs"
)

// LogQueryServiceClient is the client API for LogQueryService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// import "google/protobuf/empty.proto";
type LogQueryServiceClient interface {
	GetLogs(ctx context.Context, in *GrepRequest, opts ...grpc.CallOption) (*GrepReply, error)
}

type logQueryServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewLogQueryServiceClient(cc grpc.ClientConnInterface) LogQueryServiceClient {
	return &logQueryServiceClient{cc}
}

func (c *logQueryServiceClient) GetLogs(ctx context.Context, in *GrepRequest, opts ...grpc.CallOption) (*GrepReply, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GrepReply)
	err := c.cc.Invoke(ctx, LogQueryService_GetLogs_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// LogQueryServiceServer is the server API for LogQueryService service.
// All implementations must embed UnimplementedLogQueryServiceServer
// for forward compatibility.
//
// import "google/protobuf/empty.proto";
type LogQueryServiceServer interface {
	GetLogs(context.Context, *GrepRequest) (*GrepReply, error)
	mustEmbedUnimplementedLogQueryServiceServer()
}

// UnimplementedLogQueryServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedLogQueryServiceServer struct{}

func (UnimplementedLogQueryServiceServer) GetLogs(context.Context, *GrepRequest) (*GrepReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetLogs not implemented")
}
func (UnimplementedLogQueryServiceServer) mustEmbedUnimplementedLogQueryServiceServer() {}
func (UnimplementedLogQueryServiceServer) testEmbeddedByValue()                         {}

// UnsafeLogQueryServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to LogQueryServiceServer will
// result in compilation errors.
type UnsafeLogQueryServiceServer interface {
	mustEmbedUnimplementedLogQueryServiceServer()
}

func RegisterLogQueryServiceServer(s grpc.ServiceRegistrar, srv LogQueryServiceServer) {
	// If the following call pancis, it indicates UnimplementedLogQueryServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&LogQueryService_ServiceDesc, srv)
}

func _LogQueryService_GetLogs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GrepRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LogQueryServiceServer).GetLogs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: LogQueryService_GetLogs_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LogQueryServiceServer).GetLogs(ctx, req.(*GrepRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// LogQueryService_ServiceDesc is the grpc.ServiceDesc for LogQueryService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var LogQueryService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "messages.LogQueryService",
	HandlerType: (*LogQueryServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetLogs",
			Handler:    _LogQueryService_GetLogs_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "messages.proto",
}
