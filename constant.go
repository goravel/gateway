package gateway

import (
	"context"

	"github.com/goravel/framework/contracts/http"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type Api struct {
	Method     string
	Url        string
	Middleware []http.Middleware
}

type Grpc struct {
	Name    string
	Handler func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error
}
