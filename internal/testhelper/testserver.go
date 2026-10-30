package testhelper

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"platform-service/internal/app/port"
	"platform-service/internal/app/usecase/webhook"
	"platform-service/internal/bootstrap"
	"platform-service/internal/config"
	"platform-service/internal/infrastructure/crypto"
	"platform-service/internal/infrastructure/keystore"
)

// TestServer membungkus engine Gin lengkap (wiring sama dengan cmd/app) di atas
// DB testcontainers.
type TestServer struct {
	DB     *sql.DB
	Engine *gin.Engine
	Config *config.Config
	// Issuer menerbitkan JWT asli (ditandatangani signing key active di DB test).
	Issuer port.TokenIssuer
	// KeyStore in-memory yang dipakai server ini (dibagi dengan server turunan).
	KeyStore *keystore.MemoryKeyStore
	// Auth terisi bila server memakai FakeAuthenticator (default saat
	// WithAuthenticator tidak diberikan).
	Auth *FakeAuthenticator
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

// WithAuthenticator mengganti FakeAuthenticator default dengan authenticator lain
// (dicoba sebelum verifier JWT).
func WithAuthenticator(a port.TokenAuthenticator) Option {
	return func(o *serverOptions) { o.authenticator = a }
}

// MustStartTestServer menyalakan DB test + engine. Panggil sekali dari TestMain
// dan jalankan cleanup setelah m.Run().
func MustStartTestServer(opts ...Option) (*TestServer, func()) {
	db, cleanup := MustStartTestDB()
	srv, err := newTestServer(db, NewFakeAuthenticator(), keystore.NewMemoryKeyStore(), opts)
	if err != nil {
		cleanup()
		log.Fatalf("testhelper: %v", err)
	}
	return srv, cleanup
}

// MustDerive membangun engine baru di atas DB yang sama dengan opsi berbeda,
// mis. server mode on_prem untuk menguji LICENSE_MANAGED. FakeAuthenticator
// ikut dipakai bersama sehingga token dari server asal tetap berlaku.
func (s *TestServer) MustDerive(opts ...Option) *TestServer {
	srv, err := newTestServer(s.DB, s.Auth, s.KeyStore, opts)
	if err != nil {
		log.Fatalf("testhelper: %v", err)
	}
	return srv
}

func newTestServer(db *sql.DB, fake *FakeAuthenticator, ks *keystore.MemoryKeyStore, opts []Option) (*TestServer, error) {
	gin.SetMode(gin.TestMode)

	o := serverOptions{deploymentMode: config.DeploymentCloud}
	for _, opt := range opts {
		opt(&o)
	}

	authn := o.authenticator
	if authn == nil {
		authn = fake
	} else {
		fake = nil
	}

	cfg := TestConfig(o.deploymentMode)
	app, err := bootstrap.New(context.Background(), bootstrap.Options{
		Config:             cfg,
		DB:                 db,
		Logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		KeyStore:           ks,
		PasswordHasher:     TestPasswordHasher(),
		ExtraAuthenticator: authn,
	})
	if err != nil {
		return nil, fmt.Errorf("bootstrap: %w", err)
	}
	return &TestServer{DB: db, Engine: app.Engine, Config: cfg, Issuer: app.TokenIssuer, KeyStore: ks, Auth: fake}, nil
}

// TestPasswordHasher — bcrypt cost minimum agar test cepat.
func TestPasswordHasher() port.PasswordHasher {
	return crypto.NewBcryptHasher(bcrypt.MinCost)
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
		Auth: config.AuthConfig{
			AccessTokenTTL:      15 * time.Minute,
			RefreshTokenTTL:     720 * time.Hour,
			ConsoleSolutionCode: TestConsoleSolutionCode,
		},
		CORS: config.CORSConfig{AllowOrigins: []string{"*"}},
		Webhook: config.WebhookConfig{
			SecretKey:        TestWebhookSecretKey,
			AllowInsecureURL: true,
			PollInterval:     time.Second,
			BatchSize:        500,
			Timeout:          2 * time.Second,
		},
	}
}

// TestConsoleSolutionCode — solution console di test (identity provider SSO).
const TestConsoleSolutionCode = "platform-console-test"

// TestWebhookSecretKey — kunci AES-256 (base64) untuk mengenkripsi secret webhook di test.
const TestWebhookSecretKey = "dGVzdC1vbmx5LXdlYmhvb2stc2VjcmV0LWtleS0zMmI="

// NewDispatcher membangun dispatcher webhook di atas DB test (HTTP sender asli).
func (s *TestServer) NewDispatcher() *webhook.Dispatcher {
	d, err := bootstrap.NewDispatcher(s.Config, s.DB, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	if err != nil {
		log.Fatalf("testhelper: %v", err)
	}
	return d
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

// POSTForm mengirim body application/x-www-form-urlencoded (mis. OAuth2 token endpoint).
func (s *TestServer) POSTForm(path string, form url.Values, headers ...map[string]string) *httptest.ResponseRecorder {
	req, err := http.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	if err != nil {
		panic("testhelper: new request: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range mergeHeaders(headers) {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	s.Engine.ServeHTTP(w, req)
	return w
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
