package gateway

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/support/json"
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
	var body *bytes.Buffer
	fallback := FacadesConfig.Get("gateway.fallback").(func(ctx contractshttp.Context, err error) contractshttp.Response)
	url := fmt.Sprintf("http://%s:%s%s", FacadesConfig.GetString("gateway.host"), FacadesConfig.GetString("gateway.port"), ctx.Request().Path())

	if method == http.MethodGet {
		if rawQuery := ctx.Request().Origin().URL.RawQuery; rawQuery != "" {
			url += "?" + rawQuery
		}
	} else {
		var err error
		contentType := ctx.Request().Header("Content-Type")
		if strings.Contains(contentType, "application/json") {
			body, err = AddInjectForJson(ctx.Request().Origin())
			if err != nil {
				return fallback(ctx, err)
			}
		}
		if strings.Contains(contentType, "application/x-www-form-urlencoded") {
			body, err = AddInjectForForm(ctx.Request().Origin())
			if err != nil {
				return fallback(ctx, err)
			}
		}
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return fallback(ctx, err)
	}

	//query := ctx.Request().Origin().URL.Query()
	//req.URL.RawQuery = query.Encode()
	for key, header := range ctx.Request().Headers() {
		if key == "Content-Length" {
			req.Header.Set(key, cast.ToString(body.Len()))
		} else {
			req.Header.Set(key, header[0])
		}
	}
	//req.Header = ctx.Request().Origin().Header.Clone()
	//req.ContentLength = int64(body.Len())

	gatewayResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fallback(ctx, err)
	}
	defer gatewayResp.Body.Close()
	content, err := io.ReadAll(gatewayResp.Body)
	if err != nil {
		return fallback(ctx, err)
	}

	resp := ctx.Response()
	for key, value := range gatewayResp.Header {
		if len(value) > 0 {
			resp = resp.Header(key, value[0])
		}
	}

	return resp.Data(200, ctx.Request().Header("Content-Type", "application/json"), content)
}

func AddInjectForJson(request *http.Request) (*bytes.Buffer, error) {
	injectData := request.URL.Query()
	originBody := request.Body

	body, err := io.ReadAll(originBody)
	if err != nil {
		return nil, err
	}

	var reqBody map[string]any
	err = json.Unmarshal(body, &reqBody)
	if err != nil {
		return nil, err
	}

	for key, value := range injectData {
		if _, exist := reqBody[key]; !exist {
			reqBody[key] = value[0]
		}
	}

	newBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(newBody), nil
}

func AddInjectForForm(request *http.Request) (*bytes.Buffer, error) {
	injectData := request.URL.Query()
	if request.PostForm == nil {
		if err := request.ParseForm(); err != nil {
			return nil, fmt.Errorf("parse form error: %v", err)
		}
	}

	for key, value := range injectData {
		if !request.PostForm.Has(key) {
			request.PostForm.Set(key, value[0])
		}
	}

	return bytes.NewBufferString(request.PostForm.Encode()), nil
}
