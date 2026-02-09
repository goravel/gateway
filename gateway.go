package gateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gookit/color"
	"github.com/goravel/framework/contracts/config"
	contractsgrpc "github.com/goravel/framework/contracts/grpc"
	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type Handler func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error

type Gateway struct {
	config config.Config
	grpc   contractsgrpc.Grpc
}

func NewGateway(config config.Config, grpc contractsgrpc.Grpc) *Gateway {
	return &Gateway{
		config: config,
		grpc:   grpc,
	}
}

func (r *Gateway) Run(serveMux ...*runtime.ServeMux) error {
	host := r.config.GetString("gateway.host")
	port := r.config.GetString("gateway.port")
	if host == "" || port == "" {
		return errors.New("please initialize GATEWAY_HOST and GATEWAY_PORT")
	}

	mux := runtime.NewServeMux()
	if len(serveMux) > 0 {
		mux = serveMux[0]
	}

	connections := make(map[string]*grpc.ClientConn)
	clients := r.config.Get("grpc.servers").(map[string]any)
	for name, params := range clients {
		if name == "" {
			return errors.New("gRPC client name is required")
		}

		if _, exist := connections[name]; !exist {
			connection, err := r.grpc.Client(context.Background(), name)
			if err != nil {
				return fmt.Errorf("init gRPC %s client failed: %v", name, err)
			}

			connections[name] = connection
		}

		handlers, exist := params.(map[string]any)["handlers"]
		if !exist {
			return fmt.Errorf("gRPC %s handlers is required", name)
		}

		for _, handler := range handlers.([]Handler) {
			if err := handler(context.Background(), mux, connections[name]); err != nil {
				return fmt.Errorf("register gRPC %s handler failed: %v", name, err)
			}
		}
	}

	addr := fmt.Sprintf("%s:%s", host, port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	color.Greenln("[Gateway] Listening and serving Gateway on " + addr)
	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("HTTP listen failed: %v", err)
	}

	return nil
}

func Inject[V NumberOrString](ctx contractshttp.Context, key string, value V) {
	if injectValue, exist := ctx.Value(InjectKey).(map[string]any); exist {
		injectValue[key] = value
	} else {
		ctx.WithValue(InjectKey, map[string]any{key: value})
	}
}
