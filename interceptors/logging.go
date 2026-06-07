package interceptors

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type LoggingInterceptor struct {
	logger *zap.Logger
}

func NewLoggingInterceptor(logger *zap.Logger) *LoggingInterceptor {
	return &LoggingInterceptor{
		logger: logger,
	}
}

func (i *LoggingInterceptor) UnaryLoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		res, err := handler(ctx, req)

		i.logger.Info(
			"gRPC request",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", time.Since(start)),
			zap.String("status", status.Code(err).String()),
			zap.Error(err),
		)

		return res, err
	}
}

func (i *LoggingInterceptor) UnaryClientLoggingInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		start := time.Now()

		err := invoker(ctx, method, req, reply, cc, opts...)

		i.logger.Info(
			"gRPC response",
			zap.String("method", method),
			zap.Duration("duration", time.Since(start)),
			zap.String("status", status.Code(err).String()),
			zap.Error(err),
		)

		return err
	}
}

func (i *LoggingInterceptor) StreamLoggingInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()
		i.logger.Info(
			"gRPC stream started",
			zap.String("method", info.FullMethod),
			zap.Bool("is_client-stream", info.IsClientStream),
			zap.Bool("is_server_stream", info.IsServerStream),
		)

		err := handler(srv, ss)

		i.logger.Info(
			"gRPC stream",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", time.Since(start)),
			zap.String("statsu", status.Code(err).String()),
			zap.Bool("is_client_stream", info.IsClientStream),
			zap.Bool("is_server_stream", info.IsServerStream),
			zap.Error(err),
		)

		return err
	}
}
