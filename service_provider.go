package gateway

import (
	"github.com/goravel/framework/contracts/config"
	"github.com/goravel/framework/contracts/foundation"
)

const Binding = "gateway"

var (
	App           foundation.Application
	FacadesConfig config.Config
)

type ServiceProvider struct {
}

func (receiver *ServiceProvider) Register(app foundation.Application) {
	App = app
	FacadesConfig = app.MakeConfig()

	app.Bind(Binding, func(app foundation.Application) (any, error) {
		return NewGateway(app.MakeConfig(), app.MakeGrpc()), nil
	})
}

func (receiver *ServiceProvider) Boot(app foundation.Application) {
	app.Publishes("github.com/goravel/gateway", map[string]string{
		"config/gateway.go": app.ConfigPath("gateway.go"),
	})
	app.Publishes("github.com/goravel/gateway", map[string]string{
		"proto": app.BasePath("proto"),
	})
}
