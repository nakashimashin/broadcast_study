// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v3.21.12
// source: match.proto

package grpc

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
	MatchRoom_Matching_FullMethodName = "/match.MatchRoom/Matching"
)

// MatchRoomClient is the client API for MatchRoom service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MatchRoomClient interface {
	Matching(ctx context.Context, in *MatchRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[MatchResponse], error)
}

type matchRoomClient struct {
	cc grpc.ClientConnInterface
}

func NewMatchRoomClient(cc grpc.ClientConnInterface) MatchRoomClient {
	return &matchRoomClient{cc}
}

func (c *matchRoomClient) Matching(ctx context.Context, in *MatchRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[MatchResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &MatchRoom_ServiceDesc.Streams[0], MatchRoom_Matching_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[MatchRequest, MatchResponse]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type MatchRoom_MatchingClient = grpc.ServerStreamingClient[MatchResponse]

// MatchRoomServer is the server API for MatchRoom service.
// All implementations must embed UnimplementedMatchRoomServer
// for forward compatibility.
type MatchRoomServer interface {
	Matching(*MatchRequest, grpc.ServerStreamingServer[MatchResponse]) error
	mustEmbedUnimplementedMatchRoomServer()
}

// UnimplementedMatchRoomServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedMatchRoomServer struct{}

func (UnimplementedMatchRoomServer) Matching(*MatchRequest, grpc.ServerStreamingServer[MatchResponse]) error {
	return status.Errorf(codes.Unimplemented, "method Matching not implemented")
}
func (UnimplementedMatchRoomServer) mustEmbedUnimplementedMatchRoomServer() {}
func (UnimplementedMatchRoomServer) testEmbeddedByValue()                   {}

// UnsafeMatchRoomServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MatchRoomServer will
// result in compilation errors.
type UnsafeMatchRoomServer interface {
	mustEmbedUnimplementedMatchRoomServer()
}

func RegisterMatchRoomServer(s grpc.ServiceRegistrar, srv MatchRoomServer) {
	// If the following call pancis, it indicates UnimplementedMatchRoomServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&MatchRoom_ServiceDesc, srv)
}

func _MatchRoom_Matching_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(MatchRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(MatchRoomServer).Matching(m, &grpc.GenericServerStream[MatchRequest, MatchResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type MatchRoom_MatchingServer = grpc.ServerStreamingServer[MatchResponse]

// MatchRoom_ServiceDesc is the grpc.ServiceDesc for MatchRoom service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var MatchRoom_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "match.MatchRoom",
	HandlerType: (*MatchRoomServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Matching",
			Handler:       _MatchRoom_Matching_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "match.proto",
}