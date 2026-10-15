package entity_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/domain/webhook/entity"
	"platform-service/internal/domain/webhook/service"
	"platform-service/internal/domain/webhook/valueobject"
)

func TestBackoff(t *testing.T) {
	assert.Equal(t, 30*time.Second, entity.Backoff(1))
	assert.Equal(t, 60*time.Second, entity.Backoff(2))
	assert.Equal(t, 8*time.Minute, entity.Backoff(5))
	assert.Equal(t, 6*time.Hour, entity.Backoff(20))
}

func TestDelivery_Evaluate(t *testing.T) {
	now := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)

	ok := (&entity.Delivery{}).Evaluate(204, nil, now)
	assert.Equal(t, valueobject.DeliverySuccess, ok.Status)
	require.NotNil(t, ok.DeliveredAt)

	failed := (&entity.Delivery{AttemptCount: 1}).Evaluate(500, nil, now)
	assert.Equal(t, valueobject.DeliveryFailed, failed.Status)
	assert.Equal(t, 2, failed.AttemptCount)
	assert.Equal(t, now.Add(time.Minute), *failed.NextAttemptAt)

	netErr := (&entity.Delivery{}).Evaluate(0, errors.New("connection refused"), now)
	assert.Equal(t, valueobject.DeliveryFailed, netErr.Status)
	assert.Nil(t, netErr.ResponseCode)
	assert.Equal(t, "connection refused", *netErr.Error)

	dead := (&entity.Delivery{AttemptCount: 7}).Evaluate(500, nil, now)
	assert.Equal(t, valueobject.DeliveryDead, dead.Status)
	assert.Nil(t, dead.NextAttemptAt)
}

func TestSignature(t *testing.T) {
	sig := service.Sign("s3cret", 1760000000, []byte(`{"a":1}`))
	assert.Regexp(t, `^sha256=[0-9a-f]{64}$`, sig)
	assert.True(t, service.Verify("s3cret", 1760000000, []byte(`{"a":1}`), sig))
	assert.False(t, service.Verify("s3cret", 1760000001, []byte(`{"a":1}`), sig))
	assert.False(t, service.Verify("lain", 1760000000, []byte(`{"a":1}`), sig))
}

func TestNewEndpointURLAndEventTypes(t *testing.T) {
	_, err := valueobject.NewEndpointURL("http://example.com/hook", false)
	assert.Error(t, err, "http ditolak di production")
	_, err = valueobject.NewEndpointURL("http://localhost:9000/hook", true)
	assert.NoError(t, err)
	_, err = valueobject.NewEndpointURL("ftp://example.com", true)
	assert.Error(t, err)
	_, err = valueobject.NewEndpointURL("/relative", true)
	assert.Error(t, err)

	types, err := valueobject.NewEventTypes([]string{"tenant.updated", "company.created", "tenant.updated"})
	require.NoError(t, err)
	assert.Equal(t, []string{"company.created", "tenant.updated"}, types)
	_, err = valueobject.NewEventTypes([]string{"tenant.exploded"})
	assert.Error(t, err)
	_, err = valueobject.NewEventTypes(nil)
	assert.Error(t, err)
}
