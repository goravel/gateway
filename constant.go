package gateway

import (
	"context"

	"github.com/goravel/framework/contracts/http"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

const InjectKey = "gateway-inject"

type NumberOrString interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64 | ~string
}

type Api struct {
	Method     string
	Url        string
	Middleware []http.Middleware
}

type Grpc struct {
	Name    string
	Handler func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error
}
