package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// DeploymentMode menentukan cara platform dipasang (spec §2.14).
type DeploymentMode string

const (
	DeploymentCloud  DeploymentMode = "cloud"
	DeploymentOnPrem DeploymentMode = "on_prem"
)

func (m DeploymentMode) Valid() bool {
	return m == DeploymentCloud || m == DeploymentOnPrem
}

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Auth     AuthConfig
	CORS     CORSConfig
	Webhook  WebhookConfig
}

// WebhookConfig mengatur enkripsi secret & worker pengiriman webhook.
type WebhookConfig struct {
	// SecretKey — kunci AES-256 (base64, 32 byte) untuk mengenkripsi secret webhook.
	SecretKey string
	// AllowInsecureURL — izinkan URL http (hanya dev/test; production wajib https).
	AllowInsecureURL bool
	PollInterval     time.Duration
	BatchSize        int
	Timeout          time.Duration
}

// devWebhookSecretKey dipakai di luar production bila WEBHOOK_SECRET_KEY kosong.
// JANGAN dipakai di production (Validate menolaknya).
const devWebhookSecretKey = "ZGV2LW9ubHktd2ViaG9vay1zZWNyZXQta2V5LTMyYiE="

type AppConfig struct {
	Env             string
	Port            string
	LogFormat       string
	LocalesDir      string
	DeploymentMode  DeploymentMode
	IssuerURL       string
	ShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DSN mengembalikan URL koneksi yang dipakai bersama oleh pgx dan golang-migrate.
func (d DatabaseConfig) DSN() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(d.User, d.Password),
		Host:     d.Host + ":" + d.Port,
		Path:     "/" + d.Name,
		RawQuery: "sslmode=" + d.SSLMode,
	}
	return u.String()
}

type AuthConfig struct {
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	// KeyStoreDir — direktori private key JWT untuk FileKeyStore.
	KeyStoreDir string
}

type CORSConfig struct {
	AllowOrigins []string
}

// Load membaca konfigurasi dari environment (dan file .env bila ada).
func Load() (*Config, error) {
	_ = godotenv.Load(".env")

	cfg := &Config{
		App: AppConfig{
			Env:            getEnv("APP_ENV", "development"),
			Port:           getEnv("APP_PORT", "8080"),
			LogFormat:      getEnv("LOG_FORMAT", ""),
			LocalesDir:     getEnv("LOCALES_DIR", "locales"),
			DeploymentMode: DeploymentMode(getEnv("DEPLOYMENT_MODE", string(DeploymentCloud))),
			IssuerURL:      getEnv("ISSUER_URL", "http://localhost:8080"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "platform"),
			Password: getEnv("DB_PASS", "secret"),
			Name:     getEnv("DB_NAME", "platform"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Auth: AuthConfig{
			KeyStoreDir: getEnv("KEYSTORE_DIR", "data/keys"),
		},
		CORS: CORSConfig{
			AllowOrigins: splitCSV(getEnv("CORS_ALLOW_ORIGINS", "*")),
		},
	}
	cfg.Webhook.SecretKey = getEnv("WEBHOOK_SECRET_KEY", "")
	if cfg.Webhook.SecretKey == "" && cfg.App.Env != "production" {
		cfg.Webhook.SecretKey = devWebhookSecretKey
	}
	cfg.Webhook.AllowInsecureURL = cfg.App.Env != "production"

	var err error
	if cfg.App.ShutdownTimeout, err = parseDuration("APP_SHUTDOWN_TIMEOUT", "15s"); err != nil {
		return nil, err
	}
	if cfg.Database.MaxOpenConns, err = parseInt("DB_MAX_OPEN_CONNS", 25); err != nil {
		return nil, err
	}
	if cfg.Database.MaxIdleConns, err = parseInt("DB_MAX_IDLE_CONNS", 5); err != nil {
		return nil, err
	}
	if cfg.Database.ConnMaxLifetime, err = parseDuration("DB_CONN_MAX_LIFETIME", "5m"); err != nil {
		return nil, err
	}
	if cfg.Auth.AccessTokenTTL, err = parseDuration("ACCESS_TOKEN_TTL", "15m"); err != nil {
		return nil, err
	}
	if cfg.Auth.RefreshTokenTTL, err = parseDuration("REFRESH_TOKEN_TTL", "720h"); err != nil {
		return nil, err
	}

	if cfg.Webhook.PollInterval, err = parseDuration("WEBHOOK_POLL_INTERVAL", "2s"); err != nil {
		return nil, err
	}
	if cfg.Webhook.Timeout, err = parseDuration("WEBHOOK_TIMEOUT", "10s"); err != nil {
		return nil, err
	}
	if cfg.Webhook.BatchSize, err = parseInt("WEBHOOK_BATCH_SIZE", 50); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if !c.App.DeploymentMode.Valid() {
		return fmt.Errorf("config: DEPLOYMENT_MODE harus %q atau %q, didapat %q",
			DeploymentCloud, DeploymentOnPrem, c.App.DeploymentMode)
	}
	if c.App.IssuerURL == "" {
		return fmt.Errorf("config: ISSUER_URL wajib diisi")
	}
	if c.App.Env == "production" && (c.Webhook.SecretKey == "" || c.Webhook.SecretKey == devWebhookSecretKey) {
		return fmt.Errorf("config: WEBHOOK_SECRET_KEY wajib diisi di production")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func parseDuration(key, fallback string) (time.Duration, error) {
	raw := getEnv(key, fallback)
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("config: %s tidak valid (%q): %w", key, raw, err)
	}
	return d, nil
}

func parseInt(key string, fallback int) (int, error) {
	raw := getEnv(key, strconv.Itoa(fallback))
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("config: %s tidak valid (%q): %w", key, raw, err)
	}
	return n, nil
}

func splitCSV(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
