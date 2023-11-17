package config

import (
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("gateway", map[string]any{
		// The Gateway host and port, the HTTP request wil be sent to this host.
		"host": config.Env("GATEWAY_HOST", ""),
		"port": config.Env("GATEWAY_PORT", ""),
		// The fallback function will be called when the request is failed, you can optimize it to your response structure.
		"fallback": func(ctx http.Context, err error) http.Response {
			return ctx.Response().Success().Json(map[string]any{
				"status": map[string]any{
					"code":  http.StatusInternalServerError,
					"error": err.Error(),
				},
			})
		},
	})
}
