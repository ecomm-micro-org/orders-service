package interceptors

import (
	"context"
	"errors"
	"testing"

	internalauth "github.com/risbern21/runaway/orders-service/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeValidator struct {
	claims        *internalauth.UserClaims
	err           error
	receivedToken string
}

func (f *fakeValidator) ValidateToken(_ context.Context, token string) (*internalauth.UserClaims, error) {
	f.receivedToken = token
	if f.err != nil {
		return nil, f.err
	}
	return f.claims, nil
}

type authTestServerStream struct {
	ctx context.Context
}

func (s authTestServerStream) SetHeader(metadata.MD) error  { return nil }
func (s authTestServerStream) SendHeader(metadata.MD) error { return nil }
func (s authTestServerStream) SetTrailer(metadata.MD)       {}
func (s authTestServerStream) Context() context.Context     { return s.ctx }
func (s authTestServerStream) SendMsg(any) error            { return nil }
func (s authTestServerStream) RecvMsg(any) error            { return nil }

func TestNewAuthInterceptorRejectsNilValidator(t *testing.T) {
	_, err := NewAuthInterceptor(nil)
	require.Error(t, err)
}

func TestUnaryAuthInterceptorRejectsMissingMetadata(t *testing.T) {
	ai, err := NewAuthInterceptor(&fakeValidator{})
	require.NoError(t, err)

	_, err = ai.UnaryAuthInterceptor()(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/orders.OrdersService/GetOrderByID"}, func(context.Context, any) (any, error) {
		return nil, nil
	})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestUnaryAuthInterceptorRejectsMissingAuthorizationHeader(t *testing.T) {
	ai, err := NewAuthInterceptor(&fakeValidator{})
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(nil))
	_, err = ai.UnaryAuthInterceptor()(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/orders.OrdersService/GetOrderByID"}, func(context.Context, any) (any, error) {
		return nil, nil
	})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestUnaryAuthInterceptorRejectsBadBearerPrefix(t *testing.T) {
	ai, err := NewAuthInterceptor(&fakeValidator{})
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("Authorization", "Token abc"))
	_, err = ai.UnaryAuthInterceptor()(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/orders.OrdersService/GetOrderByID"}, func(context.Context, any) (any, error) {
		return nil, nil
	})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestUnaryAuthInterceptorRejectsInvalidToken(t *testing.T) {
	ai, err := NewAuthInterceptor(&fakeValidator{err: errors.New("bad token")})
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("Authorization", "Bearer abc"))
	_, err = ai.UnaryAuthInterceptor()(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/orders.OrdersService/GetOrderByID"}, func(context.Context, any) (any, error) {
		return nil, nil
	})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestUnaryAuthInterceptorAddsClaimsToContext(t *testing.T) {
	validator := &fakeValidator{claims: &internalauth.UserClaims{UserID: "user-1", Role: "seller"}}
	ai, err := NewAuthInterceptor(validator)
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("Authorization", "Bearer abc"))
	res, err := ai.UnaryAuthInterceptor()(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/orders.OrdersService/GetOrderByID"}, func(ctx context.Context, req any) (any, error) {
		assert.Equal(t, "user-1", ctx.Value("userID"))
		assert.Equal(t, "seller", ctx.Value("role"))
		return "ok", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", res)
	assert.Equal(t, "abc", validator.receivedToken)
}

func TestStreamAuthInterceptorRejectsMissingMetadata(t *testing.T) {
	ai, err := NewAuthInterceptor(&fakeValidator{})
	require.NoError(t, err)

	err = ai.StreamAuthInterceptor()(nil, authTestServerStream{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: "/orders.OrdersService/GetOrdersByCustomerID", IsServerStream: true}, func(any, grpc.ServerStream) error {
		return nil
	})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestStreamAuthInterceptorAddsClaimsToWrappedContext(t *testing.T) {
	validator := &fakeValidator{claims: &internalauth.UserClaims{UserID: "user-2", Role: "buyer"}}
	ai, err := NewAuthInterceptor(validator)
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("Authorization", "Bearer stream-token"))
	called := false
	err = ai.StreamAuthInterceptor()(nil, authTestServerStream{ctx: ctx}, &grpc.StreamServerInfo{FullMethod: "/orders.OrdersService/GetOrdersByCustomerID", IsServerStream: true}, func(_ any, ss grpc.ServerStream) error {
		called = true
		assert.Equal(t, "user-2", ss.Context().Value("userID"))
		assert.Equal(t, "buyer", ss.Context().Value("role"))
		return nil
	})
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "stream-token", validator.receivedToken)
}

func TestForwardMetadataInterceptorCopiesIncomingMetadataToOutgoingContext(t *testing.T) {
	ai, err := NewAuthInterceptor(&fakeValidator{})
	require.NoError(t, err)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer abc", "x-request-id", "req-1"))
	called := false
	err = ai.ForwardMetadataInterceptor()(ctx, "/orders.OrdersService/GetOrderByID", nil, nil, nil, func(ctx context.Context, method string, req any, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		called = true
		md, ok := metadata.FromOutgoingContext(ctx)
		require.True(t, ok)
		assert.Equal(t, []string{"Bearer abc"}, md.Get("authorization"))
		assert.Equal(t, []string{"req-1"}, md.Get("x-request-id"))
		return nil
	})
	require.NoError(t, err)
	assert.True(t, called)
}
