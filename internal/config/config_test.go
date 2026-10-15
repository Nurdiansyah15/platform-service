package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	t.Chdir(t.TempDir()) // tanpa .env
	t.Setenv("DB_PASS", "p@ss word")

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, config.DeploymentCloud, cfg.App.DeploymentMode)
	assert.Equal(t, "8080", cfg.App.Port)
	assert.Equal(t, []string{"*"}, cfg.CORS.AllowOrigins)
	assert.Equal(t, "postgres://platform:p%40ss%20word@localhost:5432/platform?sslmode=disable", cfg.Database.DSN())
}

func TestLoad_OnPrem(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("DEPLOYMENT_MODE", "on_prem")
	t.Setenv("CORS_ALLOW_ORIGINS", "https://a.example, https://b.example")

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, config.DeploymentOnPrem, cfg.App.DeploymentMode)
	assert.Equal(t, []string{"https://a.example", "https://b.example"}, cfg.CORS.AllowOrigins)
}

func TestLoad_InvalidDeploymentMode(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("DEPLOYMENT_MODE", "hybrid")

	_, err := config.Load()

	assert.ErrorContains(t, err, "DEPLOYMENT_MODE")
}

func TestLoad_InvalidDuration(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("ACCESS_TOKEN_TTL", "15 minutes")

	_, err := config.Load()

	assert.ErrorContains(t, err, "ACCESS_TOKEN_TTL")
}

func TestLoad_WebhookSecretKey(t *testing.T) {
	t.Chdir(t.TempDir())
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.NotEmpty(t, cfg.Webhook.SecretKey, "dev memakai kunci bawaan")
	assert.True(t, cfg.Webhook.AllowInsecureURL)

	t.Setenv("APP_ENV", "production")
	_, err = config.Load()
	assert.ErrorContains(t, err, "WEBHOOK_SECRET_KEY")

	t.Setenv("WEBHOOK_SECRET_KEY", "cHJvZC1rZXktcHJvZC1rZXktcHJvZC1rZXktMzJiISE=")
	cfg, err = config.Load()
	require.NoError(t, err)
	assert.False(t, cfg.Webhook.AllowInsecureURL, "production wajib https")
}
