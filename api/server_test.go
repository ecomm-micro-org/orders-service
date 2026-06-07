package api

import (
	"context"
	"testing"

	"github.com/risbern21/runaway/orders-service/interceptors"
	internalauth "github.com/risbern21/runaway/orders-service/internal/auth"
	"github.com/risbern21/runaway/orders-service/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type apiTestValidator struct{}

func (apiTestValidator) ValidateToken(context.Context, string) (*internalauth.UserClaims, error) {
	return &internalauth.UserClaims{UserID: "user-1", Role: "buyer"}, nil
}

func TestNewGRPCServerRegistersOrdersService(t *testing.T) {
	li := interceptors.NewLoggingInterceptor(zap.NewNop())
	ai, err := interceptors.NewAuthInterceptor(apiTestValidator{})
	require.NoError(t, err)

	srv := NewGRPCServer(&services.OrderService{}, li, ai)
	require.NotNil(t, srv)

	info := srv.GetServiceInfo()
	svc, ok := info["orders.OrdersService"]
	require.True(t, ok)
	assert.Len(t, svc.Methods, 9)
}
