package gateway

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	configmocks "github.com/goravel/framework/mocks/config"
	grpcmocks "github.com/goravel/framework/mocks/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestRun(t *testing.T) {
	var (
		mockConfig *configmocks.Config
		mockGrpc   *grpcmocks.Grpc
		gateway    *Gateway
	)

	beforeEach := func() {
		mockConfig = new(configmocks.Config)
		mockGrpc = new(grpcmocks.Grpc)
		gateway = NewGateway(mockConfig, mockGrpc)
	}

	tests := []struct {
		name      string
		setup     func()
		expectErr error
	}{
		{
			name: "Happy path",
			setup: func() {
				mockConfig.On("GetString", "gateway.host").Return("127.0.0.1")
				mockConfig.On("GetString", "gateway.port").Return("4001")
				mockConfig.On("Get", "grpc.clients").Return(map[string]any{
					"goravel": map[string]any{
						"handlers": []Handler{
							func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
								return nil
							},
						},
					},
				})
				mockGrpc.On("Client", context.Background(), "goravel").Return(&grpc.ClientConn{}, nil)
			},
		},
		{
			name: "Happy path when gateway.grpc is empty",
			setup: func() {
				mockConfig.On("GetString", "gateway.host").Return("127.0.0.1")
				mockConfig.On("GetString", "gateway.port").Return("4002")
				mockConfig.On("Get", "grpc.clients").Return(map[string]any{})
			},
		},
		{
			name: "error, gateway.host is empty",
			setup: func() {
				mockConfig.On("GetString", "gateway.host").Return("")
				mockConfig.On("GetString", "gateway.port").Return("4001")
			},
			expectErr: errors.New("please initialize GATEWAY_HOST and GATEWAY_PORT"),
		},
		{
			name: "error, gateway.port is empty",
			setup: func() {
				mockConfig.On("GetString", "gateway.host").Return("127.0.0.1")
				mockConfig.On("GetString", "gateway.port").Return("")
			},
			expectErr: errors.New("please initialize GATEWAY_HOST and GATEWAY_PORT"),
		},
		{
			name: "error, grpc handler is nil",
			setup: func() {
				mockConfig.On("GetString", "gateway.host").Return("127.0.0.1")
				mockConfig.On("GetString", "gateway.port").Return("4001")
				mockConfig.On("Get", "grpc.clients").Return(map[string]any{
					"goravel": map[string]any{},
				})
				mockGrpc.On("Client", context.Background(), "goravel").Return(&grpc.ClientConn{}, nil)
			},
			expectErr: fmt.Errorf("gRPC %s handlers is required", "goravel"),
		},
		{
			name: "error, grpc handler returns error",
			setup: func() {
				mockConfig.On("GetString", "gateway.host").Return("127.0.0.1")
				mockConfig.On("GetString", "gateway.port").Return("4001")
				mockConfig.On("Get", "grpc.clients").Return(map[string]any{
					"goravel": map[string]any{
						"handlers": []Handler{
							func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
								return errors.New("error")
							},
						},
					},
				})
				mockGrpc.On("Client", context.Background(), "goravel").Return(&grpc.ClientConn{}, nil)
			},
			expectErr: fmt.Errorf("register gRPC %s handler failed: %v", "goravel", errors.New("error")),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			beforeEach()
			test.setup()
			var err error
			go func() {
				err = gateway.Run()
			}()
			time.Sleep(1 * time.Second)
			if test.expectErr == nil {
				assert.Nil(t, err)
			} else {
				assert.EqualError(t, err, test.expectErr.Error())
			}

			mockConfig.AssertExpectations(t)
			mockGrpc.AssertExpectations(t)
		})
	}
}
