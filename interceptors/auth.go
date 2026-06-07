package interceptors

import (
	"context"
	"fmt"
	"strings"

	"github.com/risbern21/runaway/orders-service/internal/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type (
	Validator interface {
		ValidateToken(context.Context, string) (*auth.UserClaims, error)
	}

	AuthInterceptor struct {
		validator Validator
	}

	serverStream struct {
		ServerStream grpc.ServerStream
		ctx          context.Context
	}
)

func NewAuthInterceptor(validator Validator) (*AuthInterceptor, error) {
	if validator == nil {
		return nil, fmt.Errorf("validator cannot be nil")
	}

	return &AuthInterceptor{
		validator: validator,
	}, nil
}

func (ai *AuthInterceptor) UnaryAuthInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "metadata not provided")
		}

		token := md.Get("Authorization")
		if len(token) == 0 {
			return nil, status.Error(codes.Unauthenticated, "no 'Authorization' token provided")
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(token[0], prefix) {
			return nil, status.Error(codes.Unauthenticated, "'Authorization' header must start with 'Bearer '")
		}
		tokenStr := strings.TrimPrefix(token[0], prefix)

		claims, err := ai.validator.ValidateToken(ctx, tokenStr)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization token")
		}

		ctx = contextWithClaims(ctx, claims)
		return handler(ctx, req)
	}
}

func (ai *AuthInterceptor) StreamAuthInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := ss.Context()

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return status.Error(codes.Unauthenticated, "metadata not provided")
		}

		token := md.Get("Authorization")
		if len(token) == 0 {
			return status.Error(codes.Unauthenticated, "'no 'Authorization' token provided")
		}

		const prefix = "Bearer "
		if !strings.HasPrefix(token[0], prefix) {
			return status.Error(codes.Unauthenticated, "Authorization' header must start with 'Bearer '")
		}
		tokenStr := strings.TrimPrefix(token[0], prefix)

		claims, err := ai.validator.ValidateToken(ctx, tokenStr)
		if err != nil {
			return status.Error(codes.Unauthenticated, "invalid authorization token")
		}

		ctx = contextWithClaims(ctx, claims)
		return handler(srv, &serverStream{
			ServerStream: ss,
			ctx:          ctx,
		})
	}
}

func (ai *AuthInterceptor) ForwardMetadataInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req any,
		reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			ctx = metadata.NewOutgoingContext(ctx, md)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func contextWithClaims(ctx context.Context, claims *auth.UserClaims) context.Context {
	ctx = context.WithValue(ctx, "userID", claims.UserID)
	ctx = context.WithValue(ctx, "role", claims.Role)
	return ctx
}

func (ss *serverStream) Context() context.Context {
	return ss.ctx
}

func (ss *serverStream) SetHeader(md metadata.MD) error {
	return ss.ServerStream.SetHeader(md)
}

func (ss *serverStream) SetTrailer(md metadata.MD) {
	ss.ServerStream.SetTrailer(md)
}

func (ss *serverStream) SendHeader(md metadata.MD) error {
	return ss.ServerStream.SendHeader(md)
}

func (ss *serverStream) RecvMsg(m any) error {
	return ss.ServerStream.RecvMsg(m)
}

func (ss *serverStream) SendMsg(m any) error {
	return ss.ServerStream.SendMsg(m)
}
