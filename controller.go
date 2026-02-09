package gateway

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/foundation/json"
	"github.com/spf13/cast"
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
	// Inject Value into Query
	if injectValue, exist := ctx.Value(InjectKey).(map[string]any); exist {
		query := ctx.Request().Origin().URL.Query()
		for key, value := range injectValue {
			query.Add(key, cast.ToString(value))
		}
		ctx.Request().Origin().URL.RawQuery = query.Encode()
	}

	fallback := FacadesConfig.Get("gateway.fallback").(func(ctx contractshttp.Context, err error) contractshttp.Response)

	var body io.Reader
	if method != http.MethodGet && method != http.MethodDelete {
		// Put Query into Body, because Gateway only accept Body
		data, err := io.ReadAll(ctx.Request().Origin().Body)
		if err != nil {
			return fallback(ctx, err)
		}

		jsonDriver := json.New()
		var dataJson map[string]any
		if err := jsonDriver.Unmarshal(data, &dataJson); err != nil {
			return fallback(ctx, err)
		}

		for key, value := range ctx.Request().Queries() {
			dataJson[key] = value
		}

		newData, err := jsonDriver.Marshal(dataJson)
		if err != nil {
			return fallback(ctx, err)
		}

		body = strings.NewReader(string(newData))
	}

	url := fmt.Sprintf("http://%s:%s%s", FacadesConfig.GetString("gateway.host"), FacadesConfig.GetString("gateway.port"), ctx.Request().Path())
	gatewayReq, err := http.NewRequest(method, url, body)
	if err != nil {
		return fallback(ctx, err)
	}

	query := ctx.Request().Origin().URL.Query()
	gatewayReq.URL.RawQuery = query.Encode()
	for key, header := range ctx.Request().Headers() {
		gatewayReq.Header.Set(key, header[0])
	}

	gatewayResp, err := http.DefaultClient.Do(gatewayReq)
	if err != nil {
		return fallback(ctx, err)
	}
	defer func() {
		_ = gatewayResp.Body.Close()
	}()
	data, err := io.ReadAll(gatewayResp.Body)
	if err != nil {
		return fallback(ctx, err)
	}

	resp := ctx.Response()
	for key, value := range gatewayResp.Header {
		if len(value) > 0 && key != "Content-Length" {
			resp = resp.Header(key, value[0])
		}
	}

	return resp.Data(200, ctx.Request().Header("Content-Type", "application/json"), data)
}
