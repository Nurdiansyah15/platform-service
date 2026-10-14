package entity_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/organization/constant"
	"platform-service/internal/domain/organization/entity"
	"platform-service/internal/domain/organization/valueobject"
	"platform-service/internal/testhelper"
)

func TestNewCompany(t *testing.T) {
	c, err := entity.NewCompany("t1", entity.OrgUnitParams{Name: " Kantor Pusat ", Code: " hq-01 "}, time.Now())
	require.NoError(t, err)
	assert.Equal(t, "Kantor Pusat", c.Name)
	assert.Equal(t, "HQ-01", c.Code.Value())
	assert.Equal(t, valueobject.OrgStatusActive, c.Status)
}

func TestNewBranch_Invalid(t *testing.T) {
	tests := []struct {
		name string
		p    entity.OrgUnitParams
		want domainerr.Code
	}{
		{"name kosong", entity.OrgUnitParams{Name: " ", Code: "B1"}, constant.CodeOrgNameInvalid},
		{"code kosong", entity.OrgUnitParams{Name: "X", Code: ""}, constant.CodeOrgCodeInvalid},
		{"code spasi", entity.OrgUnitParams{Name: "X", Code: "B 1"}, constant.CodeOrgCodeInvalid},
		{"code diawali tanda", entity.OrgUnitParams{Name: "X", Code: "-B1"}, constant.CodeOrgCodeInvalid},
		{"status tak dikenal", entity.OrgUnitParams{Name: "X", Code: "B1", Status: "closed"}, constant.CodeOrgStatusInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := entity.NewBranch("c1", tt.p, time.Now())
			testhelper.AssertDomainError(t, err, tt.want)
		})
	}
}
