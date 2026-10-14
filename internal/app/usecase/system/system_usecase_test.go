package system_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"platform-service/internal/app/dto"
	"platform-service/internal/app/usecase/system"
)

type fakeDB struct{ err error }

func (f fakeDB) Ping(context.Context) error { return f.err }

func TestSystemUseCase_Health(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name       string
		pingErr    error
		wantStatus string
	}{
		{name: "db sehat", pingErr: nil, wantStatus: dto.HealthStatusOK},
		{name: "db mati", pingErr: errors.New("connection refused"), wantStatus: dto.HealthStatusDegraded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := system.NewSystemUseCase(system.SystemDeps{DB: fakeDB{err: tt.pingErr}, DeploymentMode: "on_prem", Logger: logger})

			got := uc.Health(context.Background())

			assert.Equal(t, tt.wantStatus, got.Status)
			assert.Equal(t, tt.wantStatus, got.Database)
			assert.Equal(t, "on_prem", got.DeploymentMode)
		})
	}
}
