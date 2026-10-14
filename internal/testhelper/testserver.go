package testhelper

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"platform-service/internal/app/port"
	"platform-service/internal/bootstrap"
	"platform-service/internal/config"
)

// TestServer membungkus engine Gin lengkap (wiring sama dengan cmd/app) di atas
// DB testcontainers.
type TestServer struct {
	DB     *sql.DB
	Engine *gin.Engine
	Config *config.Config
}

type serverOptions struct {
	deploymentMode config.DeploymentMode
	authenticator  port.TokenAuthenticator
}

type Option func(*serverOptions)

// WithDeploymentMode menjalankan server dalam mode cloud / on_prem (default cloud).
func WithDeploymentMode(mode config.DeploymentMode) Option {
	return func(o *serverOptions) { o.deploymentMode = mode }
}

// WithAuthenticator mengganti verifier token (mis. fake untuk test middleware).
func WithAuthenticator(a port.TokenAuthenticator) Option {
	return func(o *serverOptions) { o.authenticator = a }
}

// MustStartTestServer menyalakan DB test + engine. Panggil sekali dari TestMain
// dan jalankan cleanup setelah m.Run().
func MustStartTestServer(opts ...Option) (*TestServer, func()) {
	gin.SetMode(gin.TestMode)

	o := serverOptions{deploymentMode: config.DeploymentCloud}
	for _, opt := range opts {
		opt(&o)
	}

	db, cleanup := MustStartTestDB()
	cfg := TestConfig(o.deploymentMode)

	engine, err := bootstrap.NewEngine(bootstrap.Options{
		Config:        cfg,
		DB:            db,
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		Authenticator: o.authenticator,
	})
	if err != nil {
		cleanup()
		log.Fatalf("testhelper: bootstrap engine: %v", err)
	}

	return &TestServer{DB: db, Engine: engine, Config: cfg}, cleanup
}

// TestConfig mengembalikan config untuk test. Database diabaikan karena DB
// disuplai langsung dari testcontainers.
func TestConfig(mode config.DeploymentMode) *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Env:            "test",
			LocalesDir:     LocalesDir(),
			DeploymentMode: mode,
			IssuerURL:      "http://platform.test",
		},
		CORS: config.CORSConfig{AllowOrigins: []string{"*"}},
	}
}

// LocalesDir mengembalikan path absolut <repo>/locales. `go test` menjalankan
// test dari direktori package, jadi path relatif tidak bisa dipakai.
func LocalesDir() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("testhelper: tidak bisa menentukan lokasi file")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "locales")
}

func (s *TestServer) do(method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			panic("testhelper: marshal request body: " + err.Error())
		}
		bodyBytes = b
	}

	req, err := http.NewRequest(method, path, bytes.NewReader(bodyBytes))
	if err != nil {
		panic("testhelper: new request: " + err.Error())
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	s.Engine.ServeHTTP(w, req)
	return w
}

func (s *TestServer) GET(path string, headers ...map[string]string) *httptest.ResponseRecorder {
	return s.do(http.MethodGet, path, nil, mergeHeaders(headers))
}

func (s *TestServer) POST(path string, body any, headers ...map[string]string) *httptest.ResponseRecorder {
	return s.do(http.MethodPost, path, body, mergeHeaders(headers))
}

func (s *TestServer) PUT(path string, body any, headers ...map[string]string) *httptest.ResponseRecorder {
	return s.do(http.MethodPut, path, body, mergeHeaders(headers))
}

func (s *TestServer) PATCH(path string, body any, headers ...map[string]string) *httptest.ResponseRecorder {
	return s.do(http.MethodPatch, path, body, mergeHeaders(headers))
}

func (s *TestServer) DELETE(path string, headers ...map[string]string) *httptest.ResponseRecorder {
	return s.do(http.MethodDelete, path, nil, mergeHeaders(headers))
}

// DELETEBody untuk endpoint DELETE yang membutuhkan request body.
func (s *TestServer) DELETEBody(path string, body any, headers ...map[string]string) *httptest.ResponseRecorder {
	return s.do(http.MethodDelete, path, body, mergeHeaders(headers))
}

func BearerHeader(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

// DecodeBody men-decode field "data" dari envelope sukses ke out.
func DecodeBody(t *testing.T, w *httptest.ResponseRecorder, out any) {
	t.Helper()
	var wrapper struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&wrapper), "decode response body")
	if out != nil {
		require.NoError(t, json.Unmarshal(wrapper.Data, out), "decode data field")
	}
}

// ErrorBody adalah envelope error standar.
type ErrorBody struct {
	Status     string          `json:"status"`
	StatusCode int             `json:"status_code"`
	ErrorCode  string          `json:"error_code"`
	Errors     json.RawMessage `json:"errors"`
}

// DecodeError men-decode envelope error.
func DecodeError(t *testing.T, w *httptest.ResponseRecorder) ErrorBody {
	t.Helper()
	var body ErrorBody
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body), "decode error body: %s", w.Body.String())
	return body
}

func mergeHeaders(headers []map[string]string) map[string]string {
	out := map[string]string{}
	for _, h := range headers {
		for k, v := range h {
			out[k] = v
		}
	}
	return out
}
