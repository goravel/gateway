package gateway

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/goravel/framework/contracts/filesystem"
	contractshttp "github.com/goravel/framework/contracts/http"
	contractsession "github.com/goravel/framework/contracts/session"
	"github.com/goravel/framework/contracts/validation"
	frameworkgrpc "github.com/goravel/framework/grpc"
	mocksconfig "github.com/goravel/framework/mocks/config"
	testingmock "github.com/goravel/framework/testing/mock"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/goravel/gateway/proto/example"
)

var (
	mockFactory = testingmock.Factory()
	exampleHost = "127.0.0.1"
	examplePort = "3000"
	gatewayHost = "127.0.0.1"
	gatewayPort = "3001"
	httpPort    = "3002"
)

type ControllerTestSuite struct {
	suite.Suite
	grpc    *frameworkgrpc.Application
	gateway *Gateway
}

func TestControllerTestSuite(t *testing.T) {
	suite.Run(t, &ControllerTestSuite{})
}

func (s *ControllerTestSuite) SetupSuite() {
	mockConfig := mockFactory.Config()
	mockConfig.EXPECT().GetString("grpc.host").Return(exampleHost).Once()
	mockConfig.EXPECT().GetString("grpc.port").Return(examplePort).Once()
	mockConfig.EXPECT().GetString("gateway.host").Return(gatewayHost).Once()
	mockConfig.EXPECT().GetString("gateway.port").Return(gatewayPort).Once()
	mockConfig.EXPECT().Get("grpc.clients").Return(map[string]any{
		"example": map[string]any{
			"host":         exampleHost,
			"port":         examplePort,
			"handlers":     []Handler{example.RegisterUserServiceHandler},
			"interceptors": []string{},
		},
	}).Once()
	mockConfig.EXPECT().GetString("grpc.clients.example.host").Return(exampleHost).Once()
	mockConfig.EXPECT().GetString("grpc.clients.example.port").Return(examplePort).Once()
	mockConfig.EXPECT().Get("grpc.clients.example.interceptors").Return([]string{}).Once()

	s.grpc = frameworkgrpc.NewApplication(mockConfig)
	s.grpc.UnaryServerInterceptors([]grpc.UnaryServerInterceptor{})
	s.grpc.UnaryClientInterceptorGroups(map[string][]grpc.UnaryClientInterceptor{})
	example.RegisterUserServiceServer(s.grpc.Server(), NewUserController())

	go func() {
		if err := s.grpc.Run(); err != nil {
			panic(err)
		}
	}()

	s.gateway = NewGateway(mockConfig, s.grpc)

	go func() {
		mux := runtime.NewServeMux(
			runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
				MarshalOptions: protojson.MarshalOptions{
					UseProtoNames: true,
				},
				UnmarshalOptions: protojson.UnmarshalOptions{
					DiscardUnknown: true,
				},
			}),
		)
		if err := s.gateway.Run(mux); err != nil {
			panic(err)
		}
	}()

	http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		var resp contractshttp.Response

		switch r.Method {
		case "GET":
			resp = Get(NewTestContext(context.Background(), w, r))
		case "POST":
			resp = Post(NewTestContext(context.Background(), w, r))
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}

		if err := resp.Render(); err != nil {
			panic(err)
		}
	})

	http.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		var resp contractshttp.Response

		switch r.Method {
		case "GET":
			resp = Get(NewTestContext(context.Background(), w, r))
		case "PUT":
			resp = Put(NewTestContext(context.Background(), w, r))
		case "DELETE":
			resp = Delete(NewTestContext(context.Background(), w, r))
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}

		if err := resp.Render(); err != nil {
			panic(err)
		}
	})

	go func() {
		if err := http.ListenAndServe(":"+httpPort, nil); err != nil {
			panic(err)
		}
	}()

	time.Sleep(1 * time.Second)

	mockConfig.AssertExpectations(s.T())
}

func (s *ControllerTestSuite) TestGet() {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "Happy path - param",
			path: fmt.Sprintf("http://127.0.0.1:%s/users/1", httpPort),
		},
		{
			name: "Happy path - query",
			path: fmt.Sprintf("http://127.0.0.1:%s/users?name=goravel&age=18", httpPort),
		},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			mockConfig := mockConfig()

			req, err := http.NewRequest(http.MethodGet, test.path, nil)
			s.Require().NoError(err)

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Grpc-Metadata-Name", "goravel")

			resp, err := http.DefaultClient.Do(req)
			s.Require().NoError(err)
			defer func() {
				_ = resp.Body.Close()
			}()

			s.Equal("goravel", resp.Header.Get("Grpc-Metadata-Custom-Header"))

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				s.Require().NoError(err)
			}

			s.Equal(`{"status":{"code":200},"user":{"id":1,"user_id":2,"name":"goravel","age":18}}`, strings.ReplaceAll(string(body), " ", ""))

			mockConfig.AssertExpectations(s.T())
		})
	}
}

func (s *ControllerTestSuite) TestPost() {
	mockConfig := mockConfig()

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%s/users", httpPort), strings.NewReader(`{
		"name": "goravel",
		"age": 18
	}`))
	s.Require().NoError(err)

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.Require().NoError(err)
	}

	s.Equal(`{"status":{"code":200},"user":{"id":1,"user_id":2,"name":"goravel","age":18}}`, strings.ReplaceAll(string(body), " ", ""))

	mockConfig.AssertExpectations(s.T())
}

func (s *ControllerTestSuite) TestPut() {
	mockConfig := mockConfig()

	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("http://127.0.0.1:%s/users/1?age=18", httpPort), strings.NewReader(`{
		"name": "goravel"
	}`))
	s.Require().NoError(err)

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.Require().NoError(err)
	}

	s.Equal(`{"status":{"code":200},"user":{"id":1,"user_id":2,"name":"goravel","age":18}}`, strings.ReplaceAll(string(body), " ", ""))

	mockConfig.AssertExpectations(s.T())
}

func (s *ControllerTestSuite) TestDelete() {
	mockConfig := mockConfig()

	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://127.0.0.1:%s/users/1?age=18", httpPort), nil)
	s.Require().NoError(err)

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.Require().NoError(err)
	}

	s.Equal(`{"status":{"code":200}}`, strings.ReplaceAll(string(body), " ", ""))

	mockConfig.AssertExpectations(s.T())
}

func mockConfig() *mocksconfig.Config {
	mockConfig := mockFactory.Config()
	mockConfig.EXPECT().Get("gateway.fallback").Return(func(ctx contractshttp.Context, err error) contractshttp.Response {
		return ctx.Response().Success().String("fallback")
	}).Once()
	mockConfig.EXPECT().GetString("gateway.host").Return(gatewayHost).Once()
	mockConfig.EXPECT().GetString("gateway.port").Return(gatewayPort).Once()
	FacadesConfig = mockConfig

	return mockConfig
}

type UserController struct {
	example.UnimplementedUserServiceServer
}

func NewUserController() *UserController {
	return &UserController{}
}

func (r *UserController) GetUser(ctx context.Context, req *example.GetUserRequest) (*example.GetUserResponse, error) {
	var name string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if n, ok := md["name"]; ok {
			name = n[0]
		}
	}

	if err := grpc.SendHeader(ctx, metadata.New(map[string]string{
		"custom-header": "goravel",
	})); err != nil {
		return nil, err
	}

	return &example.GetUserResponse{
		Status: &example.Status{
			Code: 200,
		},
		User: &example.User{
			Id:     req.GetId(),
			UserId: req.GetUserId(),
			Name:   name,
			Age:    18,
		},
	}, nil
}

func (r *UserController) GetUsers(ctx context.Context, req *example.GetUsersRequest) (*example.GetUsersResponse, error) {
	if err := grpc.SendHeader(ctx, metadata.New(map[string]string{
		"custom-header": "goravel",
	})); err != nil {
		return nil, err
	}

	return &example.GetUsersResponse{
		Status: &example.Status{
			Code: 200,
		},
		User: &example.User{
			Id:     1,
			UserId: req.GetUserId(),
			Name:   req.GetName(),
			Age:    req.GetAge(),
		},
	}, nil
}

func (r *UserController) CreateUser(ctx context.Context, req *example.CreateUserRequest) (*example.CreateUserResponse, error) {
	return &example.CreateUserResponse{
		Status: &example.Status{
			Code: 200,
		},
		User: &example.User{
			Id:     1,
			UserId: req.GetUserId(),
			Name:   req.GetName(),
			Age:    req.GetAge(),
		},
	}, nil
}

func (r *UserController) UpdateUser(ctx context.Context, req *example.UpdateUserRequest) (*example.UpdateUserResponse, error) {
	return &example.UpdateUserResponse{
		Status: &example.Status{
			Code: 200,
		},
		User: &example.User{
			Id:     1,
			UserId: req.GetUserId(),
			Name:   req.GetName(),
			Age:    req.GetAge(),
		},
	}, nil
}

func (r *UserController) DeleteUser(ctx context.Context, req *example.DeleteUserRequest) (*example.DeleteUserResponse, error) {
	return &example.DeleteUserResponse{
		Status: &example.Status{
			Code: 200,
		},
	}, nil
}

type DataResponse struct {
	code        int
	contentType string
	data        []byte
	writer      http.ResponseWriter
}

func (r *DataResponse) Abort() error {
	return nil
}

func (r *DataResponse) Render() error {
	r.writer.WriteHeader(r.code)
	r.writer.Header().Set("Content-Type", r.contentType)
	if _, err := r.writer.Write(r.data); err != nil {
		return err
	}

	return nil
}

type TestContext struct {
	ctx     context.Context
	request *http.Request
	writer  http.ResponseWriter
}

func NewTestContext(ctx context.Context, w http.ResponseWriter, r *http.Request) *TestContext {
	testContext := &TestContext{
		ctx:     ctx,
		request: r,
		writer:  w,
	}
	Inject(testContext, "user_id", 2)

	return testContext
}

func (r *TestContext) Deadline() (deadline time.Time, ok bool) {
	panic("do not need to implement it")
}

func (r *TestContext) Done() <-chan struct{} {
	panic("do not need to implement it")
}

func (r *TestContext) Err() error {
	panic("do not need to implement it")
}

func (r *TestContext) Value(key any) any {
	return r.ctx.Value(key)
}

func (r *TestContext) Context() context.Context {
	return r.ctx
}

func (r *TestContext) WithContext(ctx context.Context) {
	r.ctx = ctx
}

func (r *TestContext) WithValue(key any, value any) {
	//nolint:all
	r.ctx = context.WithValue(r.ctx, key, value)
}

func (r *TestContext) Request() contractshttp.ContextRequest {
	return NewTestRequest(r)
}

func (r *TestContext) Response() contractshttp.ContextResponse {
	return NewTestResponse(r)
}

type TestRequest struct {
	ctx *TestContext
}

func NewTestRequest(ctx *TestContext) *TestRequest {
	return &TestRequest{
		ctx: ctx,
	}
}

func (r *TestRequest) Abort(code ...int) {
	panic("do not need to implement it")
}

func (r *TestRequest) Header(key string, def ...string) string {
	header := r.ctx.request.Header.Get(key)
	if header != "" {
		return header
	}

	if len(def) == 0 {
		return ""
	}

	return def[0]
}

func (r *TestRequest) Headers() http.Header {
	return r.ctx.request.Header
}

func (r *TestRequest) Method() string {
	panic("do not need to implement it")
}

func (r *TestRequest) Path() string {
	return r.ctx.request.URL.Path
}

func (r *TestRequest) Url() string {
	panic("do not need to implement it")
}

func (r *TestRequest) FullUrl() string {
	panic("do not need to implement it")
}

func (r *TestRequest) Ip() string {
	return "127.0.0.1"
}

func (r *TestRequest) Host() string {
	panic("do not need to implement it")
}

func (r *TestRequest) All() map[string]any {
	panic("do not need to implement it")
}

func (r *TestRequest) Cookie(key string, defaultValue ...string) string {
	return ""
}

func (r *TestRequest) Bind(any) error {
	panic("do not need to implement it")
}

func (r *TestRequest) BindQuery(any) error {
	panic("do not need to implement it")
}

func (r *TestRequest) Route(string) string {
	panic("do not need to implement it")
}

func (r *TestRequest) RouteInt(string) int {
	panic("do not need to implement it")
}

func (r *TestRequest) RouteInt64(string) int64 {
	panic("do not need to implement it")
}

func (r *TestRequest) Query(string, ...string) string {
	panic("do not need to implement it")
}

func (r *TestRequest) QueryInt(string, ...int) int {
	panic("do not need to implement it")
}

func (r *TestRequest) QueryInt64(string, ...int64) int64 {
	panic("do not need to implement it")
}

func (r *TestRequest) QueryBool(string, ...bool) bool {
	panic("do not need to implement it")
}

func (r *TestRequest) QueryArray(string) []string {
	panic("do not need to implement it")
}

func (r *TestRequest) QueryMap(string) map[string]string {
	panic("do not need to implement it")
}

func (r *TestRequest) Queries() map[string]string {
	queries := make(map[string]string)

	for key, query := range r.ctx.request.URL.Query() {
		queries[key] = strings.Join(query, ",")
	}

	return queries
}

func (r *TestRequest) HasSession() bool {
	_, ok := r.ctx.Value("session").(contractsession.Session)
	return ok
}

func (r *TestRequest) Session() contractsession.Session {
	s, ok := r.ctx.Value("session").(contractsession.Session)
	if !ok {
		return nil
	}
	return s
}

func (r *TestRequest) SetSession(session contractsession.Session) contractshttp.ContextRequest {
	r.ctx.WithValue("session", session)
	r.ctx.request = r.ctx.request.WithContext(r.ctx.Context())
	return r
}

func (r *TestRequest) Input(string, ...string) string {
	panic("do not need to implement it")
}

func (r *TestRequest) InputArray(string, ...[]string) []string {
	panic("do not need to implement it")
}

func (r *TestRequest) InputMap(string, ...map[string]string) map[string]string {
	panic("do not need to implement it")
}

func (r *TestRequest) InputInt(string, ...int) int {
	panic("do not need to implement it")
}

func (r *TestRequest) InputInt64(string, ...int64) int64 {
	panic("do not need to implement it")
}

func (r *TestRequest) InputBool(string, ...bool) bool {
	panic("do not need to implement it")
}

func (r *TestRequest) File(string) (filesystem.File, error) {
	panic("do not need to implement it")
}

func (r *TestRequest) AbortWithStatus(int) {}

func (r *TestRequest) AbortWithStatusJson(int, any) {
	panic("do not need to implement it")
}

func (r *TestRequest) Next() {
	panic("do not need to implement it")
}

func (r *TestRequest) Origin() *http.Request {
	return r.ctx.request
}

func (r *TestRequest) Validate(map[string]string, ...validation.Option) (validation.Validator, error) {
	panic("do not need to implement it")
}

func (r *TestRequest) ValidateRequest(contractshttp.FormRequest) (validation.Errors, error) {
	panic("do not need to implement it")
}

type TestResponse struct {
	ctx *TestContext
}

func NewTestResponse(ctx *TestContext) *TestResponse {
	return &TestResponse{
		ctx: ctx,
	}
}

func (r *TestResponse) Cookie(cookie contractshttp.Cookie) contractshttp.ContextResponse {
	return r
}

func (r *TestResponse) Data(code int, contentType string, data []byte) contractshttp.AbortableResponse {
	return &DataResponse{code, contentType, data, r.ctx.writer}
}

func (r *TestResponse) Download(string, string) contractshttp.Response {
	panic("do not need to implement it")
}

func (r *TestResponse) File(string) contractshttp.Response {
	panic("do not need to implement it")
}

func (r *TestResponse) Header(key, value string) contractshttp.ContextResponse {
	r.ctx.writer.Header().Set(key, value)

	return r
}

func (r *TestResponse) Json(int, any) contractshttp.AbortableResponse {
	panic("do not need to implement it")
}

func (r *TestResponse) NoContent(...int) contractshttp.AbortableResponse {
	panic("do not need to implement it")
}

func (r *TestResponse) Origin() contractshttp.ResponseOrigin {
	panic("do not need to implement it")
}

func (r *TestResponse) Redirect(int, string) contractshttp.AbortableResponse {
	panic("do not need to implement it")
}

func (r *TestResponse) String(int, string, ...any) contractshttp.AbortableResponse {
	panic("do not need to implement it")
}

func (r *TestResponse) Success() contractshttp.ResponseStatus {
	panic("do not need to implement it")
}

func (r *TestResponse) Status(int) contractshttp.ResponseStatus {
	panic("do not need to implement it")
}

func (r *TestResponse) Stream(int, func(contractshttp.StreamWriter) error) contractshttp.Response {
	panic("do not need to implement it")
}

func (r *TestResponse) WithoutCookie(string) contractshttp.ContextResponse {
	panic("do not need to implement it")
}

func (r *TestResponse) Writer() http.ResponseWriter {
	panic("do not need to implement it")
}

func (r *TestResponse) Flush() {
	panic("do not need to implement it")
}

func (r *TestResponse) View() contractshttp.ResponseView {
	panic("do not need to implement it")
}
