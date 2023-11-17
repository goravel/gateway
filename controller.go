package gateway

import (
	"fmt"
	"io"
	"net/http"

	contractshttp "github.com/goravel/framework/contracts/http"
)

func Get(ctx contractshttp.Context) contractshttp.Response {
	return request(ctx, http.MethodGet)
}

func Post(ctx contractshttp.Context) contractshttp.Response {
	return request(ctx, http.MethodPost)
}

func Put(ctx contractshttp.Context) contractshttp.Response {
	return request(ctx, http.MethodPut)
}

func Delete(ctx contractshttp.Context) contractshttp.Response {
	return request(ctx, http.MethodDelete)
}

func Patch(ctx contractshttp.Context) contractshttp.Response {
	return request(ctx, http.MethodPatch)
}

func request(ctx contractshttp.Context, method string) contractshttp.Response {
	var body io.Reader
	if method != http.MethodGet {
		body = ctx.Request().Origin().Body
	}

	fallback := FacadesConfig.Get("gateway.fallback").(func(ctx contractshttp.Context, err error) contractshttp.Response)
	url := fmt.Sprintf("http://%s:%s%s", FacadesConfig.GetString("gateway.host"), FacadesConfig.GetString("gateway.port"), ctx.Request().Path())
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return fallback(ctx, err)
	}

	req.Header.Set("Content-Type", ctx.Request().Header("Content-Type"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fallback(ctx, err)
	}
	defer resp.Body.Close()
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fallback(ctx, err)
	}

	return ctx.Response().Data(200, "application/json", content)
}
