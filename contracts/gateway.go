package contracts

import (
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

type Gateway interface {
	Run(mux ...*runtime.ServeMux) error
}
