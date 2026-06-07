package interceptors

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type loggingTestServerStream struct {
	ctx context.Context
}

func (s loggingTestServerStream) SetHeader(metadata.MD) error  { return nil }
func (s loggingTestServerStream) SendHeader(metadata.MD) error { return nil }
func (s loggingTestServerStream) SetTrailer(metadata.MD)       {}
func (s loggingTestServerStream) Context() context.Context     { return s.ctx }
func (s loggingTestServerStream) SendMsg(any) error            { return nil }
func (s loggingTestServerStream) RecvMsg(any) error            { return nil }

func TestUnaryLoggingInterceptorLogsRequest(t *testing.T) {
	core, observed := observer.New(zap.InfoLevel)
	li := NewLoggingInterceptor(zap.New(core))

	res, err := li.UnaryLoggingInterceptor()(context.Background(), "req", &grpc.UnaryServerInfo{FullMethod: "/orders.OrdersService/GetOrderByID"}, func(ctx context.Context, req any) (any, error) {
		return "ok", status.Error(codes.NotFound, "missing")
	})
	require.Error(t, err)
	assert.Equal(t, "ok", res)

	entries := observed.All()
	require.Len(t, entries, 1)
	assert.Equal(t, "gRPC request", entries[0].Message)
	assert.Equal(t, "/orders.OrdersService/GetOrderByID", entries[0].ContextMap()["method"])
	assert.Equal(t, "NotFound", entries[0].ContextMap()["status"])
}

func TestUnaryClientLoggingInterceptorLogsResponse(t *testing.T) {
	core, observed := observer.New(zap.InfoLevel)
	li := NewLoggingInterceptor(zap.New(core))

	called := false
	err := li.UnaryClientLoggingInterceptor()(context.Background(), "/orders.OrdersService/GetOrderByID", "req", nil, nil, func(ctx context.Context, method string, req any, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		called = true
		return nil
	})
	require.NoError(t, err)
	assert.True(t, called)

	entries := observed.All()
	require.Len(t, entries, 1)
	assert.Equal(t, "gRPC response", entries[0].Message)
	assert.Equal(t, "/orders.OrdersService/GetOrderByID", entries[0].ContextMap()["method"])
	assert.Equal(t, "OK", entries[0].ContextMap()["status"])
}

func TestStreamLoggingInterceptorLogsStartAndEnd(t *testing.T) {
	core, observed := observer.New(zap.InfoLevel)
	li := NewLoggingInterceptor(zap.New(core))

	called := false
	err := li.StreamLoggingInterceptor()(nil, loggingTestServerStream{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: "/orders.OrdersService/GetOrdersByCustomerID", IsServerStream: true}, func(any, grpc.ServerStream) error {
		called = true
		return errors.New("stream failed")
	})
	require.Error(t, err)
	assert.True(t, called)

	entries := observed.All()
	require.Len(t, entries, 2)
	assert.Equal(t, "gRPC stream started", entries[0].Message)
	assert.Equal(t, "/orders.OrdersService/GetOrdersByCustomerID", entries[0].ContextMap()["method"])
	assert.Equal(t, true, entries[0].ContextMap()["is_server_stream"])

	assert.Equal(t, "gRPC stream", entries[1].Message)
	assert.Equal(t, "/orders.OrdersService/GetOrdersByCustomerID", entries[1].ContextMap()["method"])
	assert.Equal(t, "Unknown", entries[1].ContextMap()["statsu"])
	assert.Equal(t, true, entries[1].ContextMap()["is_server_stream"])
}
